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

func (tx *CachedTransaction) Create(ctx context.Context, resource *schema.Resource) error {
	tx.ClearCache()
	return tx.TxInterface.Create(ctx, resource)
}

func (tx *CachedTransaction) Update(ctx context.Context, resource *schema.Resource) error {
	tx.ClearCache()
	return tx.TxInterface.Update(context.Background(), resource)
}

func (tx *CachedTransaction) StateUpdate(ctx context.Context, resource *schema.Resource, state *transaction.ResourceState) error {
	tx.ClearCache()
	return tx.TxInterface.StateUpdate(context.Background(), resource, state)
}

func (tx *CachedTransaction) Delete(ctx context.Context, s *schema.Schema, resourceID interface{}) error {
	tx.ClearCache()
	return tx.TxInterface.Delete(context.Background(), s, resourceID)
}

//List resources in the db
func (tx *CachedTransaction) List(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator) (list []*schema.Resource, total uint64, err error) {
	sc := listContextHelper(s, filter, options, pg)

	list, total, wasCached, _, err := tx.getCached(s.ID, sc)

	if err != nil {
		return
	}

	if !wasCached {
		metrics.UpdateCounter(1, "tx.%s.cache.miss", s.ID)
		list, total, err = tx.TxInterface.List(ctx, s, filter, options, pg)
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

func (tx *CachedTransaction) LockList(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator, lockPolicy schema.LockPolicy) (list []*schema.Resource, total uint64, err error) {

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

		list, total, err = tx.TxInterface.LockList(ctx, s, filter, options, pg, lockPolicy)
		tx.saveCache(s.ID, sc, list, total, true, err)
	} else {
		metrics.UpdateCounter(1, "tx.%s.cache.hitLock", s.ID)
	}
	return
}

func (tx *CachedTransaction) Query(ctx context.Context, s *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	tx.ClearCache()
	return tx.TxInterface.Query(context.Background(), s, query, arguments)
}

func (tx *CachedTransaction) Fetch(ctx context.Context, s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions) (*schema.Resource, error) {
	list, _, err := tx.List(ctx, s, filter, options, nil)
	return fetchContextHelper(list, err, filter)
}

func (tx *CachedTransaction) LockFetch(ctx context.Context, s *schema.Schema, filter transaction.Filter, lockPolicy schema.LockPolicy, options *transaction.ViewOptions) (*schema.Resource, error) {
	list, _, err := tx.LockList(ctx, s, filter, nil, nil, lockPolicy)
	return lockFetchContextHelper(err, list, filter)
}
