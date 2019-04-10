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
	"sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

const (
	syncPath = "/gohan/cluster/sync"
	lockPath = "/gohan/cluster/lock"

	configPrefix = "/config"

	defaultBackoff       = 5 * time.Second
	defaultUnlockTimeout = 3 * time.Second
)

// SyncWriter copies data from the RDBMS to the sync layer.
// All changes happens in the RDBMS will be synchronized into the
// sync layer by SyncWriter.
// SyncWriter gets items to sync from the event table.
type SyncWriter struct {
	sync          gohan_sync.Sync
	db            db.DB
	backoff       time.Duration
	unlockTimeout time.Duration
}

// NewSyncWriter creates a new instance of SyncWriter.
func NewSyncWriter(sync gohan_sync.Sync, db db.DB) *SyncWriter {
	return &SyncWriter{
		sync:          sync,
		db:            db,
		backoff:       getBackoff(),
		unlockTimeout: getUnlockTimeout(),
	}
}

func getBackoff() time.Duration {
	return util.GetConfig().GetDuration("sync_writer/backoff", defaultBackoff)
}

func getUnlockTimeout() time.Duration {
	return util.GetConfig().GetDuration("sync_writer/unlock", defaultUnlockTimeout)
}

// NewSyncWriterFromServer is a helper method for test.
// Should be removes in the future.
func NewSyncWriterFromServer(server *Server) *SyncWriter {
	return NewSyncWriter(server.sync, server.db)
}

// Run starts a loop to keep running Sync().
// This method blocks until the ctx is canceled.
func (writer *SyncWriter) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	for {
		if err := writer.run(ctx); err != nil {
			log.Error("SyncWriter was interrupted: %s", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(writer.backoff):
		}
	}
}

func (writer *SyncWriter) run(ctx context.Context) error {
	lost, err := writer.sync.Lock(ctx, syncPath, true)
	if err != nil {
		return err
	}
	defer func() {
		// can't use the parent context, it may be already canceled
		unlockCtx, cancel := context.WithTimeout(context.Background(), writer.unlockTimeout)
		defer cancel()

		if err := writer.sync.Unlock(unlockCtx, syncPath); err != nil {
			log.Warning("SyncWriter: unlocking failed: %s", err)
		}
	}()

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()
	triggerCh := writer.sync.Watch(watchCtx, SyncKeyTxCommitted, goext.RevisionCurrent)

	for {
		select {
		case <-lost:
			writer.updateCounter(1, "locks_lost")
			return fmt.Errorf("lost lock for sync")
		case <-ctx.Done():
			return nil
		case event, open := <-triggerCh:
			if !open {
				// triggerCh could be closed due to context.Cancel()
				// it's go schedulers arbitrary decision if <-ctx.Done() or <-triggerCh receives first,
				// so we ensure the expected channel priorities ourselves
				return nil
			}

			if event.Err != nil {
				writer.updateCounter(1, "event_error")
				return event.Err
			}

			if err := writer.triggerSync(ctx, getEventId(event)); err != nil {
				return err
			}

		}
	}
}

func getEventId(event *gohan_sync.Event) int {
	eventId := event.Data["event_id"]

	// our sync implementation converts data from ETCD to map[string]interface{}
	// numbers are by default unmarshalled to float64
	return int(eventId.(float64))
}

func (writer *SyncWriter) triggerSync(ctx context.Context, eventId int) error {
	writer.updateCounter(1, "wake_up.on_trigger")
	_, err := writer.Sync(ctx)
	return err
}

// Sync runs a synchronization iteration, which
// executes requests in the event table.
func (writer *SyncWriter) Sync(ctx context.Context) (synced int, err error) {
	writer.updateCounter(1, "syncs")
	resourceList, err := writer.listEvents()
	if err != nil {
		return
	}
	for _, resource := range resourceList {
		err = writer.syncEvent(ctx, resource)
		if err != nil {
			return
		}
		synced++
	}

	if synced == 0 {
		writer.updateCounter(1, "empty_syncs")
	}

	return
}

func (writer *SyncWriter) listEvents() ([]*schema.Resource, error) {
	var resourceList []*schema.Resource
	if dbErr := db.WithinTx(writer.db, func(tx transaction.Transaction) error {
		schemaManager := schema.GetManager()
		eventSchema, _ := schemaManager.Schema("event")
		paginator, _ := pagination.NewPaginator(
			pagination.OptionKey(eventSchema, "id"),
			pagination.OptionOrder(pagination.ASC),
		)
		res, _, err := tx.List(context.Background(), eventSchema, nil, nil, paginator)
		resourceList = res
		return err
	}); dbErr != nil {
		return nil, dbErr
	}

	return resourceList, nil
}

func (writer *SyncWriter) syncEvent(ctx context.Context, resource *schema.Resource) error {
	schemaManager := schema.GetManager()
	eventSchema, _ := schemaManager.Schema("event")
	return db.WithinTx(writer.db, func(tx transaction.Transaction) error {
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

			err = writer.sync.Update(ctx, path, content)
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
					return fmt.Errorf("Delete from sync failed: %s - generating of custom path failed", err)
				}
			}
			log.Debug("deleting %s", statePrefix+deletePath)
			err = writer.sync.Delete(ctx, statePrefix+deletePath, false)
			if err != nil {
				log.Error(fmt.Sprintf("Delete from sync failed: %s", err))
			}
			log.Debug("deleting %s", monitoringPrefix+deletePath)
			err = writer.sync.Delete(ctx, monitoringPrefix+deletePath, false)
			if err != nil {
				log.Error(fmt.Sprintf("Delete from sync failed: %s", err))
			}
			log.Debug("deleting %s", resourcePath)
			err = writer.sync.Delete(ctx, path, false)
			if err != nil {
				return fmt.Errorf("delete from sync failed: %s", err)
			}
		}
		log.Debug("delete event %d", resource.Get("id"))
		id := resource.Get("id")
		err = tx.Delete(ctx, eventSchema, id)
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

func (writer *SyncWriter) updateCounter(delta int64, metric string) {
	metrics.UpdateCounter(delta, "sync_writer.%s", metric)
}
