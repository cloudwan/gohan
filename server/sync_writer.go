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
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	gohan_sync "github.com/cloudwan/gohan/sync"
)

const (
	syncPath = "/gohan/cluster/sync"
	lockPath = "/gohan/cluster/lock"

	configPrefix = "/config"

	eventPollingTime  = 30 * time.Second
	eventPollingLimit = 10000
)

// SyncWriter copies data from the RDBMS to the sync layer.
// All changes happens in the RDBMS will be synchronized into the
// sync layer by SyncWriter.
// SyncWriter gets items to sync from the event table.
type SyncWriter struct {
	sync    gohan_sync.Sync
	db      db.DB
	backoff time.Duration
}

// NewSyncWriter creates a new instance of SyncWriter.
func NewSyncWriter(sync gohan_sync.Sync, db db.DB) *SyncWriter {
	return &SyncWriter{
		sync:    sync,
		db:      db,
		backoff: time.Second * 5,
	}
}

// NewSyncWriterFromServer is a helper method for test.
// Should be removes in the future.
func NewSyncWriterFromServer(server *Server) *SyncWriter {
	return NewSyncWriter(server.sync, server.db)
}

// Run starts a loop to keep running Sync().
// This method blocks until the ctx is canceled.
func (writer *SyncWriter) Run(ctx context.Context) error {
	pollingTicker := time.Tick(eventPollingTime)
	committed := transactionCommitInformer()

	recentlySynced := false
	for {
		err := func() error {
			lost, err := writer.sync.Lock(syncPath, true)
			if err != nil {
				return err
			}
			defer writer.sync.Unlock(syncPath)

			for {
				select {
				case <-lost:
					return fmt.Errorf("lost lock for sync")
				case <-ctx.Done():
					return nil
				case <-pollingTicker:
					if recentlySynced {
						recentlySynced = false
						continue
					}
				case <-committed:
					recentlySynced = true
				}
				_, err := writer.Sync()
				if err != nil {
					return err
				}
			}
		}()

		if err != nil {
			log.Error("sync writer is intrupted: %s", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(writer.backoff):
		}
	}
}

// Sync runs a synchronization iteration, which
// executes requests in the event table.
func (writer *SyncWriter) Sync() (synced int, err error) {
	resourceList, err := writer.listEvents()
	if err != nil {
		return
	}
	for _, resource := range resourceList {
		err = writer.syncEvent(resource)
		if err != nil {
			return
		}
		synced++
	}
	return
}

func (writer *SyncWriter) listEvents() ([]*schema.Resource, error) {
	var resourceList []*schema.Resource
	if dbErr := db.Within(writer.db, func(tx transaction.Transaction) error {
		schemaManager := schema.GetManager()
		eventSchema, _ := schemaManager.Schema("event")
		paginator, _ := pagination.NewPaginator(
			pagination.OptionKey(eventSchema, "id"),
			pagination.OptionOrder(pagination.ASC),
			pagination.OptionLimit(eventPollingLimit))
		res, _, err := tx.List(eventSchema, nil, nil, paginator)
		resourceList = res
		return err
	}); dbErr != nil {
		return nil, dbErr
	}

	return resourceList, nil
}

func (writer *SyncWriter) syncEvent(resource *schema.Resource) error {
	schemaManager := schema.GetManager()
	eventSchema, _ := schemaManager.Schema("event")
	return db.Within(writer.db, func(tx transaction.Transaction) error {
		var err error
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
					return fmt.Errorf("failed to unmarshal body on sync: %s", err)
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
					return fmt.Errorf("failed to marshal marshalling sync object: %s", err)
				}
				content = string(data)
			}

			err = writer.sync.Update(path, content)
			if err != nil {
				return fmt.Errorf("Update() failed on sync: %s", err)
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
					return fmt.Errorf("Delete from sync failed %s - generating of custom path failed", err)
				}
			}
			log.Debug("deleting %s", statePrefix+deletePath)
			err = writer.sync.Delete(statePrefix+deletePath, false)
			if err != nil {
				log.Error(fmt.Sprintf("Delete from sync failed %s", err))
			}
			log.Debug("deleting %s", monitoringPrefix+deletePath)
			err = writer.sync.Delete(monitoringPrefix+deletePath, false)
			if err != nil {
				log.Error(fmt.Sprintf("Delete from sync failed %s", err))
			}
			log.Debug("deleting %s", resourcePath)
			err = writer.sync.Delete(path, false)
			if err != nil {
				return fmt.Errorf("delete from sync failed %s", err)
			}
		}
		log.Debug("delete event %d", resource.Get("id"))
		id := resource.Get("id")
		err = tx.Delete(eventSchema, id)
		if err != nil {
			return fmt.Errorf("delete failed: %s", err)
		}

		return nil
	})
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
	if !curSchema.SkipConfigPrefix() {
		path = configPrefix + path
	}
	log.Info("Generated path: %s", path)
	return path
}
