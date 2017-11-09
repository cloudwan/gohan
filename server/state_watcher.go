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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	gohan_sync "github.com/cloudwan/gohan/sync"
)

const (
	stateWatchPrefix = "/state_watch"
	statePrefix      = stateWatchPrefix + "/state"
	monitoringPrefix = stateWatchPrefix + "/monitoring"

	//StateUpdateEventName used in etcd path
	StateUpdateEventName = "state_update"
	//MonitoringUpdateEventName used in etcd path
	MonitoringUpdateEventName = "monitoring_update"
)

var stateWatchTrimmer = regexp.MustCompile("^(" + statePrefix + "|" + monitoringPrefix + ")")

// StateWatcher handles Gohan's state update events from the sync layer.
// Resources configured with `state_versioning: true` receives
// monitoring_update and state_update events from StateWatcher when it
// detects updates in the sync layer.
type StateWatcher struct {
	sync     gohan_sync.Sync
	db       db.DB
	identity middleware.IdentityService
	backoff  time.Duration
}

// NewStateWatcher creates a new instance of StateWatcher.
func NewStateWatcher(sync gohan_sync.Sync, db db.DB, identity middleware.IdentityService) *StateWatcher {
	return &StateWatcher{
		sync:     sync,
		db:       db,
		identity: identity,
		backoff:  time.Second * 5,
	}
}

// NewStateWatcherFromServer is a constructor for StateWatcher
func NewStateWatcherFromServer(server *Server) *StateWatcher {
	return NewStateWatcher(server.sync, server.db, server.keystoneIdentity)
}

