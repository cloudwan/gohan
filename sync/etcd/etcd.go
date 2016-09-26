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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cloudwan/gohan/sync"
	"github.com/coreos/go-etcd/etcd"
	cmap "github.com/streamrail/concurrent-map"
	"github.com/twinj/uuid"
)

const masterTTL = 10

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

//Update sync update sync
func (s *Sync) Update(key, jsonString string) error {
	return s.UpdateTTL(key, jsonString, 0)
}

//UpdateTTL like Update, but allows to specify time to live in seconds
func (s *Sync) UpdateTTL(key, jsonString string, ttlSec uint64) error {
	var err error
	if jsonString == "" {
		_, err = s.etcdClient.SetDir(key, ttlSec)
	} else {
		_, err = s.etcdClient.Set(key, jsonString, ttlSec)
	}
	if err != nil {
		log.Error(fmt.Sprintf("failed to sync with backend %s", err))
		return err
	}
	return nil
}

//Delete sync update sync
func (s *Sync) Delete(key string) error {
	s.etcdClient.Delete(key, false)
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
		Key: node.Key,
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
	isLocked, _ := value.(bool)
	return isLocked && ok
}

// Lock locks resources on sync
// This call blocks until you can get lock
func (s *Sync) Lock(path string, block bool) error {
	for {
		if s.HasLock(path) {
			return nil
		}
		_, err := s.etcdClient.Create(path, s.processID, masterTTL)
		if err != nil {
			log.Notice("failed to lock path %s: %s", path, err)
			s.locks.Set(path, false)
			if !block {
				return err
			}
			time.Sleep(masterTTL * time.Second)
			continue
		}
		log.Info("Locked %s", path)
		s.locks.Set(path, true)
		//Refresh master token
		go func() {
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
		return nil
	}
}

//Unlock path
func (s *Sync) Unlock(path string) error {
	s.locks.Set(path, false)
	s.etcdClient.CompareAndDelete(path, s.processID, 0)
	log.Info("Unlocked path %s", path)
	return nil
}

func eventsFromNode(action string, node *etcd.Node, responseChan chan *sync.Event) {
	event := &sync.Event{
		Action: action,
		Key:    node.Key,
	}
	json.Unmarshal([]byte(node.Value), &event.Data)
	responseChan <- event
	for _, subnode := range node.Nodes {
		eventsFromNode(action, subnode, responseChan)
	}
}

//Watch keep watch update under the path
func (s *Sync) Watch(path string, responseChan chan *sync.Event, stopChan chan bool) error {
	var etcdResponseChan chan *etcd.Response

	etcdResponseChan = make(chan *etcd.Response)
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
	lastIndex := response.EtcdIndex + 1
	eventsFromNode(response.Action, response.Node, responseChan)
	go func() {
		_, err = s.etcdClient.Watch(path, lastIndex, true, etcdResponseChan, stopChan)
		if err != nil {
			log.Error(fmt.Sprintf("watch error: %s", err))
			stopChan <- true
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
		case stop := <-stopChan:
			if stop == true {
				return nil
			}
		}
	}
}
