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
	"runtime/debug"
	"strings"

	"github.com/cloudwan/gohan/util"
	"github.com/robfig/cron"
)

//CRON Process
func startCRONProcess(server *Server) {
	config := util.GetConfig()
	jobList := config.GetParam("cron", nil)
	if jobList == nil {
		return
	}
	if server.sync == nil {
		log.Fatalf("Could not start CRON process because of sync backend misconfiguration.")
	}
	log.Info("Started CRON process")
	c := cron.New()
	var jobLocks = map[string](chan int){}

	for _, rawJob := range jobList.([]interface{}) {
		job := rawJob.(map[string]interface{})
		path := job["path"].(string)
		timing := job["timing"].(string)
		name := strings.TrimPrefix(path, "cron://")
		log.Info("New job for %s / %s", path, timing)
		lockKey := lockPath + "/" + name
		jobLocks[lockKey] = make(chan int, 1)
		jobLocks[lockKey] <- 1
		env, err := server.NewEnvironmentForPath(name, path)
		if err != nil {
			log.Fatal(err.Error())
		}

		takeLock := func(ctx context.Context) error {
			select {
			case <-jobLocks[lockKey]:
				_, err := server.sync.Lock(ctx, lockKey, false)
				if err != nil {
					log.Debug("Failed to take ETCD lock")
					jobLocks[lockKey] <- 1
				}
				return err
			default:
				log.Debug("Failed to take lock: %s", lockKey)
				return errors.New("Another cron job is running")
			}
		}

		if err = c.AddFunc(timing, func() {
			ctx := context.Background()
			err := takeLock(ctx)
			if err != nil {
				log.Info("Failed to schedule cron job, err: %s", err.Error())
				return
			}
			defer func() {
				if r := recover(); r != nil {
					log.Error("Cron job '%s' panicked: %s %s", path, r, string(debug.Stack()))
				}
				log.Debug("Unlocking %s", lockKey)
				jobLocks[lockKey] <- 1
				if err := server.sync.Unlock(ctx, lockKey); err != nil {
					log.Warning("CRON: unlocking etcd failed: %s", err)
				}
			}()

			eventCtx := map[string]interface{}{
				"path":     path,
				"context":  ctx,
				"trace_id": util.NewTraceID(),
			}

			clone := env.Clone()
			if err := clone.HandleEvent("notification", eventCtx); err != nil {
				log.Warning(fmt.Sprintf("extension error: %s", err))
				return
			}
			return
		}); err != nil {
			log.Panicf("Adding CRON job failed, check server config: %s", err)
		}
	}
	c.Start()
}

func stopCRONProcess(server *Server) {

}
