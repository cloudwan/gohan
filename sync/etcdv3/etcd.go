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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	syn "sync"
	"time"

	"github.com/cloudwan/gohan/sync"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	pb "github.com/coreos/etcd/mvcc/mvccpb"
	cmap "github.com/streamrail/concurrent-map"
	"github.com/twinj/uuid"
	"google.golang.org/grpc"
)

const (
	processPath = "/gohan/cluster/process"
	masterTTL   = 10
)

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

//GetProcessID returns processID
func (s *Sync) GetProcessID() string {
	return s.processID
}

//Update sync update sync
//When jsonString is empty, this method do nothing because
//etcd v3 doesn't support directories.
func (s *Sync) Update(key, jsonString string) error {
	var err error
	if jsonString == "" {
		// do nothing, because clientv3 doesn't have directories
		return nil
	}
	_, err = s.etcdClient.Put(s.withTimeout(), key, jsonString)
	if err != nil {
		log.Error(fmt.Sprintf("failed to sync with backend %s", err))
		return err
	}
	return nil
}

//Delete sync update sync
func (s *Sync) Delete(key string, prefix bool) error {
	opts := []etcd.OpOption{}
	if prefix {
		opts = append(opts, etcd.WithPrefix())
	}
	_, err := s.etcdClient.Delete(s.withTimeout(), key, opts...)
	return err
}

//Fetch data from sync
func (s *Sync) Fetch(key string) (*sync.Node, error) {
	node, err := s.etcdClient.Get(s.withTimeout(), key, etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
	if err != nil {
		return nil, err
	}
	dir, err := s.etcdClient.Get(s.withTimeout(), key, etcd.WithPrefix(), etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
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
			Key:      key,
			Value:    string(kv.Value),
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
func (s *Sync) Lock(path string, block bool) (chan struct{}, error) {
	for {
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

			if !block {
				return nil, errors.New(msg)
			}
			time.Sleep(masterTTL * time.Second)
			continue
		}
		s.locks.Set(path, lease.ID)
		log.Info("Locked %s", path)

		//Refresh master token
		lost := make(chan struct{})
		go func() {
			defer s.abortLock(path)
			defer close(lost)

			for s.HasLock(path) {
				resp, err := s.etcdClient.KeepAliveOnce(s.withTimeout(), lease.ID)
				if err != nil || resp.TTL <= 0 {
					log.Notice("failed to keepalive lock for %s %s", path, err)
					return
				}
				time.Sleep(masterTTL / 2 * time.Second)
			}
		}()

		return lost, nil
	}
}

func (s *Sync) abortLock(path string) etcd.LeaseID {
	leaseID, ok := s.locks.Get(path)
	if !ok {
		return 0
	}
	s.locks.Remove(path)
	log.Info("Unlocked path %s", path)
	return leaseID.(etcd.LeaseID)
}

//Unlock path
func (s *Sync) Unlock(path string) error {
	leaseID := s.abortLock(path)
	if leaseID > 0 {
		s.etcdClient.Revoke(s.withTimeout(), leaseID)

		cmp := etcd.Compare(etcd.Value(path), "=", s.processID)
		del := etcd.OpDelete(path)
		s.etcdClient.Txn(s.withTimeout()).If(cmp).Then(del).Commit()
	}
	return nil
}

func eventsFromNode(action string, kvs []*pb.KeyValue, responseChan chan *sync.Event) {
	for _, kv := range kvs {
		event := &sync.Event{
			Action:   action,
			Key:      string(kv.Key),
			Revision: kv.ModRevision,
		}
		if kv.Value != nil {
			err := json.Unmarshal(kv.Value, &event.Data)
			if err != nil {
				log.Warning("failed to unmarshal watch response value %s: %s", kv.Value, err)
			}
		}
		responseChan <- event
	}
}

//Watch keep watch update under the path
func (s *Sync) Watch(path string, responseChan chan *sync.Event, stopChan chan bool, revision int64) error {
	options := []etcd.OpOption{etcd.WithPrefix(), etcd.WithSort(etcd.SortByModRevision, etcd.SortAscend)}
	if revision != sync.RevisionCurrent {
		options = append(options, etcd.WithMinModRev(revision))
	}
	node, err := s.etcdClient.Get(s.withTimeout(), path, options...)
	if err != nil {
		return err
	}
	eventsFromNode("get", node.Kvs, responseChan)
	revision = node.Header.Revision + 1

	ctx, cancel := context.WithCancel(context.Background())
	errors := make(chan error, 1)
	var wg syn.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := func() error {
			rch := s.etcdClient.Watch(ctx, path, etcd.WithPrefix(), etcd.WithRev(revision))

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
	defer func() {
		cancel()
		wg.Wait()
	}()

	// since Watch() doesn't close the returning channel even when
	// it gets an error, we need a side channel to see the connection state.
	session, err := concurrency.NewSession(s.etcdClient, concurrency.WithTTL(masterTTL))
	if err != nil {
		return err
	}
	defer session.Close()

	select {
	case <-session.Done():
		return fmt.Errorf("Watch aborted by etcd session close")
	case <-stopChan:
		return nil
	case err := <-errors:
		return err
	}
}

// WatchContext keep watch update under the path until context is canceled
func (s *Sync) WatchContext(ctx context.Context, path string, revision int64) <-chan *sync.Event {
	eventCh := make(chan *sync.Event, 32)
	stopCh := make(chan bool)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Watch(path, eventCh, stopCh, revision)
	}()
	go func() {
		select {
		case <-ctx.Done():
			stopCh <- true
			err := <-errCh
			if err != nil && err != grpc.ErrClientConnClosing {
				log.Warning("Watch returned an unexpected error after context was cancelled: %v", err)
			}
		case err := <-errCh:
			if err != nil {
				eventCh <- &sync.Event{
					Err: err,
				}
			}
		}
		close(errCh)
		close(stopCh)
		close(eventCh)
	}()
	return eventCh
}

// Close closes etcd client
func (s *Sync) Close() {
	s.etcdClient.Close()
}
