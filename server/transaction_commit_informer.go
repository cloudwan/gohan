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
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"

	"context"

	"sync"

	"github.com/cloudwan/gohan/schema"
)

var (
	transactionCommited     chan int
	transactionCommitedOnce sync.Once
)

func transactionCommitInformer() chan int {
	transactionCommitedOnce.Do(func() {
		transactionCommited = make(chan int, 1)
	})
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

// BeginTx wraps transaction object with sync
func (sw *DbSyncWrapper) BeginTx(ctx context.Context, options *transaction.TxOptions) (transaction.Transaction, error) {
	tx, err := sw.DB.BeginTx(ctx, options)
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

	syncPlain := false
	syncPlainRaw, ok := resource.Schema().Metadata["sync_plain"]
	if ok {
		syncPlainBool, ok := syncPlainRaw.(bool)
		if ok {
			syncPlain = syncPlainBool
		}
	}

	syncProperty := ""
	syncPropertyRaw, ok := resource.Schema().Metadata["sync_property"]
	if ok {
		syncPropertyStr, ok := syncPropertyRaw.(string)
		if ok {
			syncProperty = syncPropertyStr
		}
	}

	if err != nil {
		return fmt.Errorf("Error during event resource deserialisation: %s", err.Error())
	}
	eventResource, err := schema.NewResource(eventSchema, map[string]interface{}{
		"type":          eventType,
		"path":          resource.Path(),
		"version":       version,
		"body":          body,
		"sync_plain":    syncPlain,
		"sync_property": syncProperty,
		"timestamp":     int64(time.Now().Unix()),
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
	state, err := tl.StateFetch(resource.Schema(), transaction.IDFilter(resource.ID()))
	if err != nil {
		return err
	}
	return tl.logEvent("update", resource, state.ConfigVersion)
}

func (tl *transactionEventLogger) Resync(resource *schema.Resource) error {
	if !resource.Schema().StateVersioning() {
		return tl.logEvent("update", resource, 0)
	}
	state, err := tl.StateFetch(resource.Schema(), transaction.IDFilter(resource.ID()))
	if err != nil {
		return err
	}
	return tl.logEvent("update", resource, state.ConfigVersion)
}

func (tl *transactionEventLogger) Delete(s *schema.Schema, resourceID interface{}) error {
	resource, err := tl.Fetch(s, transaction.IDFilter(resourceID), nil)
	if err != nil {
		return err
	}
	configVersion := int64(0)
	if resource.Schema().StateVersioning() {
		state, err := tl.StateFetch(s, transaction.IDFilter(resourceID))
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
