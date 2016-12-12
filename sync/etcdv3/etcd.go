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

package etcdv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	syn "sync"
	"time"

	"github.com/cloudwan/gohan/sync"
	etcd "github.com/coreos/etcd/clientv3"
	pb "github.com/coreos/etcd/mvcc/mvccpb"
	cmap "github.com/streamrail/concurrent-map"
	"github.com/twinj/uuid"
	"golang.org/x/net/context"
)

const masterTTL = 10

//Sync is struct for etcd based sync
type Sync struct {
	locks      cmap.ConcurrentMap
	timeout    time.Duration
	etcdClient *etcd.Client
	processID  string
}

func (s *Sync) withTimeout() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), s.timeout)
	return ctx
}

//NewSync initialize new etcd sync
func NewSync(etcdServers []string, timeout time.Duration) (*Sync, error) {
	sync := &Sync{
		locks:   cmap.New(),
		timeout: timeout,
	}
	client, err := etcd.New(
		etcd.Config{
			Endpoints:   etcdServers,
			DialTimeout: timeout,
		},
	)
	if err != nil {
		return nil, err
	}
	sync.etcdClient = client
	hostname, _ := os.Hostname()
	sync.processID = hostname + uuid.NewV4().String()
	return sync, nil
}

//Update sync update sync
//When jsonString is empty, this method do nothing because
//etcd v3 doesn't support directories.
func (s *Sync) Update(key, jsonString string) error {
	var err error
	if jsonString == "" {
		// do nothing, because clientv3 doesn't have directories
		return nil
	} else {
		_, err = s.etcdClient.Put(s.withTimeout(), key, jsonString)
	}
	if err != nil {
		log.Error(fmt.Sprintf("failed to sync with backend %s", err))
		return err
	}
	return nil
}

//Delete sync update sync
func (s *Sync) Delete(key string) error {
	_, err := s.etcdClient.Delete(s.withTimeout(), key)
	return err
}

