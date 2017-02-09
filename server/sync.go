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
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwan/gohan/db/pagination"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
)

const (
	syncPath = "/gohan/cluster/sync"
	lockPath = "/gohan/cluster/lock"

	configPrefix = "/config"

	eventPollingTime  = 30 * time.Second
	eventPollingLimit = 10000
)

//Start sync Process
func startSyncProcess(server *Server) {
	pollingTicker := time.Tick(eventPollingTime)
	committed := transactionCommitInformer()
	go func() {
		defer l.LogFatalPanic(log)
		recentlySynced := false
		for server.running {
			select {
			case <-pollingTicker:
				if recentlySynced {
					recentlySynced = false
					continue
				}
			case <-committed:
				recentlySynced = true
			}
			server.sync.Lock(syncPath, true)
			server.Sync()
		}
		server.sync.Unlock(syncPath)
	}()
}

//Stop Sync Process
func stopSyncProcess(server *Server) {
	server.sync.Unlock(syncPath)
}

//Sync to sync backend database table
func (server *Server) Sync() error {
	resourceList, err := server.listEvents()
	if err != nil {
		return err
	}
	for _, resource := range resourceList {
		err = server.syncEvent(resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (server *Server) listEvents() ([]*schema.Resource, error) {
	tx, err := server.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Close()
	schemaManager := schema.GetManager()
	eventSchema, _ := schemaManager.Schema("event")
	paginator, _ := pagination.NewPaginator(eventSchema, "id", pagination.ASC, eventPollingLimit, 0)
	resourceList, _, err := tx.List(eventSchema, nil, paginator)
	if err != nil {
		return nil, err
	}
	return resourceList, nil
}

func (server *Server) syncEvent(resource *schema.Resource) error {
	schemaManager := schema.GetManager()
	eventSchema, _ := schemaManager.Schema("event")
	tx, err := server.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	eventType := resource.Get("type").(string)
	resourcePath := resource.Get("path").(string)
	body := resource.Get("body").(string)
	syncPlain := resource.Get("sync_plain").(bool)
	syncProperty := resource.Get("sync_property").(string)

	path := generatePath(resourcePath, body)

	version, ok := resource.Get("version").(int)
	if !ok {
		log.Debug("cannot cast version value in int for %s", path)
	}
	log.Debug("event %s", eventType)

	if eventType == "create" || eventType == "update" {
		log.Debug("set %s on sync", path)

		content := body

		var data map[string]interface{}
		if syncProperty != "" {
			err = json.Unmarshal(([]byte)(body), &data)
			if err != nil {
				log.Error(fmt.Sprintf("failed to unmarshal body on sync: %s", err))
				return err
			}
			target, ok := data[syncProperty]
			if !ok {
				return fmt.Errorf("could not find property `%s`", syncProperty)
			}
			jsonData, err := json.Marshal(target)
			if err != nil {
				return err
			}
			content = string(jsonData)
		}

		if syncPlain {
			var target interface{}
			json.Unmarshal([]byte(content), &target)
			switch target.(type) {
			case string:
				content = fmt.Sprintf("%v", target)
			}
		} else {
			data, err := json.Marshal(map[string]interface{}{
				"body":    content,
				"version": version,
			})
			if err != nil {
				log.Error(fmt.Sprintf("When marshalling sync object: %s", err))
				return err
			}
			content = string(data)
		}

		err = server.sync.Update(path, content)
		if err != nil {
			log.Error(fmt.Sprintf("%s on sync", err))
			return err
		}
	} else if eventType == "delete" {
		log.Debug("delete %s", resourcePath)
		deletePath := resourcePath
		resourceSchema := schema.GetSchemaByURLPath(resourcePath)
		if _, ok := resourceSchema.SyncKeyTemplate(); ok {
			var data map[string]interface{}
			json.Unmarshal(([]byte)(body), &data)
			deletePath, err = resourceSchema.GenerateCustomPath(data)
			if err != nil {
				log.Error(fmt.Sprintf("Delete from sync failed %s - generating of custom path failed", err))
				return err
			}
		}
		log.Debug("deleting %s", statePrefix+deletePath)
		err = server.sync.Delete(statePrefix + deletePath)
		if err != nil {
			log.Error(fmt.Sprintf("Delete from sync failed %s", err))
		}
		log.Debug("deleting %s", monitoringPrefix+deletePath)
		err = server.sync.Delete(monitoringPrefix + deletePath)
		if err != nil {
			log.Error(fmt.Sprintf("Delete from sync failed %s", err))
		}
		log.Debug("deleting %s", resourcePath)
		err = server.sync.Delete(path)
		if err != nil {
			log.Error(fmt.Sprintf("Delete from sync failed %s", err))
			return err
		}
	}
	log.Debug("delete event %d", resource.Get("id"))
	id := resource.Get("id")
	err = tx.Delete(eventSchema, id)
	if err != nil {
		log.Error(fmt.Sprintf("delete failed: %s", err))
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error(fmt.Sprintf("commit failed: %s", err))
		return err
	}
	return nil
}

func generatePath(resourcePath string, body string) string {
	var curSchema = schema.GetSchemaByURLPath(resourcePath)
	path := resourcePath
	if _, ok := curSchema.SyncKeyTemplate(); ok {
		var data map[string]interface{}
		err := json.Unmarshal(([]byte)(body), &data)
		if err != nil {
			log.Error(fmt.Sprintf("Error %v during unmarshaling data %v", err, data))
		} else {
			path, err = curSchema.GenerateCustomPath(data)
			if err != nil {
				path = resourcePath
				log.Error(fmt.Sprintf("%v", err))
			}
		}
	}
	path = configPrefix + path
	log.Info("Generated path: %s", path)
	return path
}
