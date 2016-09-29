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
	"strings"
	"time"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"

	"github.com/cloudwan/gohan/schema"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

const (
	statePrefix      = "/state/"
	monitoringPrefix = "/monitoring/"
)

//TODO(nati) integrate with watch process
func startStateUpdatingProcess(server *Server) {

	stateResponseChan := make(chan *gohan_sync.Event)
	stateStopChan := make(chan bool)

	if _, err := server.sync.Fetch(statePrefix); err != nil {
		server.sync.Update(statePrefix, "")
	}

	if _, err := server.sync.Fetch(monitoringPrefix); err != nil {
		server.sync.Update(monitoringPrefix, "")
	}

	go func() {
		defer util.LogFatalPanic(log)
		for server.running {
			lockKey := lockPath + "state"
			err := server.sync.Lock(lockKey, true)
			if err != nil {
				log.Warning("Can't start state watch process due to lock", err)
				time.Sleep(5 * time.Second)
				continue
			}
			defer func() {
				server.sync.Unlock(lockKey)
			}()

			err = server.sync.Watch(statePrefix, stateResponseChan, stateStopChan)
			if err != nil {
				log.Error(fmt.Sprintf("sync watch error: %s", err))
			}
		}
	}()
	go func() {
		defer util.LogFatalPanic(log)
		for server.running {
			response := <-stateResponseChan
			go func() {
				err := StateUpdate(response, server)
				if err != nil {
					log.Warning(fmt.Sprintf("error during state update: %s", err))
				}
				log.Info("Completed StateUpdate")
			}()
		}
		stateStopChan <- true
	}()
	monitoringResponseChan := make(chan *gohan_sync.Event)
	monitoringStopChan := make(chan bool)
	go func() {
		defer util.LogFatalPanic(log)
		for server.running {
			lockKey := lockPath + "monitoring"
			err := server.sync.Lock(lockKey, true)
			if err != nil {
				log.Warning("Can't start state watch process due to lock", err)
				time.Sleep(5 * time.Second)
				continue
			}
			defer func() {
				server.sync.Unlock(lockKey)
			}()
			err = server.sync.Watch(monitoringPrefix, monitoringResponseChan, monitoringStopChan)
			if err != nil {
				log.Error(fmt.Sprintf("sync watch error: %s", err))
			}
		}
	}()
	go func() {
		defer util.LogFatalPanic(log)
		for server.running {
			response := <-monitoringResponseChan
			go func() {
				err := MonitoringUpdate(response, server)
				if err != nil {
					log.Warning(fmt.Sprintf("error during monitoring update: %s", err))
				}
				log.Info("Completed MonitoringUpdate")
			}()
		}
		monitoringStopChan <- true
	}()
}

func stopStateUpdatingProcess(server *Server) {
}

//StateUpdate updates the state in the db based on the sync event
func StateUpdate(response *gohan_sync.Event, server *Server) error {
	dataStore := server.db
	schemaPath := "/" + strings.TrimPrefix(response.Key, statePrefix)
	var curSchema = schema.GetSchemaByPath(schemaPath)
	if curSchema == nil || !curSchema.StateVersioning() {
		log.Debug("State update on unexpected path '%s'", schemaPath)
		return nil
	}
	resourceID := curSchema.GetResourceIDFromPath(schemaPath)
	log.Info("Started StateUpdate for %s %s %v", response.Action, response.Key, response.Data)

	tx, err := dataStore.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	err = tx.SetIsolationLevel(transaction.GetIsolationLevel(curSchema, StateUpdateEventName))
	if err != nil {
		return err
	}
	curResource, err := tx.Fetch(curSchema, transaction.IDFilter(resourceID))
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
	stateVersion, ok := response.Data["version"].(float64)
	if !ok {
		return fmt.Errorf("No version in state information")
	}
	oldStateVersion := resourceState.StateVersion
	resourceState.StateVersion = int64(stateVersion)
	if resourceState.StateVersion < oldStateVersion {
		return nil
	}
	if newError, ok := response.Data["error"].(string); ok {
		resourceState.Error = newError
	}
	if newState, ok := response.Data["state"].(string); ok {
		resourceState.State = newState
	}

	environmentManager := extension.GetManager()
	environment, haveEnvironment := environmentManager.GetEnvironment(curSchema.ID)
	context := map[string]interface{}{}

	if haveEnvironment {
		serviceAuthorization, err := server.keystoneIdentity.GetServiceAuthorization()
		if err != nil {
			return err
		}

		context["catalog"] = serviceAuthorization.Catalog()
		context["auth_token"] = serviceAuthorization.AuthToken()
		context["resource"] = curResource.Data()
		context["schema"] = curSchema
		context["state"] = response.Data
		context["config_version"] = resourceState.ConfigVersion
		context["transaction"] = tx

		if err := extension.HandleEvent(context, environment, "pre_state_update_in_transaction"); err != nil {
			return err
		}
	}

	err = tx.StateUpdate(curResource, &resourceState)
	if err != nil {
		return err
	}

	if haveEnvironment {
		if err := extension.HandleEvent(context, environment, "post_state_update_in_transaction"); err != nil {
			return err
		}
	}

	return tx.Commit()
}

//MonitoringUpdate updates the state in the db based on the sync event
func MonitoringUpdate(response *gohan_sync.Event, server *Server) error {
	dataStore := server.db
	schemaPath := "/" + strings.TrimPrefix(response.Key, monitoringPrefix)
	var curSchema = schema.GetSchemaByPath(schemaPath)
	if curSchema == nil || !curSchema.StateVersioning() {
		log.Debug("Monitoring update on unexpected path '%s'", schemaPath)
		return nil
	}
	resourceID := curSchema.GetResourceIDFromPath(schemaPath)
	log.Info("Started MonitoringUpdate for %s %s %v", response.Action, response.Key, response.Data)

	tx, err := dataStore.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	err = tx.SetIsolationLevel(transaction.GetIsolationLevel(curSchema, MonitoringUpdateEventName))
	if err != nil {
		return err
	}
	curResource, err := tx.Fetch(curSchema, transaction.IDFilter(resourceID))
	if err != nil {
		return err
	}
	resourceState, err := tx.StateFetch(curSchema, transaction.IDFilter(resourceID))
	if err != nil {
		return err
	}
	if resourceState.ConfigVersion != resourceState.StateVersion {
		log.Debug("Skipping MonitoringUpdate, because config version (%s) != state version (%s)",
			resourceState.ConfigVersion, resourceState.StateVersion)
		return nil
	}
	var ok bool
	monitoringVersion, ok := response.Data["version"].(float64)
	if !ok {
		return fmt.Errorf("No version in monitoring information")
	}
	if resourceState.ConfigVersion != int64(monitoringVersion) {
		return nil
	}
	resourceState.Monitoring, ok = response.Data["monitoring"].(string)
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
		if err := extension.HandleEvent(context, environment, "pre_monitoring_update_in_transaction"); err != nil {
			return err
		}
	}

	err = tx.StateUpdate(curResource, &resourceState)
	if err != nil {
		return err
	}

	if haveEnvironment {
		if err := extension.HandleEvent(context, environment, "post_monitoring_update_in_transaction"); err != nil {
			return err
		}
	}

	return tx.Commit()
}
