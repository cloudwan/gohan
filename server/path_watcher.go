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
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/metrics"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
	stan "github.com/nats-io/go-nats-streaming"
)

type PathWatcher struct {
	nats        stan.Conn
	path        string
	escapedPath string
	extensions  map[string]extension.Environment
}

var (
	replacer = strings.NewReplacer(".", "_", "/", "_")
)

func NewPathWatcher(nats stan.Conn, extensions map[string]extension.Environment, path string) *PathWatcher {
	return &PathWatcher{
		nats:        nats,
		extensions:  extensions,
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

	for {
		err := watcher.run(ctx)

		switch err {
		case context.Canceled:
			// Do nothing, normal shutdown
		default:
			watcher.updateCounter(1, "error")
			log.Error("%s aborted, retrying...: %s", watcher, err)
		}

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
	}
}

// run handles events on a path with a handler.
// Returns any error or context cancel.
// This method gets a lock on the sync backend and returns with an error when fails.
func (watcher *PathWatcher) run(ctx context.Context) error {
	watcher.updateCounter(1, "active")
	defer watcher.updateCounter(-1, "active")

	subscription, err := watcher.nats.QueueSubscribe(watcher.path, "gohan-group", func(msg *stan.Msg) {
		watcher.watchExtensionHandler(msg)

		if ackErr := msg.Ack(); ackErr != nil {
			log.Error("%s ACKing failed: %s", watcher, ackErr)
		}

		watcher.storeRevision(msg.Sequence)

	}, stan.DurableName("durable-gohan-queue"), stan.SetManualAckMode(), stan.DeliverAllAvailable())

	if err != nil {
		return fmt.Errorf("subscription failed: %s", err)
	}

	<-ctx.Done()

	if err := subscription.Close(); err != nil {
		log.Error("%s closing subscription failed: %s", watcher, err)
	}

	return ctx.Err()
}

func (watcher *PathWatcher) watchExtensionHandler(msg *stan.Msg) {
	ctx := context.Background()

	response, err := parse(msg)
	if err != nil {
		log.Error("parsing failed: %s", err)
		return
	}

	defer watcher.recoverPanic()
	for event, env := range watcher.extensions {
		if strings.HasPrefix(response.Key, "/"+event) {
			watcher.runExtensionOnSync(ctx, response, env.Clone())
			return
		}
	}
}

type natsMessage struct {
	ClientID string `json:"client_id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

func parse(rawMsg *stan.Msg) (*gohan_sync.Event, error) {
	var msg natsMessage
	if err := json.Unmarshal(rawMsg.Data, &msg); err != nil {
		return nil, err
	}

	ev := gohan_sync.Event{
		Action:   "set",
		Key:      msg.Key,
		ClientID: msg.ClientID,
		Revision: int64(rawMsg.Sequence),
		Err:      nil,
	}

	err := json.Unmarshal([]byte(msg.Value), &ev.Data)
	if err != nil {
		log.Warning("failed to unmarshal watch response value %s: %s", msg.Value, err)
	}

	return &ev, nil
}

func (watcher *PathWatcher) recoverPanic() {
	err := recover()
	if err != nil {
		log.Error("%s panicked: %s: %s", watcher, err, debug.Stack())
	}
}

// storeRevision puts a revision number for a path to the sync backend.
func (watcher *PathWatcher) storeRevision(revision uint64) {
	watcher.updateGauge(revision, "previous_processed_revision")
}

//Run extension on sync
func (watcher *PathWatcher) runExtensionOnSync(ctx context.Context, response *gohan_sync.Event, env extension.Environment) {
	defer watcher.measureTime(time.Now(), response.Action)

	context := map[string]interface{}{
		"action":    response.Action,
		"data":      response.Data,
		"key":       response.Key,
		"client_id": response.ClientID,
		"revision":  response.Revision,
		"context":   ctx,
		"trace_id":  util.NewTraceID(),
	}
	if err := env.HandleEvent("notification", context); err != nil {
		log.Error("%s extension error, last processed event may be lost: %s", watcher, err)
	}
	return
}

func (watcher *PathWatcher) measureTime(timeStarted time.Time, action string) {
	metrics.UpdateTimer(timeStarted, "path_watcher.%s.%s", watcher.escapedPath, action)
}

func (watcher *PathWatcher) updateCounter(delta int64, metric string) {
	metrics.UpdateCounter(delta, "path_watcher.%s.%s", watcher.escapedPath, metric)
}

func (watcher *PathWatcher) updateGauge(value uint64, metric string) {
	metrics.UpdateGauge(int64(value), "path_watcher.%s.%s", watcher.escapedPath, metric)
}
