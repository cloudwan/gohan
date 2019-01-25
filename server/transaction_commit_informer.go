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
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	gohan_sync "github.com/cloudwan/gohan/sync"
)

var transactionCommitted = make(chan int64, 1024)

const (
	SyncKeyTxCommitted = "/gohan/cluster/sync/tx_committed"
	badEventId         = -1
)

func NewTransactionCommitInformer(sync gohan_sync.Sync) *TransactionCommitInformer {
	return &TransactionCommitInformer{
		sync: sync,
	}
}

type TransactionCommitInformer struct {
	sync gohan_sync.Sync
}

func (t *TransactionCommitInformer) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	for {
		select {
		case id := <-transactionCommitted:
			t.notify(ctx, id)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *TransactionCommitInformer) notify(ctx context.Context, lastId int64) {
	for {
		lastId = drain(transactionCommitted, lastId)

		err := t.sync.Update(SyncKeyTxCommitted, buildSyncValue(lastId))
		if err == nil {
			return
		}

		log.Error("Failed to notify about committed transaction: %s", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func drain(ch <-chan int64, v int64) int64 {
	for {
		select {
		case v = <-ch:
		default:
			return v
		}
	}
}

func buildSyncValue(id int64) string {
	type syncedEvent struct {
		EventId int64 `json:"event_id"`
	}

	data, err := json.Marshal(syncedEvent{id})
	if err != nil {
		panic(fmt.Sprintf("Can't marshall data: %s", err))
	}

	return string(data)
}

//DbSyncWrapper wraps db.DB so it logs events in database on every transaction.
type DbSyncWrapper struct {
	db db.DB
}

func NewDbSyncWrapper(db db.DB) db.DB {
	return &DbSyncWrapper{db}
}

// BeginTx wraps transaction object with sync
func (sw *DbSyncWrapper) BeginTx(options ...transaction.Option) (transaction.Transaction, error) {
	tx, err := sw.db.BeginTx(options...)
	if err != nil {
		return nil, err
	}
	return syncTransactionWrap(tx), nil
}

func (sw *DbSyncWrapper) Connect(dbType string, conn string, maxOpenConn int) error {
	return sw.db.Connect(dbType, conn, maxOpenConn)
}

func (sw *DbSyncWrapper) Close() {
	sw.db.Close()
}

func (sw *DbSyncWrapper) RegisterTable(s *schema.Schema, cascade, migrate bool) error {
	return sw.db.RegisterTable(s, cascade, migrate)
}

func (sw *DbSyncWrapper) DropTable(s *schema.Schema) error {
	return sw.db.DropTable(s)
}

func (sw *DbSyncWrapper) Options() options.Options {
	return sw.db.Options()
}

type transactionEventLogger struct {
	transaction.Transaction
	eventLogged bool
	lastEventId int64
}

func syncTransactionWrap(tx transaction.Transaction) *transactionEventLogger {
	return &transactionEventLogger{tx, false, badEventId}
}

func (tl *transactionEventLogger) logEvent(ctx context.Context, eventType string, resource *schema.Resource, version int64) error {
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
	eventResource := schema.NewResource(eventSchema, map[string]interface{}{
		"type":          eventType,
		"path":          resource.Path(),
		"version":       version,
		"body":          body,
		"sync_plain":    syncPlain,
		"sync_property": syncProperty,
		"timestamp":     int64(time.Now().Unix()),
	})
	tl.eventLogged = true

	result, err := tl.Transaction.Create(ctx, eventResource)
	if err != nil {
		return err
	}

	tl.lastEventId, err = result.LastInsertId()
	return err
}

func (tl *transactionEventLogger) Create(ctx context.Context, resource *schema.Resource) (transaction.Result, error) {
	result, err := tl.Transaction.Create(ctx, resource)
	if err != nil {
		return nil, err
	}
	if err = tl.logEvent(ctx, "create", resource, 1); err != nil {
		return nil, err
	}

	return result, nil
}

func (tl *transactionEventLogger) Update(ctx context.Context, resource *schema.Resource) error {
	err := tl.Transaction.Update(ctx, resource)
	if err != nil {
		return err
	}
	if !resource.Schema().StateVersioning() {
		return tl.logEvent(ctx, "update", resource, 0)
	}
	state, err := tl.StateFetch(ctx, resource.Schema(), transaction.IDFilter(resource.ID()))
	if err != nil {
		return err
	}
	return tl.logEvent(ctx, "update", resource, state.ConfigVersion)
}

func (tl *transactionEventLogger) Resync(ctx context.Context, resource *schema.Resource) error {
	if !resource.Schema().StateVersioning() {
		return tl.logEvent(context.Background(), "update", resource, 0)
	}
	state, err := tl.StateFetch(ctx, resource.Schema(), transaction.IDFilter(resource.ID()))
	if err != nil {
		return err
	}
	return tl.logEvent(context.Background(), "update", resource, state.ConfigVersion)
}

func (tl *transactionEventLogger) Delete(ctx context.Context, s *schema.Schema, resourceID interface{}) error {
	resource, err := tl.Fetch(ctx, s, transaction.IDFilter(resourceID), nil)
	if err != nil {
		return err
	}
	configVersion := int64(0)
	if resource.Schema().StateVersioning() {
		state, err := tl.StateFetch(ctx, s, transaction.IDFilter(resourceID))
		if err != nil {
			return err
		}
		configVersion = state.ConfigVersion + 1
	}
	err = tl.Transaction.Delete(ctx, s, resourceID)
	if err != nil {
		return err
	}
	return tl.logEvent(ctx, "delete", resource, configVersion)
}

func (tl *transactionEventLogger) Commit() error {
	err := tl.Transaction.Commit()
	if err != nil {
		return err
	}

	tl.triggerSyncWriter()
	return nil
}

func (tl *transactionEventLogger) triggerSyncWriter() {
	if !tl.eventLogged {
		return
	}

	if tl.lastEventId == badEventId {
		panic("logic error, lastEventId not set")
	}

	transactionCommitted <- tl.lastEventId
	metrics.UpdateCounter(1, "event_logger.notified")
}
