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
	"sync"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/util"
	stan "github.com/nats-io/go-nats-streaming"
)

// SyncWatchRevisionPrefix
const (
	SyncWatchRevisionPrefix = "/gohan/watch/revision"
)

// SyncWatcher runs extensions when it detects a change on the sync.
// The watcher implements a load balancing mechanism that uses
// entries on the sync.
type SyncWatcher struct {
	nats stan.Conn
	// list of key names to watch
	watchKeys []string
	// map from event names to VM environments
	watchExtensions map[string]extension.Environment
}

// NewSyncWatcher creates a new instance of syncWatcher
func NewSyncWatcher(nats stan.Conn, keys []string, extensions map[string]extension.Environment) *SyncWatcher {
	return &SyncWatcher{
		nats:            nats,
		watchKeys:       keys,
		watchExtensions: extensions,
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

	return NewSyncWatcher(server.nats, keys, extensions)
}

// Run starts the main loop of the watcher.
// This method blocks until the ctx is canceled by the caller
func (watcher *SyncWatcher) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	childrenWg := sync.WaitGroup{}
	watcher.runSyncWatches(ctx, &childrenWg)

	childrenWg.Wait()
	return ctx.Err()
}

// runSyncWatches starts goroutines to watch changes on the sync and run extensions for them.
// This method block until the context ctx is canceled and returns once all the goroutines are closed.
func (watcher *SyncWatcher) runSyncWatches(ctx context.Context, wg *sync.WaitGroup) {
	for _, path := range watcher.watchKeys {
		wg.Add(1)
		log.Debug("(SyncWatch) Priority of `%s` starting", path)

		pathWatcher := NewPathWatcher(watcher.nats, watcher.watchExtensions, path)

		go func(ctx context.Context, wg *sync.WaitGroup, pw *PathWatcher) {
			pw.Run(ctx, wg)
		}(ctx, wg, pathWatcher)
	}
}
