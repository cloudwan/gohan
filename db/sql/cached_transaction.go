// Copyright (C) 2018 NTT Innovation Institute, Inc.
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

package sql

import (
	"context"
	"fmt"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	// Import mysql lib
	_ "github.com/go-sql-driver/mysql"
	// Import go-sqlite3 lib
	_ "github.com/mattn/go-sqlite3"
	// Import go-fakedb lib
	"github.com/cloudwan/gohan/metrics"
	"github.com/mitchellh/hashstructure"
	_ "github.com/nati/go-fakedb"
)

type CachedState struct {
	list     []*schema.Resource
	total    uint64
	isLocked bool
}

type CachedTransaction struct {
	TxInterface
	QueryCache map[string]CachedState
}

func MakeCachedTransaction(transx TxInterface) TxInterface {
	cachedTransaction := &CachedTransaction{transx, nil}
	cachedTransaction.ClearCache()
	return cachedTransaction
}

func (tx *CachedTransaction) Create(resource *schema.Resource) error {
	return tx.CreateContext(context.Background(), resource)
}

func (tx *CachedTransaction) CreateContext(ctx context.Context, resource *schema.Resource) error {
	tx.ClearCache()
	return tx.TxInterface.CreateContext(ctx, resource)
}

func (tx *CachedTransaction) Update(resource *schema.Resource) error {
	return tx.UpdateContext(context.Background(), resource)
}

func (tx *CachedTransaction) UpdateContext(ctx context.Context, resource *schema.Resource) error {
	tx.ClearCache()
	return tx.TxInterface.UpdateContext(context.Background(), resource)
}

func (tx *CachedTransaction) StateUpdate(resource *schema.Resource, state *transaction.ResourceState) error {
	return tx.StateUpdateContext(context.Background(), resource, state)
}

func (tx *CachedTransaction) StateUpdateContext(ctx context.Context, resource *schema.Resource, state *transaction.ResourceState) error {
	tx.ClearCache()
	return tx.TxInterface.StateUpdateContext(context.Background(), resource, state)
}

func (tx *CachedTransaction) Delete(s *schema.Schema, resourceID interface{}) error {
	return tx.DeleteContext(context.Background(), s, resourceID)
}

func (tx *CachedTransaction) DeleteContext(ctx context.Context, s *schema.Schema, resourceID interface{}) error {
	tx.ClearCache()
	return tx.TxInterface.DeleteContext(context.Background(), s, resourceID)
}

func (tx *CachedTransaction) List(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator) (list []*schema.Resource, total uint64, err error) {
	return tx.ListContext(context.Background(), s, filter, options, pg)
}

//List resources in the db
func (tx *CachedTransaction) ListContext(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator) (list []*schema.Resource, total uint64, err error) {
	sc := listContextHelper(s, filter, options, pg)

	list, total, wasCached, _, err := tx.getCached(s.ID, sc)

	if err != nil {
		return
	}

	if !wasCached {
		metrics.UpdateCounter(1, "tx.%s.cache.miss", s.ID)
		list, total, err = tx.TxInterface.ListContext(ctx, s, filter, options, pg)
		tx.saveCache(s.ID, sc, list, total, false, err)
	} else {
		metrics.UpdateCounter(1, "tx.%s.cache.hit", s.ID)
	}
	return
}

func (tx *CachedTransaction) createKey(schemaID string, sc *selectContext) (string, error) {
	filterHash, err := hashstructure.Hash(sc.filter, nil)
	if err != nil {
		return "", err
	}
	res := fmt.Sprintf("%s %v %v %v", schemaID, sc.join, filterHash, sc.fields)
	if sc.paginator != nil {
		res = fmt.Sprintf("%s %v", res, *sc.paginator)
	}
	return res, err
}

func (tx *CachedTransaction) getCached(schemeID string, sc *selectContext) (list []*schema.Resource, total uint64, wasCached bool, wasLocked bool, err error) {
	key, err := tx.createKey(schemeID, sc)
	if err != nil {
		return nil, 0, false, false, err
	}

	res, present := tx.QueryCache[key]
	if present {
		return res.list, res.total, true, res.isLocked, nil
	} else {
		return nil, 0, false, false, nil
	}
}

func (tx *CachedTransaction) saveCache(schemeID string, sc *selectContext, list []*schema.Resource, total uint64, wasLocked bool, errList error) error {
	if errList != nil {
		return nil
	}
	key, err := tx.createKey(schemeID, sc)
	if err != nil {
		return err
	}
	tx.QueryCache[key] = CachedState{list, total, wasLocked}
	return nil
}

func (tx *CachedTransaction) ClearCache() {
	tx.QueryCache = make(map[string]CachedState)
}

func (tx *CachedTransaction) LockList(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator, lockPolicy schema.LockPolicy) (list []*schema.Resource, total uint64, err error) {
	return tx.LockListContext(context.Background(), s, filter, options, pg, lockPolicy)
}

func (tx *CachedTransaction) LockListContext(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator, lockPolicy schema.LockPolicy) (list []*schema.Resource, total uint64, err error) {

	sc := lockListContextHelper(s, filter, options, pg, lockPolicy)

	list, total, wasCached, wasLocked, err := tx.getCached(s.ID, sc)
	if err != nil {
		return
	}

	if !wasCached || !wasLocked {
		if wasCached {
			metrics.UpdateCounter(1, "tx.%s.cache.notLocked", s.ID)
		}
		metrics.UpdateCounter(1, "tx.%s.cache.missLock", s.ID)

		list, total, err = tx.TxInterface.LockListContext(ctx, s, filter, options, pg, lockPolicy)
		tx.saveCache(s.ID, sc, list, total, true, err)
	} else {
		metrics.UpdateCounter(1, "tx.%s.cache.hitLock", s.ID)
	}
	return
}

func (tx *CachedTransaction) Query(s *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	return tx.QueryContext(context.Background(), s, query, arguments)
}

func (tx *CachedTransaction) QueryContext(ctx context.Context, s *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	tx.ClearCache()
	return tx.TxInterface.QueryContext(context.Background(), s, query, arguments)
}

func (tx *CachedTransaction) Fetch(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions) (*schema.Resource, error) {
	return tx.FetchContext(context.Background(), s, filter, options)
}

func (tx *CachedTransaction) FetchContext(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions) (*schema.Resource, error) {
	list, _, err := tx.ListContext(ctx, s, filter, options, nil)
	return fetchContextHelper(list, err, filter)
}

func (tx *CachedTransaction) LockFetch(s *schema.Schema, filter transaction.Filter, lockPolicy schema.LockPolicy, options *transaction.ViewOptions) (*schema.Resource, error) {
	return tx.LockFetchContext(context.Background(), s, filter, lockPolicy, options)
}

func (tx *CachedTransaction) LockFetchContext(ctx context.Context, s *schema.Schema, filter transaction.Filter, lockPolicy schema.LockPolicy, options *transaction.ViewOptions) (*schema.Resource, error) {
	list, _, err := tx.LockListContext(ctx, s, filter, nil, nil, lockPolicy)
	return lockFetchContextHelper(err, list, filter)
}
