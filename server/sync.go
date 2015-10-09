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
	"strings"
	"sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"

	"github.com/cloudwan/gohan/schema"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/cloudwan/gohan/util"
)

const (
	syncPath = "gohan/cluster/sync"
	lockPath = "gohan/cluster/lock"

	configPrefix     = "/config/"
	statePrefix      = "/state/"
	monitoringPrefix = "/monitoring/"

	eventPollingTime  = 30 * time.Second
	eventPollingLimit = 10000
)

var transactionCommited chan int

func transactionCommitInformer() chan int {
	if transactionCommited == nil {
		transactionCommited = make(chan int, 1)
	}
	return transactionCommited
}

//DbSyncWrapper wraps db.DB so it logs events in database on every transaction.
type DbSyncWrapper struct {
	db.DB
}

// Begin wraps transaction object with sync
func (sw *DbSyncWrapper) Begin() (transaction.Transaction, error) {
	tx, err := sw.DB.Begin()
	if err != nil {
		return nil, err
	}
	return syncTransactionWrap(tx), nil
}

type transactionEventLogger struct {
	transaction.Transaction
	eventLogged bool
}

func syncTransactionWrap(tx transaction.Transaction) *transactionEventLogger {
	return &transactionEventLogger{tx, false}
}

func (tl *transactionEventLogger) logEvent(eventType string, resource *schema.Resource, version int64) error {
	schemaManager := schema.GetManager()
	eventSchema, ok := schemaManager.Schema("event")
	if !ok {
		return fmt.Errorf("event schema not found")
	}

	if resource.Schema().Metadata["nosync"] == true {
		log.Debug("skipping event logging for schema: %s", resource.Schema().ID)
		return nil
	}
	body, err := resource.JSONString()
	if err != nil {
		return fmt.Errorf("Error during event resource deserialisation: %s", err.Error())
	}
	eventResource, err := schema.NewResource(eventSchema, map[string]interface{}{
		"type":      eventType,
		"path":      resource.Path(),
		"version":   version,
		"body":      body,
		"timestamp": int64(time.Now().Unix()),
	})
	tl.eventLogged = true
	return tl.Transaction.Create(eventResource)
}

func (tl *transactionEventLogger) Create(resource *schema.Resource) error {
	err := tl.Transaction.Create(resource)
	if err != nil {
		return err
	}
	return tl.logEvent("create", resource, 1)
}

func (tl *transactionEventLogger) Update(resource *schema.Resource) error {
	err := tl.Transaction.Update(resource)
	if err != nil {
		return err
	}
	if !resource.Schema().StateVersioning() {
		return tl.logEvent("update", resource, 0)
	}
	state, err := tl.StateFetch(resource.Schema(), resource.ID(), nil)
	if err != nil {
		return err
	}
	return tl.logEvent("update", resource, state.ConfigVersion)
}

func (tl *transactionEventLogger) Delete(s *schema.Schema, resourceID interface{}) error {
	resource, err := tl.Fetch(s, resourceID, nil)
	if err != nil {
		return err
	}
	configVersion := int64(0)
	if resource.Schema().StateVersioning() {
		state, err := tl.StateFetch(s, resourceID, nil)
		if err != nil {
			return err
		}
		configVersion = state.ConfigVersion + 1
	}
	err = tl.Transaction.Delete(s, resourceID)
	if err != nil {
		return err
	}
	return tl.logEvent("delete", resource, configVersion)
}

