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
	"github.com/cloudwan/gohan/job"

	l "github.com/cloudwan/gohan/log"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

const (
	//StateUpdateEventName used in etcd path
	StateUpdateEventName = "state_update"
	//MonitoringUpdateEventName used in etcd path
	MonitoringUpdateEventName = "monitoring_update"

	SyncWatchRevisionPrefix = "/gohan/watch/revision"

	processPathPrefix = "/gohan/cluster/process"

	masterTTL = 10
)

// StartSyncWatchProcess initiates event processing on "watches".
func StartSyncWatchProcess(server *Server) {
	config := util.GetConfig()
	watch := config.GetStringList("watch/keys", nil)
	events := config.GetStringList("watch/events", nil)
	if watch == nil {
		return
	}

	extensions := map[string]extension.Environment{}
	for _, event := range events {
		path := "sync://" + event
		env, err := server.NewEnvironmentForPath("sync."+event, path)
		if err != nil {
			log.Fatal(err.Error())
		}
		extensions[event] = env
	}

	handler := func(response *gohan_sync.Event) {
		defer l.LogPanic(log)
		for _, event := range events {
			//match extensions
			if strings.HasPrefix(response.Key, "/"+event) {
				env := extensions[event]
				runExtensionOnSync(server, response, env.Clone())
				return
			}
		}
	}

	go func() {
		processList := []string{}
		myPosition := -1
		prevCancel := func() {}
		prevWG := new(sync.WaitGroup)

		defer func() {
			prevCancel()
			prevWG.Wait()
		}()

		for server.running {
			processWatchEvent := <-server.respChanProcessWatch
			log.Debug("cluster change detected: %s process %s", processWatchEvent.Action, processWatchEvent.Key)

			// modify gohan process list
			pos := -1
			for p, v := range processList {
				if v == processWatchEvent.Key {
					pos = p
				}
			}
			switch processWatchEvent.Action {
			case "delete":
				// remove detected process from list
				if pos > -1 {
					processList = append((processList)[:pos], (processList)[pos+1:]...)
				}
			default:
				// add detected process from list
				if pos == -1 {
					processList = append(processList, processWatchEvent.Key)
					sort.Sort(sort.StringSlice(processList))
				}
			}

			for p, v := range processList {
				if v == processPathPrefix+"/"+server.sync.GetProcessID() {
					myPosition = p
					break
				}
			}

			log.Debug("Current cluster consists of following processes: %s, my poistion: %d", processList, myPosition)

			// stop goroutines created by the previous iteration
			prevCancel()
			prevWG.Wait()

			ctx, cancel := context.WithCancel(context.TODO())
			prevCancel = cancel
			prevWG = new(sync.WaitGroup)
			for idx, path := range watch {
				prevWG.Add(1)
				size := len(processList)
				prio := (myPosition - (idx % size) + size) % size
				log.Debug("SyncWatch Priority of `%s`: `%d`", watch, prio)

				go func(ctx context.Context, idx int, path string, prio int) {
					defer prevWG.Done()

					backoff := time.NewTimer(time.Duration(prio*masterTTL) * time.Second)
					select {
					case <-backoff.C:
					case <-ctx.Done():
						return
					}

					for {
						err := server.processSyncWatch(ctx, path, handler)
						if err != nil && err != context.Canceled && err != lockFailedErr {
							log.Error("Sync Watch on `%s` aborted, retrying...: %s", path, err)
						}

						backoff := time.NewTimer(time.Second * masterTTL)
						select {
						case <-backoff.C:
						case <-ctx.Done():
							return
						}
					}
				}(ctx, idx, path, prio)
			}
		}
	}()
}

var lockFailedErr = errors.New("failed to lock on sync backend")

// processSyncWatch handles events on a path with a handler using the server queue.
// Returns any error or context cancel.
// This method gets a lock on the sync backend and returns with an error when fails.
func (server *Server) processSyncWatch(ctx context.Context, path string, handler func(*gohan_sync.Event)) error {
	lockKey := lockPath + "/watch" + path
	err := server.sync.Lock(lockKey, false)
	if err != nil {
		return lockFailedErr
	}
	defer server.sync.Unlock(lockKey)

	watchCtx, watchCancel := context.WithCancel(ctx)
	respCh, err := server.watchPath(watchCtx, path)
	if err != nil {
		return err
	}
	defer watchCancel()

	for response := range respCh {
		if response.Err != nil {
			return response.Err
		}

		server.queue.Add(job.NewJob(
			func() {
				resp := response
				handler(resp)
			},
		))
		err := server.storeRevision(path, response.Revision)
		if err != nil {
			return err
		}
	}

	return nil
}

// watchPath watches on the sync backend on path from the previously stored revision and emits
// events to the returned channel.
// Cancel by ctx.Cancel(). The retunred channel will be closed by any error or cancelation.
func (server *Server) watchPath(ctx context.Context, path string) (<-chan *gohan_sync.Event, error) {
	fromRevision := server.fetchStoredRevision(path)
	return server.sync.WatchContext(ctx, path, fromRevision)
}

// fetchStoredRevision returns the revision number stored in the sync backend for a path.
// When it's a new in the backend, returns sync.RevisionCurrent.
func (server *Server) fetchStoredRevision(path string) int64 { // TODO error?
	fromRevision := int64(gohan_sync.RevisionCurrent)
	lastSeen, err := server.sync.Fetch(SyncWatchRevisionPrefix + path)
	if err == nil {
		inStore, err := strconv.ParseInt(lastSeen.Value, 10, 64)
		if err == nil {
			log.Info("Using last seen revision `%d` for watching path `%s`", inStore, path)
			fromRevision = inStore
		}
	}
	return fromRevision
}

// storeRevision puts a reivision number for a path to the sync backend.
func (server *Server) storeRevision(path string, revision int64) error {
	err := server.sync.Update(SyncWatchRevisionPrefix+path, strconv.FormatInt(revision, 10))
	if err != nil {
		return fmt.Errorf("Failed to update revision number for watch path `%s` in sync storage", path)
	}
	return nil
}

//Stop Watch Process
func StopSyncWatchProcess(server *Server) {
}

//Run extension on sync
func runExtensionOnSync(server *Server, response *gohan_sync.Event, env extension.Environment) {
	context := map[string]interface{}{
		"action": response.Action,
		"data":   response.Data,
		"key":    response.Key,
	}
	if err := env.HandleEvent("notification", context); err != nil {
		log.Warning(fmt.Sprintf("extension error: %s", err))
		return
	}
	return
}
