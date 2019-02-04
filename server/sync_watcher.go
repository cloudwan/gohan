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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwan/gohan/extension"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/metrics"
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
	// list of event names
	watchEvents []string
	// map from event naems to VM environments
	watchExtensions map[string]extension.Environment
	backoff         time.Duration
}

// NewSyncWatcher creates a new instance of syncWatcher
func NewSyncWatcher(sync gohan_sync.Sync, keys []string, events []string, extensions map[string]extension.Environment) *SyncWatcher {
	return &SyncWatcher{
		sync:            sync,
		watchKeys:       keys,
		watchEvents:     events,
		watchExtensions: extensions,
		backoff:         time.Second * 5,
	}
}

// NewSyncWatcherFromServer creates a new instance of syncWatcher from server
func NewSyncWatcherFromServer(server *Server) *SyncWatcher {
	config := util.GetConfig()
	keys := config.GetStringList("watch/keys", []string{})
	events := config.GetStringList("watch/events", []string{})
	extensions := map[string]extension.Environment{}
	for _, event := range events {
		path := "sync://" + event
		env, err := server.NewEnvironmentForPath("sync."+event, path)
		if err != nil {
			log.Fatal(err.Error())
		}
		extensions[event] = env
	}

	return NewSyncWatcher(server.sync, keys, events, extensions)
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

		go func(ctx context.Context, idx int, path string, prio int) {
			defer wg.Done()

			select {
			case <-time.After(time.Duration(prio*masterTTL) * time.Second):
			case <-ctx.Done():
				return
			}

			for {
				err := watcher.processSyncWatch(ctx, path)

				switch err {
				case errLockFailed:
					watcher.updateCounter(1, "lock.failed")
					log.Debug("(SyncWatcher) failed to acquire lock, retrying...")
				case context.Canceled:
					// Do nothing, normal shutdown
				default:
					watcher.updateCounter(1, "error")
					log.Error("(SyncWatcher) on `%s` aborted, retrying...: %s", path, err)
				}

				select {
				case <-time.After(time.Duration(prio*masterTTL+1) * time.Second):
				case <-ctx.Done():
					return
				}
			}
		}(ctx, idx, path, prio)
	}
}

// processSyncWatch handles events on a path with a handler.
// Returns any error or context cancel.
// This method gets a lock on the sync backend and returns with an error when fails.
func (watcher *SyncWatcher) processSyncWatch(ctx context.Context, path string) error {
	watcher.updateCounter(1, "active")
	defer watcher.updateCounter(-1, "active")

	lockKey := lockPath + "/watch" + path
	lost, err := watcher.sync.Lock(ctx, lockKey, false)
	if err != nil {
		return errLockFailed
	}
	defer func() {
		// can't use the parent context, it may be already canceled
		if err := watcher.sync.Unlock(context.Background(), lockKey); err != nil {
			log.Warning("SyncWatcher: unlocking etcd failed on %s: %s", lockKey, err)
		}
	}()

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()
	fromRevision := watcher.fetchStoredRevision(ctx, path) + 1
	respCh := watcher.sync.Watch(watchCtx, path, fromRevision)
	watchErr := make(chan error, 1)
	go func() {
		watcher.updateCounter(1, "running")
		defer watcher.updateCounter(-1, "running")
		watchErr <- func() error {
			for response := range respCh {
				if response.Err != nil {
					return response.Err
				}
				watcher.watchExtensionHandler(response)

				err := watcher.storeRevision(ctx, path, response.Revision)
				if err != nil {
					return err
				}
			}
			return nil
		}()
	}()

	select {
	case <-ctx.Done():
		if err := <-watchErr; err != nil {
			log.Error("(SyncWatcher) error after done: %s", err)
		}
		return ctx.Err()
	case <-lost:
		watcher.updateCounter(1, "lock.lost")
		watchCancel()
		if err := <-watchErr; err != nil {
			log.Error("(SyncWatcher) error after lost lock: %s", err)
		}
		return fmt.Errorf("lock for path `%s` is lost", path)
	case err := <-watchErr:
		return err
	}
}

func (watcher *SyncWatcher) watchExtensionHandler(response *gohan_sync.Event) {
	defer l.Panic(log)
	for _, event := range watcher.watchEvents {
		//match extensions
		if strings.HasPrefix(response.Key, "/"+event) {
			env := watcher.watchExtensions[event]
			watcher.runExtensionOnSync(response, env.Clone())
			return
		}
	}
}

// fetchStoredRevision returns the revision number stored in the sync backend for a path.
// When it's a new in the backend, returns sync.RevisionCurrent.
func (watcher *SyncWatcher) fetchStoredRevision(ctx context.Context, path string) int64 {
	fromRevision := int64(gohan_sync.RevisionCurrent)
	lastSeen, err := watcher.sync.Fetch(ctx, SyncWatchRevisionPrefix+path)
	if err == nil {
		inStore, err := strconv.ParseInt(lastSeen.Value, 10, 64)
		if err == nil {
			log.Info("(SyncWatcher) Using last seen revision `%d` for watching path `%s`", inStore, path)
			fromRevision = inStore
		} else {
			log.Warning("(SyncWatcher) Revision `%s` is not a valid int64 number, using the current one, which is %d (%s)", lastSeen.Value, fromRevision, err)
		}
	} else {
		log.Warning("(SyncWatcher) Failed to fetch last seen revision number, using the current one, which is %d: (%s)", fromRevision, err)
	}
	return fromRevision
}

// storeRevision puts a revision number for a path to the sync backend.
func (watcher *SyncWatcher) storeRevision(ctx context.Context, path string, revision int64) error {
	err := watcher.sync.Update(ctx, SyncWatchRevisionPrefix+path, strconv.FormatInt(revision, 10))
	if err != nil {
		return fmt.Errorf("Failed to update revision number for watch path `%s` in sync storage", path)
	}
	return nil
}

func (watcher *SyncWatcher) measureSyncTime(timeStarted time.Time, action string) {
	metrics.UpdateTimer(timeStarted, "sync.%s", action)
}

//Run extension on sync
func (watcher *SyncWatcher) runExtensionOnSync(response *gohan_sync.Event, env extension.Environment) {
	defer watcher.measureSyncTime(time.Now(), response.Action)

	context := map[string]interface{}{
		"action":   response.Action,
		"data":     response.Data,
		"key":      response.Key,
		"context":  context.TODO(), // change to proper context after SyncWatcher refactoring towards contexts
		"trace_id": util.NewTraceID(),
	}
	if err := env.HandleEvent("notification", context); err != nil {
		log.Warning(fmt.Sprintf("(SyncWatcher) extension error: %s", err))
		return
	}
	return
}

func (watcher *SyncWatcher) updateCounter(delta int64, metric string) {
	metrics.UpdateCounter(delta, "sync_watcher.%s", metric)
}
