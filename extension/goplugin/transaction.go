// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package goplugin

import (
	"context"
	"fmt"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
)

//Transaction is common interface for handling transaction
type Transaction struct {
	tx transaction.Transaction
}

func (t *Transaction) findRawSchema(id goext.SchemaID) *schema.Schema {
	manager := schema.GetManager()
	schema, ok := manager.Schema(string(id))

	if !ok {
		log.Warning(fmt.Sprintf("cannot find schema '%s'", id))
		return nil
	}
	return schema
}

// Create creates a new resource
func (t *Transaction) Create(ctx context.Context, s goext.ISchema, resource map[string]interface{}) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	res := schema.NewResource(t.findRawSchema(s.ID()), resource)

	// use context.Background to avoid cancellation mid-query for all Queries/Exec
	// ESI-16552 context cancellation tends to break next Begin
	// It doesn't work in mysql driver anyway https://github.com/go-sql-driver/mysql/issues/731
	_, err := t.tx.Create(context.Background(), res)
	return err
}

// Update updates an existing resource
func (t *Transaction) Update(ctx context.Context, s goext.ISchema, resource map[string]interface{}) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	res := schema.NewResource(t.findRawSchema(s.ID()), resource)
	return t.tx.Update(context.Background(), res)
}

func mapGoExtResourceState(resourceState *goext.ResourceState) *transaction.ResourceState {
	if resourceState == nil {
		return nil
	}
	return &transaction.ResourceState{
		ConfigVersion: resourceState.ConfigVersion,
		StateVersion:  resourceState.StateVersion,
		Error:         resourceState.Error,
		State:         resourceState.State,
		Monitoring:    resourceState.Monitoring,
	}
}

func mapTransactionResourceState(resourceState transaction.ResourceState) goext.ResourceState {
	return goext.ResourceState{
		ID:            resourceState.ID,
		ConfigVersion: resourceState.ConfigVersion,
		StateVersion:  resourceState.StateVersion,
		Error:         resourceState.Error,
		State:         resourceState.State,
		Monitoring:    resourceState.Monitoring,
	}
}

// StateUpdate updates state of an existing resource
func (t *Transaction) StateUpdate(ctx context.Context, s goext.ISchema, resource map[string]interface{}, resourceState *goext.ResourceState) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	res := schema.NewResource(t.findRawSchema(s.ID()), resource)
	return t.tx.StateUpdate(context.Background(), res, mapGoExtResourceState(resourceState))
}

// Delete deletes an existing resource
func (t *Transaction) Delete(ctx context.Context, schema goext.ISchema, resourceID interface{}) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	return t.tx.Delete(context.Background(), t.findRawSchema(schema.ID()), resourceID)
}

func (t *Transaction) DeleteFilter(ctx context.Context, schema goext.ISchema, filter goext.Filter) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	return t.tx.DeleteFilter(context.Background(), t.findRawSchema(schema.ID()), transaction.Filter(filter))
}

func (t *Transaction) StateList(ctx context.Context, schema goext.ISchema, filter goext.Filter) ([]goext.ResourceState, error) {
	if err := ctx.Err(); err != nil {
		return nil, ctx.Err()
	}
	rawResourceStates, err := t.tx.StateList(context.Background(), t.findRawSchema(schema.ID()), transaction.Filter(filter))
	if err != nil {
		return nil, err
	}
	resourceStates := make([]goext.ResourceState, 0, len(rawResourceStates))
	for _, rawResourceState := range rawResourceStates {
		resourceStates = append(resourceStates, mapTransactionResourceState(rawResourceState))
	}
	return resourceStates, nil
}

// Fetch fetches an existing resource
func (t *Transaction) Fetch(ctx context.Context, schema goext.ISchema, filter goext.Filter) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, ctx.Err()
	}
	res, err := t.tx.Fetch(context.Background(), t.findRawSchema(schema.ID()), transaction.Filter(filter), nil)
	if err != nil {
		return nil, err
	}
	return res.Data(), nil
}

// LockFetch locks and fetches an existing resource
func (t *Transaction) LockFetch(ctx context.Context, schema goext.ISchema, filter goext.Filter, lockPolicy goext.LockPolicy) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, ctx.Err()
	}
	res, err := t.tx.LockFetch(context.Background(), t.findRawSchema(schema.ID()), transaction.Filter(filter), convertLockPolicy(lockPolicy), nil)
	if err != nil {
		return nil, err
	}
	return res.Data(), nil
}

