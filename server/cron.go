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

	"github.com/robfig/cron"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

//CRON Process
func startCRONProcess(server *Server) {
	manager := schema.GetManager()
	config := util.GetConfig()
	jobList := config.GetParam("cron", nil)
	if jobList == nil {
		return
	}
	log.Info("Started CRON process")
	c := cron.New()
	for _, rawJob := range jobList.([]interface{}) {
		job := rawJob.(map[string]interface{})
		path := job["path"].(string)
		timing := job["timing"].(string)
		env := newEnvironment(server.db, server.keystoneIdentity)
		err := env.LoadExtensionsForPath(manager.Extensions, path)
		if err != nil {
			log.Fatal(fmt.Sprintf("Extensions parsing error: %v", err))
		}
		log.Info("New job for %s / %s", path, timing)
		c.AddFunc(timing, func() {
			lockKey := lockPath + "/" + path
			err := server.sync.Lock(lockKey, false)
			if err != nil {
				return
			}
			defer func() {
				server.sync.Unlock(lockKey)
			}()
			tx, err := server.db.Begin()
			defer tx.Close()
			context := map[string]interface{}{
				"path": path,
			}
			if err != nil {
				log.Warning(fmt.Sprintf("extension error: %s", err))
				return
			}
			if err := env.HandleEvent("notification", context); err != nil {
				log.Warning(fmt.Sprintf("extension error: %s", err))
				return
			}
			err = tx.Commit()
			if err != nil {
				log.Warning(fmt.Sprintf("extension error: %s", err))
				return
			}
			return
		})
	}
	c.Start()
}

func stopCRONProcess(server *Server) {

}
