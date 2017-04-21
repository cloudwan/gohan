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
	"fmt"
	"strconv"
	"strings"
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
)

//Sync Watch Process
func startSyncWatchProcess(server *Server) {
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
	responseChans := make(map[string]chan *gohan_sync.Event)
	stopChan := make(chan bool)
	for _, path := range watch {
		responseChans[path] = make(chan *gohan_sync.Event)
		go func(path string) {
			defer l.LogFatalPanic(log)
			responseChan := responseChans[path]
			for server.running {
				func() {
					lockKey := lockPath + "/watch" + path
					err := server.sync.Lock(lockKey, true)
					if err != nil {
						log.Warning("Can't start watch process due to lock", err)
						time.Sleep(5 * time.Second)
						return
					}
					defer server.sync.Unlock(lockKey)

					fromRevision := int64(gohan_sync.RevisionCurrent)
					lastSeen, err := server.sync.Fetch(SyncWatchRevisionPrefix + path)
					if err == nil {
						inStore, err := strconv.ParseInt(lastSeen.Value, 10, 64)
						if err == nil {
							log.Info("Using last seen revision `%d` for watching path `%s`", inStore, path)
							fromRevision = inStore
						}
					}

					err = server.sync.Watch(path, responseChan, stopChan, fromRevision)
					if err != nil {
						log.Error(fmt.Sprintf("sync watch error: %s", err))
					}
				}()
			}
		}(path)
	}
	//main response lisnter process
	for _, path := range watch {
		go func(path string) {
			defer l.LogFatalPanic(log)
			responseChan := responseChans[path]
			for server.running {
				response := <-responseChan
				err := server.sync.Update(SyncWatchRevisionPrefix+path, strconv.FormatInt(response.Revision, 10))
				if err != nil {
					log.Error("Failed to update revision number for watch path `%s` in sync storage", path)
				}
				server.queue.Add(job.NewJob(
					func() {
						defer l.LogPanic(log)
						for _, event := range events {
							//match extensions
							if strings.HasPrefix(response.Key, "/"+event) {
								env := extensions[event]
								runExtensionOnSync(server, response, env.Clone())
								return
							}
						}
					}),
				)
			}
		}(path)
	}
}

//Stop Watch Process
func stopSyncWatchProcess(server *Server) {
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