// Run starts the main loop of the watcher.
// This method blocks until canceled by the ctx.
// Errors are logged, but do not interrupt the loop.
func (watcher *StateWatcher) Run(ctx context.Context) error {
	for {
		err := watcher.iterate(ctx)
		if err != nil {
			log.Error("state watch error: %s", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(watcher.backoff):
		}
	}
}

func (watcher *StateWatcher) iterate(ctx context.Context) error {
	lockKey := lockPath + "/state_watch"
	lost, err := watcher.sync.Lock(lockKey, true)
	if err != nil {
		// lock failed, another process is running
		return nil
	}
	defer watcher.sync.Unlock(lockKey)

	watchCtx, watchCancel := context.WithCancel(ctx)
	respCh := watcher.sync.WatchContext(watchCtx, lockKey, gohan_sync.RevisionCurrent)
	watchErr := make(chan error, 1)
	go func() {
		watchErr <- func() error {
			for response := range respCh {
				if response.Err != nil {
					return err
				}
				watcher.processEvent(response)
			}

			return nil
		}()
	}()

	select {
	case <-ctx.Done():
		<-watchErr
		return ctx.Err()
	case <-lost:
		watchCancel()
		<-watchErr
		return fmt.Errorf("state watch canceled as lock is lost")
	case err := <-watchErr:
		watchCancel()
		return err
	}
}

func (watcher *StateWatcher) processEvent(event *gohan_sync.Event) {
	var err error
	if strings.HasPrefix(event.Key, statePrefix) {
		err = watcher.StateUpdate(event)
		log.Info("Completed StateUpdate")
	} else if strings.HasPrefix(event.Key, monitoringPrefix) {
		err = watcher.MonitoringUpdate(event)
		log.Info("Completed MonitoringUpdate")
	}
	if err != nil {
		log.Warning(fmt.Sprintf("error during state update: %s", err))
	}
}

//StateUpdate updates the state in the db based on the sync event
// todo: make private once tests are fixed
func (watcher *StateWatcher) StateUpdate(event *gohan_sync.Event) error {
	schemaPath := strings.TrimPrefix(event.Key, statePrefix)
	var curSchema = schema.GetSchemaByPath(schemaPath)
	if curSchema == nil || !curSchema.StateVersioning() {
		log.Debug("State update on unexpected path '%s'", schemaPath)
		return nil
	}

	defer watcher.measureStateUpdateTime(time.Now(), "state_update", curSchema.ID)

	resourceID := curSchema.GetResourceIDFromPath(schemaPath)
	log.Info("Started StateUpdate for %s %s %v", event.Action, event.Key, event.Data)

	return db.WithinTx(context.Background(), watcher.db, &transaction.TxOptions{IsolationLevel: transaction.GetIsolationLevel(curSchema, StateUpdateEventName)},
		func(tx transaction.Transaction) error {
			curResource, err := tx.Fetch(curSchema, transaction.IDFilter(resourceID), nil)
			if err != nil {
				return err
			}
			resourceState, err := tx.StateFetch(curSchema, transaction.IDFilter(resourceID))
			if err != nil {
				return err
			}
			if resourceState.StateVersion == resourceState.ConfigVersion {
				return nil
			}
			stateVersion, ok := event.Data["version"].(float64)
			if !ok {
				return fmt.Errorf("No version in state information")
			}
			oldStateVersion := resourceState.StateVersion
			resourceState.StateVersion = int64(stateVersion)
			if resourceState.StateVersion < oldStateVersion {
				return nil
			}
			if newError, ok := event.Data["error"].(string); ok {
				resourceState.Error = newError
			}
			if newState, ok := event.Data["state"].(string); ok {
				resourceState.State = newState
			}

			environmentManager := extension.GetManager()
			environment, haveEnvironment := environmentManager.GetEnvironment(curSchema.ID)
			context := map[string]interface{}{}

			if haveEnvironment {
				serviceAuthorization, err := watcher.identity.GetServiceAuthorization()
				if err != nil {
					return err
				}

				context["catalog"] = serviceAuthorization.Catalog()
				context["auth_token"] = serviceAuthorization.AuthToken()
				context["resource"] = curResource.Data()
				context["schema"] = curSchema
				context["state"] = event.Data
				context["config_version"] = resourceState.ConfigVersion
				context["transaction"] = tx

				if err := extension.HandleEvent(context, environment, "pre_state_update_in_transaction", curSchema.ID); err != nil {
					return err
				}
			}

			err = tx.StateUpdate(curResource, &resourceState)
			if err != nil {
				return err
			}

			if haveEnvironment {
				if err := extension.HandleEvent(context, environment, "post_state_update_in_transaction", curSchema.ID); err != nil {
					return err
				}
			}

			return tx.Commit()
		})
}

//MonitoringUpdate updates the state in the db based on the sync event
// todo: make private once tests are fixed
func (watcher *StateWatcher) MonitoringUpdate(event *gohan_sync.Event) error {
	schemaPath := strings.TrimPrefix(event.Key, monitoringPrefix)
	var curSchema = schema.GetSchemaByPath(schemaPath)
	if curSchema == nil || !curSchema.StateVersioning() {
		log.Debug("Monitoring update on unexpected path '%s'", schemaPath)
		return nil
	}
	defer watcher.measureStateUpdateTime(time.Now(), "monitoring_update", curSchema.ID)

	resourceID := curSchema.GetResourceIDFromPath(schemaPath)
	log.Info("Started MonitoringUpdate for %s %s %v", event.Action, event.Key, event.Data)

	return db.WithinTx(context.Background(), watcher.db, &transaction.TxOptions{IsolationLevel: transaction.GetIsolationLevel(curSchema, MonitoringUpdateEventName)},
		func(tx transaction.Transaction) error {
			curResource, err := tx.Fetch(curSchema, transaction.IDFilter(resourceID), nil)
			if err != nil {
				return err
			}
			resourceState, err := tx.StateFetch(curSchema, transaction.IDFilter(resourceID))
			if err != nil {
				return err
			}

			if resourceState.ConfigVersion != resourceState.StateVersion {
				log.Debug("Skipping MonitoringUpdate, because config version (%d) != state version (%d)",
					resourceState.ConfigVersion, resourceState.StateVersion)
				return nil
			}
			var ok bool
			monitoringVersion, ok := event.Data["version"].(float64)
			if !ok {
				return fmt.Errorf("No version in monitoring information")
			}
			if resourceState.ConfigVersion != int64(monitoringVersion) {
				log.Debug("Dropping MonitoringUpdate, because config version (%d) != input monitoring version (%d)",
					resourceState.ConfigVersion, monitoringVersion)
				return nil
			}
			resourceState.Monitoring, ok = event.Data["monitoring"].(string)
			if !ok {
				return fmt.Errorf("No monitoring in monitoring information")
			}

			environmentManager := extension.GetManager()
			environment, haveEnvironment := environmentManager.GetEnvironment(curSchema.ID)
			context := map[string]interface{}{}
			context["resource"] = curResource.Data()
			context["schema"] = curSchema
			context["monitoring"] = resourceState.Monitoring
			context["transaction"] = tx

			if haveEnvironment {
				if err := extension.HandleEvent(context, environment, "pre_monitoring_update_in_transaction", curSchema.ID); err != nil {
					return err
				}
			}

			err = tx.StateUpdate(curResource, &resourceState)
			if err != nil {
				return err
			}

			if haveEnvironment {
				if err := extension.HandleEvent(context, environment, "post_monitoring_update_in_transaction", curSchema.ID); err != nil {
					return err
				}
			}

			return tx.Commit()
		})
}

func (watcher *StateWatcher) measureStateUpdateTime(timeStarted time.Time, event string, schemaID string) {
	metrics.UpdateTimer(timeStarted, "state.%s.%s", schemaID, event)
}