func convertLockPolicy(policy goext.LockPolicy) schema.LockPolicy {
	switch policy {
	case goext.SkipRelatedResources:
		return schema.SkipRelatedResources
	case goext.LockRelatedResources:
		return schema.LockRelatedResources
	case goext.NoLock:
		return schema.NoLocking
	default:
		panic(fmt.Sprintf("Unknown lock policy: %d", policy))
	}
}

// StateFetch fetches the state of an existing resource
func (t *Transaction) StateFetch(ctx context.Context, schema goext.ISchema, filter goext.Filter) (goext.ResourceState, error) {
	if err := ctx.Err(); err != nil {
		return goext.ResourceState{}, ctx.Err()
	}
	transactionResourceState, err := t.tx.StateFetch(context.Background(), t.findRawSchema(schema.ID()), transaction.Filter(filter))
	if err != nil {
		return goext.ResourceState{}, err
	}
	return mapTransactionResourceState(transactionResourceState), err
}

// List lists existing resources
func (t *Transaction) List(ctx context.Context, schema goext.ISchema, filter goext.Filter, listOptions *goext.ListOptions, paginator *goext.Paginator) ([]map[string]interface{}, uint64, error) {
	schemaID := schema.ID()

	if err := ctx.Err(); err != nil {
		return nil, 0, ctx.Err()
	}
	var o *transaction.ViewOptions
	if listOptions != nil {
		o = &transaction.ViewOptions{
			Details: listOptions.Details,
			Fields: listOptions.Fields,
		}
	}
	data, _, err := t.tx.List(context.Background(), t.findRawSchema(schemaID), transaction.Filter(filter), o, (*pagination.Paginator)(paginator))
	if err != nil {
		return nil, 0, err
	}

	return resourcesToMap(data)
}

func resourcesToMap(data []*schema.Resource) ([]map[string]interface{}, uint64, error) {
	resourceProperties := make([]map[string]interface{}, len(data))
	for i := range data {
		resourceProperties[i] = data[i].Data()
	}

	return resourceProperties, uint64(len(resourceProperties)), nil
}

// LockList locks and lists existing resources
func (t *Transaction) LockList(ctx context.Context, schema goext.ISchema, filter goext.Filter, listOptions *goext.ListOptions, paginator *goext.Paginator, lockingPolicy goext.LockPolicy) ([]map[string]interface{}, uint64, error) {
	schemaID := schema.ID()

	if err := ctx.Err(); err != nil {
		return nil, 0, ctx.Err()
	}
	data, _, err := t.tx.LockList(context.Background(), t.findRawSchema(schemaID), transaction.Filter(filter), nil, (*pagination.Paginator)(paginator), convertLockPolicy(lockingPolicy))
	if err != nil {
		return nil, 0, err
	}

	return resourcesToMap(data)
}

// RawTransaction returns the raw transaction
func (t *Transaction) RawTransaction() interface{} {
	return t.tx
}

// Query executes a query
func (t *Transaction) Query(ctx context.Context, schema goext.ISchema, query string, args []interface{}) (list []map[string]interface{}, err error) {
	schemaID := schema.ID()

	if err := ctx.Err(); err != nil {
		return nil, ctx.Err()
	}
	data, err := t.tx.Query(context.Background(), t.findRawSchema(schemaID), query, args)
	if err != nil {
		return nil, err
	}

	resourceProperties := make([]map[string]interface{}, len(data))
	for i := range data {
		resourceProperties[i] = data[i].Data()
	}

	return resourceProperties, nil
}

// Commit performs a commit of the transaction
func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

// Exec performs an exec in transaction
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	return t.tx.Exec(context.Background(), query, args...)
}

// Close closes the transaction
func (t *Transaction) Close() error {
	return t.tx.Close()
}

// Closed return whether the transaction is closed
func (t *Transaction) Closed() bool {
	return t.tx.Closed()
}

// GetIsolationLevel returns the isolation level of the transaction
func (t *Transaction) GetIsolationLevel() goext.Type {
	return goext.Type(t.tx.GetIsolationLevel())
}

func (t *Transaction) Count(ctx context.Context, schema goext.ISchema, filter goext.Filter) (uint64, error) {
	schemaID := schema.ID()

	if err := ctx.Err(); err != nil {
		return 0, ctx.Err()
	}
	return t.tx.Count(context.Background(), t.findRawSchema(schemaID), transaction.Filter(filter))
}
