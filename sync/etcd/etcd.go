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
	"github.com/twinj/uuid"
)

const masterTTL = 10

//Sync is struct for etcd based sync
type Sync struct {
	locks      map[string]bool
	etcdClient *etcd.Client
	processID  string
}

//NewSync initialize new etcd sync
func NewSync(etcdServers []string) *Sync {
	sync := &Sync{locks: map[string]bool{}}
	sync.etcdClient = etcd.NewClient(etcdServers)
	hostname, _ := os.Hostname()
	sync.processID = hostname + uuid.NewV4().String()
	return sync
}

//Update sync update sync
func (s *Sync) Update(key, jsonString string) error {
	_, err := s.etcdClient.Set(key, jsonString, 0)
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
func (s *Sync) Fetch(key string) (interface{}, error) {
	return s.etcdClient.Get(key, true, true)
}

//HasLock checks current process owns lock or not
func (s *Sync) HasLock(path string) bool {
	value, ok := s.locks[path]
	return value && ok
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
			s.locks[path] = false
			if !block {
				return err
			}
			time.Sleep(masterTTL * time.Second)
			continue
		}
		log.Info("Locked %s", path)
		s.locks[path] = true
		//Refresh master token
		go func() {
			for s.locks[path] {
				_, err := s.etcdClient.CompareAndSwap(
					path, s.processID, masterTTL, s.processID, 0)
				if err != nil {
					log.Notice("failed to keepalive lock for %s %s", path, err)
					s.locks[path] = false
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
	s.locks[path] = false
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
		log.Error(fmt.Sprintf("watch error: %s", err))
		return err
	}
	lastIndex := response.EtcdIndex + 1
	eventsFromNode(response.Action, response.Node, responseChan)
	go func() {
		_, err = s.etcdClient.Watch(path, lastIndex, true, etcdResponseChan, stopChan)
		if err != nil {
			log.Error(fmt.Sprintf("watch error: %s", err))
			return
		}
	}()

	for {
		select {
		case response := <-etcdResponseChan:
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