//Fetch data from sync
func (s *Sync) Fetch(key string) (*sync.Node, error) {
	node, err := s.etcdClient.Get(s.withTimeout(), key, etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
	if err != nil {
		return nil, err
	}
	dir, err := s.etcdClient.Get(s.withTimeout(), key+"/", etcd.WithPrefix(), etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
	if err != nil {
		return nil, err
	}

	return s.recursiveFetch(key, node.Kvs, dir.Kvs)
}

func (s *Sync) recursiveFetch(rootKey string, node []*pb.KeyValue, children []*pb.KeyValue) (*sync.Node, error) {
	if len(node) == 0 && len(children) == 0 {
		return nil, errors.New("Not found")
	}

	subMap := make(map[string]*sync.Node, len(children))

	rootNode := &sync.Node{}
	subMap[""] = rootNode
	if len(node) != 0 {
		rootNode.Key = string(node[0].Key)
		rootNode.Value = string(node[0].Value)
		rootNode.Revision = node[0].ModRevision
	} else {
		rootNode.Key = rootKey
	}

	for _, kv := range children {
		key := string(kv.Key)
		n := &sync.Node{
			Key: key,
			Value: string(kv.Value),
			Revision: kv.ModRevision,
		}
		path := strings.TrimPrefix(key, rootKey)
		steps := strings.Split(path, "/")

		for i := 1; i < len(steps); i++ {
			parent, ok := subMap[strings.Join(steps[:len(steps)-i], "/")]
			if ok {
				for j := 1; j < i; j++ {
					bridge := &sync.Node{
						Key: rootKey + strings.Join(steps[:len(steps)-i+j], "/"),
					}
					parent.Children = []*sync.Node{bridge}
					parent = bridge
				}
				if parent.Children == nil {
					parent.Children = make([]*sync.Node, 0, 1)
				}
				parent.Children = append(parent.Children, n)
				break
			}
		}
	}

	return rootNode, nil
}

//HasLock checks current process owns lock or not
func (s *Sync) HasLock(path string) bool {
	return s.locks.Has(path)
}

// Lock locks resources on sync
// This call blocks until you can get lock
func (s *Sync) Lock(path string, block bool) error {
	for {
		if s.HasLock(path) {
			return nil
		}
		var err error
		lease, err := s.etcdClient.Grant(s.withTimeout(), masterTTL)
		var resp *etcd.TxnResponse
		if err == nil {
			cmp := etcd.Compare(etcd.CreateRevision(path), "=", 0)
			put := etcd.OpPut(path, s.processID, etcd.WithLease(lease.ID))
			resp, err = s.etcdClient.Txn(s.withTimeout()).If(cmp).Then(put).Commit()
		}
		if err != nil || !resp.Succeeded {
			msg := fmt.Sprintf("failed to lock path %s", path)
			if err != nil {
				msg = fmt.Sprintf("failed to lock path %s: %s", path, err)
			}
			log.Notice(msg)

			s.locks.Remove(path)
			if !block {
				return errors.New(msg)
			}
			time.Sleep(masterTTL * time.Second)
			continue
		}
		log.Info("Locked %s", path)
		s.locks.Set(path, lease.ID)
		//Refresh master token
		go func() {
			defer func() {
				log.Notice("releasing keepalive lock for %s", path)
				s.locks.Remove(path)
			}()
			for s.HasLock(path) {
				ch, err := s.etcdClient.KeepAlive(s.withTimeout(), lease.ID)
				if err != nil {
					log.Notice("failed to keepalive lock for %s %s", path, err)
					return
				}
				for range ch {
				}
			}
		}()

		return nil
	}
}

//Unlock path
func (s *Sync) Unlock(path string) error {
	leaseID, ok := s.locks.Get(path)
	if !ok {
		return nil
	}
	s.locks.Remove(path)
	s.etcdClient.Revoke(s.withTimeout(), leaseID.(etcd.LeaseID))
	log.Info("Unlocked path %s", path)
	return nil
}

func eventsFromNode(action string, kvs []*pb.KeyValue, responseChan chan *sync.Event) {
	for _, kv := range kvs {
		event := &sync.Event{
			Action: action,
			Key:    string(kv.Key),
		}
		if kv.Value != nil {
			err := json.Unmarshal(kv.Value, &event.Data)
			if err != nil {
				log.Warning("failed to unmarshal watch response: %s", err)
				continue
			}
		}
		responseChan <- event
	}
}

//Watch keep watch update under the path
func (s *Sync) Watch(path string, responseChan chan *sync.Event, stopChan chan bool, revision int64) error {
	if revision == sync.RevisionCurrent {
		node, err := s.etcdClient.Get(s.withTimeout(), path, etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
		if err != nil {
			return err
		}
		eventsFromNode("get", node.Kvs, responseChan)

		revision = node.Header.Revision + 1
	}

	dir, err := s.etcdClient.Get(s.withTimeout(), path+"/", etcd.WithPrefix(), etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
	if err != nil {
		return err
	}
	eventsFromNode("get", dir.Kvs, responseChan)

	ctx, cancel := context.WithCancel(context.Background())
	errors := make(chan error, 2)
	var wg syn.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := func() error {
			rch := s.etcdClient.Watch(ctx, path, etcd.WithRev(revision))

			for wresp := range rch {
				err := wresp.Err()
				if err != nil {
					return err
				}
				for _, ev := range wresp.Events {
					action := "unknown"
					switch ev.Type {
					case etcd.EventTypePut:
						action = "set"
					case etcd.EventTypeDelete:
						action = "delete"
					}
					eventsFromNode(action, []*pb.KeyValue{ev.Kv}, responseChan)
				}
			}

			return nil
		}()
		errors <- err
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := func() error {
			rch := s.etcdClient.Watch(ctx, path+"/", etcd.WithPrefix(), etcd.WithRev(revision))

			for wresp := range rch {
				err := wresp.Err()
				if err != nil {
					return err
				}
				for _, ev := range wresp.Events {
					action := "unknown"
					switch ev.Type {
					case etcd.EventTypePut:
						action = "set"
					case etcd.EventTypeDelete:
						action = "delete"
					}
					eventsFromNode(action, []*pb.KeyValue{ev.Kv}, responseChan)
				}
			}

			return nil
		}()
		errors <- err
	}()

	select {
	case <-stopChan:
		cancel()
		wg.Wait()
		return nil
	case err := <-errors:
		cancel()
		wg.Wait()
		return err
	}
}

func (s *Sync) Close()  {
	s.etcdClient.Close()
}