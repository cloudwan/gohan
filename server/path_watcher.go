// Copyright (C) 2019 NTT Innovation Institute, Inc.
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
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/metrics"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
	"github.com/pkg/errors"
)

type PathWatcher struct {
	sync                      gohan_sync.Sync
	priority                  int
	path                      string
	escapedPath               string
	extensions                map[string]extension.Environment
	previousProcessedRevision int64
}

var (
	errInconsistentCluster = errors.New("inconsistent cluster state detected")
	replacer               = strings.NewReplacer(".", "_", "/", "_")
)

func NewPathWatcher(sync gohan_sync.Sync, extensions map[string]extension.Environment, path string, priority int) *PathWatcher {
	return &PathWatcher{
		sync:        sync,
		extensions:  extensions,
		priority:    priority,
		path:        path,
		escapedPath: replacer.Replace(path),
	}
}

func (watcher PathWatcher) String() string {
	sb := strings.Builder{}
	sb.WriteString("(PathWatcher ")
	sb.WriteString(watcher.path)
	sb.WriteString(")")
	return sb.String()
}

func (watcher *PathWatcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-time.After(time.Duration(watcher.priority*masterTTL) * time.Second):
	case <-ctx.Done():
		return
	}

	for {
		err := watcher.run(ctx)

		switch err {
		case errLockFailed:
			watcher.updateCounter(1, "lock.failed")
			log.Debug("%s failed to acquire lock, retrying...", watcher)
		case context.Canceled:
			// Do nothing, normal shutdown
		default:
			watcher.updateCounter(1, "error")
			log.Error("%s aborted, retrying...: %s", watcher, err)
		}

		select {
		case <-time.After(time.Duration(watcher.priority*masterTTL+1) * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

// run handles events on a path with a handler.
// Returns any error or context cancel.
// This method gets a lock on the sync backend and returns with an error when fails.
func (watcher *PathWatcher) run(parentCtx context.Context) error {
	watcher.updateCounter(1, "active")
	defer watcher.updateCounter(-1, "active")

	lockKey := lockPath + "/watch" + watcher.path
	lost, err := watcher.sync.Lock(parentCtx, lockKey, false)
	if err != nil {
		return errLockFailed
	}
	defer func() {
		// can't use the parent context, it may be already canceled
		if err := watcher.sync.Unlock(context.Background(), lockKey); err != nil {
			log.Warning("%s unlocking etcd failed on %s: %s", watcher, lockKey, err)
		}
	}()

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	watcher.previousProcessedRevision = watcher.fetchStoredRevision(parentCtx)
	eventsCh := watcher.sync.Watch(ctx, watcher.path, watcher.previousProcessedRevision+1)
	doneCh := make(chan error, 1)

	go watcher.consumeEvents(ctx, eventsCh, doneCh)

	select {
	case <-ctx.Done():
		if err := <-doneCh; err != nil {
			log.Error("%s consuming events failed: %s", watcher, err)
		}
		return ctx.Err()
	case <-lost:
		watcher.updateCounter(1, "lock.lost")
		cancel()
		if err := <-doneCh; err != nil {
			log.Error("%s error after lost lock: %s", watcher, err)
		}
		return fmt.Errorf("lock for path `%s` is lost", watcher.path)
	case err := <-doneCh:
		return err
	}
}

func (watcher *PathWatcher) consumeEvents(ctx context.Context, eventCh <-chan *gohan_sync.Event, watchErr chan<- error) {
	watcher.updateCounter(1, "running")
	defer watcher.updateCounter(-1, "running")

	var err error
	defer func() {
		watchErr <- err
	}()

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				err = ctx.Err()
				return
			}
			if err = watcher.consumeEvent(event); err != nil {
				return
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

func (watcher *PathWatcher) consumeEvent(event *gohan_sync.Event) (err error) {
	// Each event & recovery must not be interrupted,
	// otherwise we could easily end up in an inconsistent state
	// if ctx is canceled either in running extensions or just before recovery.
	// There's a potential defect around it: if any of the called methods block,
	// we'll wait forever. At the moment of writing, the codebase is not affected -
	// we've got calls only to ETCD and extensions, both protected by timeouts.
	ctx := context.Background()

	defer func() {
		watcher.tryRecover(ctx, err, event)
	}()

	if event.Err != nil {
		return event.Err
	}
	watcher.watchExtensionHandler(ctx, event)

	return watcher.storeRevision(ctx, event.Revision)
}

func (watcher *PathWatcher) tryRecover(ctx context.Context, err error, event *gohan_sync.Event) {
	if err == nil {
		return
	}

	if err == errInconsistentCluster {
		watcher.tryRecoverInconsistentCluster(ctx, event)
	} else if errCompacted, ok := err.(goext.ErrCompacted); ok {
		watcher.tryRecoverCompaction(ctx, event, errCompacted.CompactRevision)
	} else {
		log.Critical("%s can't recover from '%s' on '%s'", watcher, err, event.Key)
	}
}

func (watcher *PathWatcher) tryRecoverInconsistentCluster(ctx context.Context, event *gohan_sync.Event) {
	node, err := watcher.sync.Fetch(ctx, event.Key)
	if err != nil {
		watcher.updateCounter(1, "inconsistency_recovery.fetch_errors")
		log.Critical(
			"%s can't recover from inconsistent cluster state on %s: Fetch() failed: %s",
			watcher, event.Key, err,
		)
		return
	}

	if recovered, err := watcher.sync.CompareAndSwap(ctx, event.Key, node.Value, watcher.sync.ByValue(node.Value)); err != nil {
		// some sync events could have be processed out-of-order and the recovery failed.
		// incorrect data could be stored in DB. a user has to recover (resync) manually
		watcher.updateCounter(1, "inconsistency_recovery.cas_errors")
		log.Critical(
			"%s can't recover from inconsistent cluster state on %s: notifying the current master failed: %s",
			watcher, event.Key, err,
		)
	} else if recovered {
		watcher.updateCounter(1, "inconsistency_recovery.success")
		log.Info("%s successfully recovered from inconsistency on %s", watcher, event.Key)
	} else {
		watcher.updateCounter(1, "inconsistency_recovery.not_needed")
		log.Info(
			"%s inconsistency on %s detected, but further events are already scheduled, the cluster will recover itself soon",
			watcher, event.Key,
		)
	}
}

func (watcher *PathWatcher) tryRecoverCompaction(ctx context.Context, event *gohan_sync.Event, compactedRevision int64) {
	// next watch should start at lastProcessed +1, so we're setting a known-good -1
	if err := watcher.storeRevision(ctx, compactedRevision-1); err != nil && err != errInconsistentCluster {
		// it's not fatal: the next leader will with again fail with errCompacted and retry the recovery
		watcher.updateCounter(1, "compaction_recovery.failed")
		log.Warning("%s can't recover from etcd compaction on %s: %s", watcher, event.Key, err)
	} else if err == errInconsistentCluster {
		watcher.updateCounter(1, "compaction_recovery.not_needed")
	} else {
		watcher.updateCounter(1, "compaction_recovery.success")
	}
}

func (watcher *PathWatcher) watchExtensionHandler(ctx context.Context, response *gohan_sync.Event) {
	defer watcher.recoverPanic()

	for event, env := range watcher.extensions {
		if strings.HasPrefix(response.Key, "/"+event) {
			watcher.runExtensionOnSync(ctx, response, env.Clone())
			return
		}
	}
}

func (watcher *PathWatcher) recoverPanic() {
	err := recover()
	if err != nil {
		log.Error("%s panicked: %s: %s", watcher, err, debug.Stack())
	}
}

// fetchStoredRevision returns the revision number stored in the sync backend for a path.
// When it's a new in the backend, returns sync.RevisionCurrent.
func (watcher *PathWatcher) fetchStoredRevision(ctx context.Context) int64 {
	fromRevision := goext.RevisionCurrent
	lastSeen, err := watcher.sync.Fetch(ctx, SyncWatchRevisionPrefix+watcher.path)
	if err == nil {
		inStore, err := strconv.ParseInt(lastSeen.Value, 10, 64)
		if err == nil {
			log.Info("%s using last seen revision `%d`", watcher, inStore)
			fromRevision = inStore
		} else {
			log.Warning(
				"%s revision `%s` is not a valid int64 number, using the current one, which is %d (%s)",
				watcher, lastSeen.Value, fromRevision, err,
			)
		}
	} else {
		log.Warning(
			"%s failed to fetch last seen revision number, using the current one, which is %d: (%s)",
			watcher, fromRevision, err,
		)
	}
	return fromRevision
}

// storeRevision puts a revision number for a path to the sync backend.
func (watcher *PathWatcher) storeRevision(ctx context.Context, revision int64) error {
	path := SyncWatchRevisionPrefix + watcher.path
	value := strconv.FormatInt(revision, 10)
	opts := make([]gohan_sync.CASCondition, 0, 1)
	if watcher.previousProcessedRevision != goext.RevisionCurrent {
		opts = append(opts, watcher.sync.ByValue(strconv.FormatInt(watcher.previousProcessedRevision, 10)))
	}

	if swapped, err := watcher.sync.CompareAndSwap(ctx, path, value, opts...); err != nil {
		return fmt.Errorf("failed to update revision number for watch path `%s` in sync storage", watcher.path)
	} else if !swapped {
		watcher.updateCounter(1, "inconsistency_detected")
		log.Warning(
			"%s cluster inconsistency detected: other node in the cluster already made progress, broken revision is %d",
			watcher, revision,
		)
		return errInconsistentCluster
	}

	watcher.updateGauge(revision, "previous_processed_revision")
	watcher.previousProcessedRevision = revision
	return nil
}

//Run extension on sync
func (watcher *PathWatcher) runExtensionOnSync(ctx context.Context, response *gohan_sync.Event, env extension.Environment) {
	defer watcher.measureTime(time.Now(), response.Action)

	context := map[string]interface{}{
		"action":   response.Action,
		"data":     response.Data,
		"key":      response.Key,
		"context":  ctx,
		"trace_id": util.NewTraceID(),
	}
	if err := env.HandleEvent("notification", context); err != nil {
		log.Error("%s extension error, last processed event may be lost: %s", watcher, err)
	}
}

func (watcher *PathWatcher) measureTime(timeStarted time.Time, action string) {
	metrics.UpdateTimer(timeStarted, "path_watcher.%s.%s", watcher.escapedPath, action)
}

func (watcher *PathWatcher) updateCounter(delta int64, metric string) {
	metrics.UpdateCounter(delta, "path_watcher.%s.%s", watcher.escapedPath, metric)
}

func (watcher *PathWatcher) updateGauge(value int64, metric string) {
	metrics.UpdateGauge(value, "path_watcher.%s.%s", watcher.escapedPath, metric)
}
