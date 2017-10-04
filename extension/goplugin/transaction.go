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

type cancelableTransaction interface {
	Commit() error
	Close() error
	Closed() bool
	GetIsolationLevel() transaction.Type

	CreateContext(context.Context, *schema.Resource) error
	UpdateContext(context.Context, *schema.Resource) error
	StateUpdateContext(context.Context, *schema.Resource, *transaction.ResourceState) error
	DeleteContext(context.Context, *schema.Schema, interface{}) error
	FetchContext(context.Context, *schema.Schema, transaction.Filter, *transaction.ViewOptions) (*schema.Resource, error)
	LockFetchContext(context.Context, *schema.Schema, transaction.Filter, schema.LockPolicy, *transaction.ViewOptions) (*schema.Resource, error)
	StateFetchContext(context.Context, *schema.Schema, transaction.Filter) (transaction.ResourceState, error)
	ListContext(context.Context, *schema.Schema, transaction.Filter, *transaction.ViewOptions, *pagination.Paginator) ([]*schema.Resource, uint64, error)
	LockListContext(context.Context, *schema.Schema, transaction.Filter, *transaction.ViewOptions, *pagination.Paginator, schema.LockPolicy) ([]*schema.Resource, uint64, error)
	QueryContext(context.Context, *schema.Schema, string, []interface{}) (list []*schema.Resource, err error)
	ExecContext(ctx context.Context, query string, args ...interface{}) error
}

//Transaction is common interface for handling transaction
type Transaction struct {
	tx cancelableTransaction
}

func (t *Transaction) findRawSchema(id string) *schema.Schema {
	manager := schema.GetManager()
	schema, ok := manager.Schema(id)

	if !ok {
		log.Warning(fmt.Sprintf("cannot find schema '%s'", id))
		return nil
	}
	return schema
}

// Create creates a new resource
func (t *Transaction) Create(ctx context.Context, s goext.ISchema, resource map[string]interface{}) error {
	res, err := schema.NewResource(t.findRawSchema(s.ID()), resource)
	if err != nil {
		return err
	}
	return t.tx.CreateContext(ctx, res)
}

// Update updates an existing resource
func (t *Transaction) Update(ctx context.Context, s goext.ISchema, resource map[string]interface{}) error {
	res, err := schema.NewResource(t.findRawSchema(s.ID()), resource)
	if err != nil {
		return err
	}
	return t.tx.UpdateContext(ctx, res)
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
		ConfigVersion: resourceState.ConfigVersion,
		StateVersion:  resourceState.StateVersion,
		Error:         resourceState.Error,
		State:         resourceState.State,
		Monitoring:    resourceState.Monitoring,
	}
}

// StateUpdate updates state of an existing resource
func (t *Transaction) StateUpdate(ctx context.Context, s goext.ISchema, resource map[string]interface{}, resourceState *goext.ResourceState) error {
	res, err := schema.NewResource(t.findRawSchema(s.ID()), resource)
	if err != nil {
		return err
	}
	return t.tx.StateUpdateContext(ctx, res, mapGoExtResourceState(resourceState))
}

// Delete deletes an existing resource
func (t *Transaction) Delete(ctx context.Context, schema goext.ISchema, resourceID interface{}) error {
	return t.tx.DeleteContext(ctx, t.findRawSchema(schema.ID()), resourceID)
}

// Fetch fetches an existing resource
func (t *Transaction) Fetch(ctx context.Context, schema goext.ISchema, filter goext.Filter) (map[string]interface{}, error) {
	res, err := t.tx.FetchContext(ctx, t.findRawSchema(schema.ID()), transaction.Filter(filter), nil)
	if err != nil {
		return nil, err
	}
	return res.Data(), nil
}

// LockFetch locks and fetches an existing resource
func (t *Transaction) LockFetch(ctx context.Context, schema goext.ISchema, filter goext.Filter, lockPolicy goext.LockPolicy) (map[string]interface{}, error) {
	res, err := t.tx.LockFetchContext(ctx, t.findRawSchema(schema.ID()), transaction.Filter(filter), convertLockPolicy(lockPolicy), nil)
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

// StateFetch fetches a state an existing resource
func (t *Transaction) StateFetch(ctx context.Context, schema goext.ISchema, filter goext.Filter) (goext.ResourceState, error) {
	transactionResourceState, err := t.tx.StateFetchContext(ctx, t.findRawSchema(schema.ID()), transaction.Filter(filter))
	if err != nil {
		return goext.ResourceState{}, err
	}
	return mapTransactionResourceState(transactionResourceState), err
}

// List lists existing resources
func (t *Transaction) List(ctx context.Context, schema goext.ISchema, filter goext.Filter, listOptions *goext.ListOptions, paginator *goext.Paginator) ([]map[string]interface{}, uint64, error) {
	schemaID := schema.ID()

	data, _, err := t.tx.ListContext(ctx, t.findRawSchema(schemaID), transaction.Filter(filter), nil, (*pagination.Paginator)(paginator))
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

	data, _, err := t.tx.LockListContext(ctx, t.findRawSchema(schemaID), transaction.Filter(filter), nil, (*pagination.Paginator)(paginator), convertLockPolicy(lockingPolicy))
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

	data, err := t.tx.QueryContext(ctx, t.findRawSchema(schemaID), query, args)
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
	return t.tx.ExecContext(ctx, query, args)
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
