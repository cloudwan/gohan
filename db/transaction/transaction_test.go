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

package transaction_test

import (
	"context"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	tx "github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/db/transaction/mocks"
	"github.com/cloudwan/gohan/schema"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transaction", func() {
	var (
		netSchema *schema.Schema
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("GetTransactionIsolationLevel", func() {
		BeforeEach(func() {
			var exists bool
			manager := schema.GetManager()
			basePath := "../../tests/test_abstract_schema.yaml"
			Expect(manager.LoadSchemaFromFile(basePath)).To(Succeed())

			schemaPath := "../../tests/test_schema.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			netSchema, exists = manager.Schema("network")
			Expect(exists).To(BeTrue())
		})

		It("Defaults to serializable", func() {
			Expect(tx.GetIsolationLevel(netSchema, "create")).To(Equal(tx.Serializable))
		})

		It("Inherits base schema isolation level", func() {
			Expect(tx.GetIsolationLevel(netSchema, "delete")).To(Equal(tx.ReadCommited))
		})

		It("Gets schema overrides", func() {
			Expect(tx.GetIsolationLevel(netSchema, "update")).To(Equal(tx.Serializable))
		})
	})

	Describe("CachedTransactionTest", func() {
		var transx *mocks.MockTransaction
		var cachedTx *sql.CachedTransaction
		var count int
		var countLock int
		var mockCtrl *gomock.Controller

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			transx = mocks.NewMockTransaction(mockCtrl)
			cachedTx = &sql.CachedTransaction{TxInterface: transx, QueryCache: nil}
			cachedTx.ClearCache()

			manager := schema.GetManager()
			basePath := "../../tests/test_abstract_schema.yaml"
			Expect(manager.LoadSchemaFromFile(basePath)).To(Succeed())

			schemaPath := "../../tests/test_schema.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			netSchema, _ = manager.Schema("network")

			countLock = 0
			count = 0
			transx.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *schema.Schema, transaction.Filter, *transaction.ViewOptions, *pagination.Paginator) (list []*schema.Resource, total uint64, err error) {
				count++
				return []*schema.Resource{}, 0, nil
			}).AnyTimes()

			transx.EXPECT().LockList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *schema.Schema, transaction.Filter, *transaction.ViewOptions, *pagination.Paginator, schema.LockPolicy) (list []*schema.Resource, total uint64, err error) {
				countLock++
				return []*schema.Resource{}, 0, nil
			}).AnyTimes()

			transx.EXPECT().Create(gomock.Any(), gomock.Any()).AnyTimes()
			transx.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes()
			transx.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			transx.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			transx.EXPECT().Exec(gomock.Any(), gomock.Any()).AnyTimes()
		})
		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("First list not cached", func() {
			Expect(count).To(Equal(0))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(1))
		})

		It("List calls get cached", func() {
			Expect(count).To(Equal(0))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(1))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(1))
			Expect(countLock).To(Equal(0))
			cachedTx.LockList(ctx, netSchema, tx.Filter{}, nil, nil, schema.LockRelatedResources)
			Expect(count).To(Equal(1))
			Expect(countLock).To(Equal(1))
			cachedTx.LockList(ctx, netSchema, tx.Filter{}, nil, nil, schema.SkipRelatedResources)
			Expect(count).To(Equal(1))
			Expect(countLock).To(Equal(2)) //Lock policy changes hash
		})

		It("Fetch calls get cached", func() {
			Expect(count).To(Equal(0))
			cachedTx.Fetch(ctx, netSchema, tx.Filter{}, nil)
			Expect(count).To(Equal(1))
			cachedTx.Fetch(ctx, netSchema, tx.Filter{}, nil)
			Expect(count).To(Equal(1))
			cachedTx.LockFetch(ctx, netSchema, tx.Filter{}, schema.LockRelatedResources, nil)
			Expect(count).To(Equal(1))
			cachedTx.LockFetch(ctx, netSchema, tx.Filter{}, schema.SkipRelatedResources, nil)
			Expect(count).To(Equal(1))

		})

		It("Updates / inserts clear cache", func() {
			Expect(count).To(Equal(0))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(1))
			cachedTx.Create(ctx, nil)
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(2))
			cachedTx.Update(ctx, nil)
			Expect(count).To(Equal(2))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(3))
			cachedTx.Delete(ctx, nil, nil)
			Expect(count).To(Equal(3))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(4))
			cachedTx.Query(ctx, nil, "", nil)
			Expect(count).To(Equal(4))
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(5))
		})

		It("Exec clears cache", func() {
			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(1))

			cachedTx.Exec(context.Background(), "DELETE ignored FROM not_important")

			cachedTx.List(ctx, netSchema, tx.Filter{}, nil, nil)
			Expect(count).To(Equal(2))
		})
	})
})
