// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package server

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cloudwan/gohan/extension"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

// SyncWatchRevisionPrefix
const (
	SyncWatchRevisionPrefix = "/gohan/watch/revision"
	processPathPrefix       = "/gohan/cluster/process"
	masterTTL               = 10
)

var errLockFailed = errors.New("failed to lock on sync backend")

// SyncWatcher runs extensions when it detects a change on the sync.
// The watcher implements a load balancing mechanism that uses
// entries on the sync.
type SyncWatcher struct {
	sync gohan_sync.Sync
	// list of key names to watch
	watchKeys []string
	// map from event names to VM environments
	watchExtensions map[string]extension.Environment
	backoff         time.Duration
}

// NewSyncWatcher creates a new instance of syncWatcher
func NewSyncWatcher(sync gohan_sync.Sync, keys []string, extensions map[string]extension.Environment) *SyncWatcher {
	return &SyncWatcher{
		sync:            sync,
		watchKeys:       keys,
		watchExtensions: extensions,
		backoff:         time.Second * 5,
	}
}

// NewSyncWatcherFromServer creates a new instance of syncWatcher from server
func NewSyncWatcherFromServer(server *Server) *SyncWatcher {
	config := util.GetConfig()
	keys := config.GetStringList("watch/keys", []string{})
	events := config.GetStringList("watch/events", []string{})
	extensions := make(map[string]extension.Environment, len(events))
	for _, event := range events {
		path := "sync://" + event
		env, err := server.NewEnvironmentForPath("sync."+event, path)
		if err != nil {
			log.Fatal(err.Error())
		}
		extensions[event] = env
	}

	return NewSyncWatcher(server.sync, keys, extensions)
}

// Run starts the main loop of the watcher.
// This method blocks until the ctx is canceled by the caller
func (watcher *SyncWatcher) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	for {
		err := func() error {
			// register self process to the cluster
			lockKey := processPathPrefix + "/" + watcher.sync.GetProcessID()
			lost, err := watcher.sync.Lock(ctx, lockKey, true)
			if err != nil {
				return err
			}
			defer func() {
				// can't use the parent context, it may be already canceled
				if err := watcher.sync.Unlock(context.Background(), lockKey); err != nil {
					log.Warning("SyncWatcher: unlocking etcd failed on %s: %s", lockKey, err)
				}
			}()

			watchCtx, watchCancel := context.WithCancel(ctx)
			defer watchCancel()
			events := watcher.sync.Watch(watchCtx, processPathPrefix, int64(gohan_sync.RevisionCurrent))
			watchErr := make(chan error, 1)
			go func() {
				watchErr <- watcher.processWatchLoop(events)
			}()

			select {
			case err := <-watchErr:
				return err
			case <-ctx.Done():
				return <-watchErr
			case <-lost:
				watchCancel()
				err := <-watchErr
				if err != nil {
					return fmt.Errorf("lock is lost: %s", err)
				}
				return fmt.Errorf("lock is lost")
			}
		}()

		if err != nil {
			log.Error("process watch interrupted: %s", err)
		}

		select {
		case <-time.After(watcher.backoff):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// processWatchLoop handles events from the watch on the process list.
// When this method detects a change, spawns new goroutines for sync event handling.
// This method blocks until the events channel is closed by the caller or
// an error event is given from the channel.
func (watcher *SyncWatcher) processWatchLoop(events <-chan *gohan_sync.Event) error {
	processList := []string{}

	previousCancel := func() {}
	previousDone := make(chan struct{})
	close(previousDone)
	defer func() {
		previousCancel()
		<-previousDone
	}()

	for event := range events {
		previousCancel()
		<-previousDone

		if event.Err != nil {
			return event.Err
		}

		log.Debug("cluster change detected: %s process %s", event.Action, event.Key)

		// modify gohan process list
		pos := -1
		for p, v := range processList {
			if v == event.Key {
				pos = p
			}
		}
		switch event.Action {
		case "delete":
			// remove detected process from list
			if pos > -1 {
				processList = append((processList)[:pos], (processList)[pos+1:]...)
			} else {
				log.Warning("unknown process was deleted from watch list: `%s`", event.Key)
			}
		default:
			// add detected process from list
			if pos == -1 {
				processList = append(processList, event.Key)
				sort.Sort(sort.StringSlice(processList))
			} else {
				log.Warning("process `%s` is already on the list", event.Key)
			}
		}

		myPosition := -1
		myValue := processPathPrefix + "/" + watcher.sync.GetProcessID()
		for p, v := range processList {
			if v == myValue {
				myPosition = p
				break
			}
		}

		if myPosition >= 0 && len(processList) > 0 {
			log.Debug("Current cluster consists of following processes: %s, my position: %d", processList, myPosition)

			var cctx context.Context
			cctx, previousCancel = context.WithCancel(context.Background())
			previousDone = make(chan struct{})

			go func() {
				defer close(previousDone)
				watcher.runSyncWatches(cctx, len(processList), myPosition)
			}()
		} else {
			log.Error("Current cluster consists of following processes: %s, my position not found: %d", processList, myPosition)
		}
	}

	return nil
}

// runSyncWatches starts goroutines to watch changes on the sync and run extensions for them.
// This method block until the context ctx is canceled and returns once all the goroutines are closed.
func (watcher *SyncWatcher) runSyncWatches(ctx context.Context, size int, position int) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for idx, path := range watcher.watchKeys {
		wg.Add(1)
		prio := (position - (idx % size) + size) % size
		log.Debug("(SyncWatch) Priority of `%s`: `%d`", path, prio)

		pathWatcher := NewPathWatcher(watcher.sync, watcher.watchExtensions, path, prio)

		go func(ctx context.Context, wg *sync.WaitGroup, pw *PathWatcher) {
			pw.Run(ctx, wg)
		}(ctx, &wg, pathWatcher)
	}
}
