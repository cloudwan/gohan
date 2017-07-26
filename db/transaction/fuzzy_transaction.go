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

package transaction

import (
	"fmt"
	"math/rand"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx"
)

// FuzzyTransaction is a fuzzer (a decorator) which returns either the underlying value or a deadlock
type FuzzyTransaction struct {
	Tx Transaction
}

func (ft *FuzzyTransaction) fuzzIt(fn func() error) error {
	if rand.Int()&1 == 0 {
		return fmt.Errorf("database is locked")
	}
	return fn()
}

// Create creates a new resource
func (ft *FuzzyTransaction) Create(resource *schema.Resource) error {
	return ft.fuzzIt(func() error { return ft.Tx.Create(resource) })
}

// Update updates an existing resource
func (ft *FuzzyTransaction) Update(resource *schema.Resource) error {
	return ft.fuzzIt(func() error { return ft.Tx.Update(resource) })
}

// StateUpdate updates a state
func (ft *FuzzyTransaction) StateUpdate(resource *schema.Resource, state *ResourceState) error {
	return ft.fuzzIt(func() error { return ft.Tx.StateUpdate(resource, state) })
}

// Delete deletes a resource
func (ft *FuzzyTransaction) Delete(s *schema.Schema, resourceID interface{}) error {
	return ft.fuzzIt(func() error { return ft.Tx.Delete(s, resourceID) })
}

// Fetch fetches a resource
func (ft *FuzzyTransaction) Fetch(s *schema.Schema, filter Filter) (*schema.Resource, error) {
	var outResource *schema.Resource
	return outResource, ft.fuzzIt(func() error {
		var err error
		outResource, err = ft.Tx.Fetch(s, filter)
		return err
	})
}

// LockFetch locks and fetches a resource
func (ft *FuzzyTransaction) LockFetch(s *schema.Schema, filter Filter, lockPolicy schema.LockPolicy) (*schema.Resource, error) {
	var outResource *schema.Resource
	return outResource, ft.fuzzIt(func() error {
		var err error
		outResource, err = ft.Tx.LockFetch(s, filter, lockPolicy)
		return err
	})
}

// StateFetch fetches a state
func (ft *FuzzyTransaction) StateFetch(s *schema.Schema, filter Filter) (ResourceState, error) {
	var outResourceState ResourceState
	return outResourceState, ft.fuzzIt(func() error {
		var err error
		outResourceState, err = ft.Tx.StateFetch(s, filter)
		return err
	})
}

// List lists resources
func (ft *FuzzyTransaction) List(s *schema.Schema, filter Filter, options *ListOptions, pagination *pagination.Paginator) ([]*schema.Resource, uint64, error) {
	var outResources []*schema.Resource
	var outCount uint64
	return outResources, outCount, ft.fuzzIt(func() error {
		var err error
		outResources, outCount, err = ft.Tx.List(s, filter, options, pagination)
		return err
	})
}

// LockList locks and lists resources
func (ft *FuzzyTransaction) LockList(s *schema.Schema, filter Filter, options *ListOptions, pagination *pagination.Paginator, lockPolicy schema.LockPolicy) ([]*schema.Resource, uint64, error) {
	var outResources []*schema.Resource
	var outCount uint64
	return outResources, outCount, ft.fuzzIt(func() error {
		var err error
		outResources, outCount, err = ft.Tx.LockList(s, filter, options, pagination, lockPolicy)
		return err
	})
}

// RawTransaction returns a raw sqlx transaction
func (ft *FuzzyTransaction) RawTransaction() *sqlx.Tx {
	return ft.Tx.RawTransaction()
}

// Query executes a query for a schema
func (ft *FuzzyTransaction) Query(s *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	var outResources []*schema.Resource
	return outResources, ft.fuzzIt(func() error {
		var err error
		outResources, err = ft.Tx.Query(s, query, arguments)
		return err
	})
}

// Commit commits the transaction
func (ft *FuzzyTransaction) Commit() error {
	return ft.fuzzIt(func() error { return ft.Tx.Commit() })
}

// Exec executes a query
func (ft *FuzzyTransaction) Exec(query string, args ...interface{}) error {
	return ft.fuzzIt(func() error { return ft.Tx.Exec(query, args) })
}

// Close closes the transaction
func (ft *FuzzyTransaction) Close() error {
	return ft.Tx.Close()
}

// Closed returns whether the transaction is closec
func (ft *FuzzyTransaction) Closed() bool {
	return ft.Tx.Closed()
}

// GetIsolationLevel returns the current isolation level
func (ft *FuzzyTransaction) GetIsolationLevel() Type {
	return ft.Tx.GetIsolationLevel()
}