func (tl *transactionEventLogger) Commit() error {
	err := tl.Transaction.Commit()
	if err != nil {
		return err
	}
	if !tl.eventLogged {
		return nil
	}
	committed := transactionCommitInformer()
	select {
	case committed <- 1:
	default:
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
	path := resource.Get("path").(string)
	path = configPrefix + path
	body := resource.Get("body").(string)
	version, _ := resource.Get("version").(uint64)
	log.Debug("event %s", eventType)

	if eventType == "create" || eventType == "update" {
		log.Debug("set %s on sync", path)
		content, err := json.Marshal(map[string]interface{}{
			"body":    body,
			"version": version,
		})
		if err != nil {
			log.Error(fmt.Sprintf("When marshalling sync object: %s", err))
			return err
		}
		err = server.sync.Update(path, string(content))
		if err != nil {
			log.Error(fmt.Sprintf("%s on sync", err))
			return err
		}
	} else if eventType == "delete" {
		log.Debug("delete %s", path)
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

//Start sync Process
func startSyncProcess(server *Server) {
	pollingTicker := time.Tick(eventPollingTime)
	committed := transactionCommitInformer()
	go func() {
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

//StateUpdate updates the state in the db based on the sync event
func StateUpdate(response *gohan_sync.Event, server *Server) error {
	lockKey := lockPath + response.Key
	err := server.sync.Lock(lockKey, false)
	if err != nil {
		return err
	}
	defer func() {
		server.sync.Unlock(lockKey)
	}()
	dataStore := server.db
	schemaPath := "/" + strings.TrimPrefix(response.Key, statePrefix)
	var curSchema *schema.Schema
	manager := schema.GetManager()
	for _, s := range manager.Schemas() {
		if strings.HasPrefix(schemaPath, s.URL) {
			curSchema = s
			break
		}
	}
	if curSchema == nil || !curSchema.StateVersioning() {
		log.Debug("State update on unexpected path '%s'", schemaPath)
		return nil
	}
	resourceID := strings.TrimPrefix(schemaPath, curSchema.URL+"/")

	tx, err := dataStore.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	curResource, err := tx.Fetch(curSchema, resourceID, nil)
	if err != nil {
		return err
	}
	resourceState, err := tx.StateFetch(curSchema, resourceID, nil)
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
		serviceAuthorization, _ := server.keystoneIdentity.GetServiceAuthorization()

		context["catalog"] = serviceAuthorization.Catalog()
		context["auth_token"] = serviceAuthorization.AuthToken()
		context["resource"] = curResource.Data()
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
	lockKey := lockPath + response.Key
	err := server.sync.Lock(lockKey, false)
	if err != nil {
		return err
	}
	defer func() {
		server.sync.Unlock(lockKey)
	}()
	dataStore := server.db
	schemaPath := "/" + strings.TrimPrefix(response.Key, monitoringPrefix)
	var curSchema *schema.Schema
	manager := schema.GetManager()
	for _, s := range manager.Schemas() {
		if strings.HasPrefix(schemaPath, s.URL) {
			curSchema = s
			break
		}
	}
	if curSchema == nil || !curSchema.StateVersioning() {
		log.Debug("Monitoring update on unexpected path '%s'", schemaPath)
		return nil
	}
	resourceID := strings.TrimPrefix(schemaPath, curSchema.URL+"/")

	tx, err := dataStore.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	curResource, err := tx.Fetch(curSchema, resourceID, nil)
	if err != nil {
		return err
	}
	resourceState, err := tx.StateFetch(curSchema, resourceID, nil)
	if err != nil {
		return err
	}
	if resourceState.ConfigVersion != resourceState.StateVersion {
		return nil
	}
	var ok bool
	resourceState.Monitoring, ok = response.Data["monitoring"].(string)
	if !ok {
		return fmt.Errorf("No monitoring in state information")
	}

	environmentManager := extension.GetManager()
	environment, haveEnvironment := environmentManager.GetEnvironment(curSchema.ID)
	context := map[string]interface{}{}
	context["resource"] = curResource.Data()
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

func startStateUpdatingProcess(server *Server) {
	stateResponseChan := make(chan *gohan_sync.Event)
	stateStopChan := make(chan bool)

	if _, err := server.sync.Fetch(statePrefix); err != nil {
		server.sync.Update(statePrefix, "{}")
	}

	if _, err := server.sync.Fetch(statePrefix); err == nil {
		server.sync.Update(monitoringPrefix, "{}")
	}

	go func() {
		for server.running {
			err := server.sync.Watch(statePrefix, stateResponseChan, stateStopChan)
			if err != nil {
				log.Error(fmt.Sprintf("sync watch error: %s", err))
			}
			time.Sleep(5 * time.Second)
		}
	}()
	go func() {
		for server.running {
			response := <-stateResponseChan
			err := StateUpdate(response, server)
			if err != nil {
				log.Warning(fmt.Sprintf("error during state update: %s", err))
			}
		}
		stateStopChan <- true
	}()
	monitoringResponseChan := make(chan *gohan_sync.Event)
	monitoringStopChan := make(chan bool)
	go func() {
		for server.running {
			err := server.sync.Watch(monitoringPrefix, monitoringResponseChan, monitoringStopChan)
			if err != nil {
				log.Error(fmt.Sprintf("sync watch error: %s", err))
			}
			time.Sleep(5 * time.Second)
		}
	}()
	go func() {
		for server.running {
			response := <-monitoringResponseChan
			err := MonitoringUpdate(response, server)
			if err != nil {
				log.Warning(fmt.Sprintf("error during state update: %s", err))
			}
		}
		monitoringStopChan <- true
	}()
}

func stopStateUpdatingProcess(server *Server) {
}

//Run extension on sync
func runExtensionOnSync(server *Server, response *gohan_sync.Event, env extension.Environment) {
	lockKey := lockPath + response.Key
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
		"transaction": tx,
		"action":      response.Action,
		"data":        response.Data,
		"key":         response.Key,
	}
	if err != nil {
		return
	}
	if err := env.HandleEvent("notification", context); err != nil {
		log.Warning(fmt.Sprintf("extension error: %s", err))
		return
	}
	err = tx.Commit()
	if err != nil {
		log.Error(fmt.Sprintf("commit error : %s", err))
		return
	}
	return
}

//Sync Watch Process
func startSyncWatchProcess(server *Server) {
	manager := schema.GetManager()
	config := util.GetConfig()
	watch := config.GetStringList("watch/keys", nil)
	events := config.GetStringList("watch/events", nil)
	maxWorkerCount := config.GetParam("watch/worker_count", 0).(int)
	if watch == nil {
		return
	}
	extensions := map[string]extension.Environment{}
	for _, event := range events {
		path := "sync://" + event
		env := newEnvironment(server.db, server.keystoneIdentity)
		err := env.LoadExtensionsForPath(manager.Extensions, path)
		if err != nil {
			log.Fatal(fmt.Sprintf("Extensions parsing error: %v", err))
		}
		extensions[event] = env
	}
	responseChan := make(chan *gohan_sync.Event)
	stopChan := make(chan bool)
	for _, path := range watch {
		go func(path string) {
			for server.running {
				err := server.sync.Watch(path, responseChan, stopChan)
				if err != nil {
					log.Error(fmt.Sprintf("sync watch error: %s", err))
				}
				time.Sleep(5 * time.Second)
			}
		}(path)
	}
	//main response lisnter process
	go func() {
		var wg sync.WaitGroup
		workerCount := 0
		for server.running {
			response := <-responseChan
			wg.Add(1)
			workerCount++
			//spawn workers up to max worker count
			go func() {
				defer func() {
					workerCount--
					wg.Done()
				}()
				for _, event := range events {
					//match extensions
					if strings.HasPrefix(response.Key, "/"+event) {
						env := extensions[event]
						runExtensionOnSync(server, response, env)
						return
					}
				}
			}()
			// Wait if worker pool is full
			if workerCount > maxWorkerCount {
				wg.Wait()
			}
		}
		stopChan <- true
	}()

}

//Stop Watch Process
func stopSyncWatchProcess(server *Server) {
}
