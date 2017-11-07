// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cloudwan/gohan/sync"
	"github.com/coreos/go-etcd/etcd"
	cmap "github.com/streamrail/concurrent-map"
	"github.com/twinj/uuid"
)

const (
	processPath = "/gohan/cluster/process"
	masterTTL   = 10
)

//Sync is struct for etcd based sync
type Sync struct {
	locks      cmap.ConcurrentMap
	etcdClient *etcd.Client
	processID  string
}

//NewSync initialize new etcd sync
func NewSync(etcdServers []string) *Sync {
	sync := &Sync{locks: cmap.New()}
	sync.etcdClient = etcd.NewClient(etcdServers)
	hostname, _ := os.Hostname()
	sync.processID = hostname + uuid.NewV4().String()
	return sync
}

//GetProcessID returns processID
func (s *Sync) GetProcessID() string {
	return s.processID
}

//Update sync update sync
func (s *Sync) Update(key, jsonString string) error {
	var err error
	if jsonString == "" {
		_, err = s.etcdClient.SetDir(key, 0)
	} else {
		_, err = s.etcdClient.Set(key, jsonString, 0)
	}
	if err != nil {
		log.Error(fmt.Sprintf("failed to sync with backend %s", err))
		return err
	}
	return nil
}

//Delete sync update sync
func (s *Sync) Delete(key string, prefix bool) error {
	if prefix {
		log.Warning("Prefix option is translated as Recursive for Etcd v2 compatibility.")
	}
	s.etcdClient.Delete(key, prefix)
	return nil
}

//Fetch data from sync
func (s *Sync) Fetch(key string) (*sync.Node, error) {
	resp, err := s.etcdClient.Get(key, true, true)
	if err != nil {
		return nil, err
	}

	return s.recursiveFetch(resp.Node)
}

func (s *Sync) recursiveFetch(node *etcd.Node) (*sync.Node, error) {
	children := make([]*sync.Node, 0, len(node.Nodes))
	for _, child := range node.Nodes {
		var err error
		childNodes, err := s.recursiveFetch(child)
		if err != nil {
			return nil, err
		}
		children = append(children, childNodes)
	}

	n := &sync.Node{
		Key:      node.Key,
		Revision: int64(node.ModifiedIndex),
	}
	if node.Dir {
		n.Children = children
	} else {
		n.Value = node.Value
	}

	return n, nil
}

//HasLock checks current process owns lock or not
func (s *Sync) HasLock(path string) bool {
	value, ok := s.locks.Get(path)
	if !ok {
		return false
	}
	isLocked, _ := value.(bool)
	return isLocked
}

// Lock locks resources on sync
// This call blocks until you can get lock
func (s *Sync) Lock(path string, block bool) (chan struct{}, error) {
	for {
		_, err := s.etcdClient.Create(path, s.processID, masterTTL)
		if err != nil {
			log.Notice("failed to lock path %s: %s", path, err)
			s.locks.Set(path, false)
			if !block {
				return nil, err
			}
			time.Sleep(masterTTL * time.Second)
			continue
		}
		s.locks.Set(path, true)
		log.Info("Locked %s", path)

		//Refresh master token
		lost := make(chan struct{})
		go func() {
			defer s.abortLock(path)
			defer close(lost)

			for s.HasLock(path) {
				_, err := s.etcdClient.CompareAndSwap(
					path, s.processID, masterTTL, s.processID, 0)
				if err != nil {
					log.Notice("failed to keepalive lock for %s %s", path, err)
					s.locks.Set(path, false)
					return
				}
				time.Sleep(masterTTL / 2 * time.Second)
			}
		}()
		return lost, nil
	}
}

func (s *Sync) abortLock(path string) {
	s.locks.Set(path, false)
}

//Unlock path
func (s *Sync) Unlock(path string) error {
	s.abortLock(path)
	s.etcdClient.CompareAndDelete(path, s.processID, 0)
	log.Info("Unlocked path %s", path)
	return nil
}

func eventsFromNode(action string, node *etcd.Node, responseChan chan *sync.Event) {
	event := &sync.Event{
		Action:   action,
		Key:      node.Key,
		Revision: int64(node.ModifiedIndex),
	}
	json.Unmarshal([]byte(node.Value), &event.Data)
	responseChan <- event
	for _, subnode := range node.Nodes {
		eventsFromNode(action, subnode, responseChan)
	}
}

// Watch keep watch update under the path
// revision is not suported by this implementation
func (s *Sync) Watch(path string, responseChan chan *sync.Event, stopChan chan bool, revision int64) error {
	etcdResponseChan := make(chan *etcd.Response)
	response, err := s.etcdClient.Get(path, true, true)
	if err != nil {
		if etcdError, ok := err.(*etcd.EtcdError); ok {
			switch etcdError.ErrorCode {
			case 100:
				response, err = s.etcdClient.CreateDir(path, 0)
				if err != nil {
					log.Error(fmt.Sprintf("failed to create dir: %s", err))
					return err
				}
			default:
				log.Error(fmt.Sprintf("etcd error[%d]: %s ", etcdError.ErrorCode, etcdError))
				return err
			}
		} else {
			log.Error(fmt.Sprintf("watch error: %s", err))
			return err
		}
	}
	var lastIndex uint64
	lastIndex = response.EtcdIndex + 1
	eventsFromNode(response.Action, response.Node, responseChan)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err = s.etcdClient.Watch(path, lastIndex, true, etcdResponseChan, stopChan)
		if err != nil {
			log.Error(fmt.Sprintf("watch error: %s", err))
			return
		}
	}()

	for {
		select {
		case response, ok := <-etcdResponseChan:
			if !ok {
				return nil
			}
			if response != nil {
				event := &sync.Event{
					Action: response.Action,
					Key:    response.Node.Key,
				}
				json.Unmarshal([]byte(response.Node.Value), &event.Data)
				responseChan <- event
			}
		case <-done:
			return nil
		}
	}
}

// WatchContext keep watch update under the path until context is canceled
func (s *Sync) WatchContext(ctx context.Context, path string, revision int64) <-chan *sync.Event {
	stopChan := make(chan bool)
	go func() {
		<-ctx.Done()
		close(stopChan)
	}()

	responseChan := make(chan *sync.Event)

	go func() {
		defer close(responseChan)
		err := s.Watch(path, responseChan, stopChan, revision)
		if err != nil {
			responseChan <- &sync.Event{
				Err: err,
			}
		}
	}()

	return responseChan
}

// Close closes sync
func (s *Sync) Close() {
	// nothing to do
}
