// Copyright (C) 2019 NTT Innovation Institute, Inc.
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

package server_test

import (
	"context"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext/filter"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transaction Commit Logger", func() {
	var (
		ctx         context.Context
		cancel      context.CancelFunc
		syncedDb    db.DB
		eventSchema *schema.Schema
	)

	withinTx := func(fn func(transaction.Transaction)) {
		Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
			fn(tx)
			return nil
		})).To(Succeed())
	}

	deleteAllEvents := func() {
		withinTx(func(tx transaction.Transaction) {
			Expect(tx.Exec(ctx, "DELETE FROM events")).To(Succeed())
		})
	}

	deleteAllNetworks := func() {
		withinTx(func(tx transaction.Transaction) {
			Expect(tx.Exec(ctx, "DELETE FROM networks")).To(Succeed())
		})
	}

	BeforeEach(func() {
		// go vet complains about cancel(), but it's called in AfterEach
		ctx, cancel = context.WithCancel(context.Background())

		syncedDb = srv.NewDbSyncWrapper(testDB)
		eventSchema, _ = schema.GetManager().Schema("event")

		deleteAllEvents()
		deleteAllNetworks()
	})

	AfterEach(func() {
		deleteAllEvents()
		deleteAllNetworks()
		cancel()
	})

	listEvents := func(tx transaction.Transaction) []*schema.Resource {
		var events []*schema.Resource
		events, _, err := tx.List(ctx, eventSchema, transaction.Filter{}, nil, nil)
		Expect(err).ToNot(HaveOccurred())
		return events
	}

	It("CUD test", func() {
		withinTx(func(tx transaction.Transaction) {
			_, network := createNetwork(ctx, tx, "red")
			Expect(tx.Update(ctx, network)).To(Succeed())
			Expect(tx.Delete(ctx, getSchema("network"), network.ID())).To(Succeed())

			events := listEvents(tx)
			Expect(events).To(HaveLen(3))
			Expect(events[0].Get("type")).To(Equal("create"))
			Expect(events[1].Get("type")).To(Equal("update"))
			Expect(events[2].Get("type")).To(Equal("delete"))

		})
	})

	It("delete filter creates multiple events for each resource", func() {
		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
			createNetwork(ctx, tx, "green")
			createNetwork(ctx, tx, "blue")

			Expect(listEvents(tx)).To(HaveLen(3))

			deleteAllFilter := filter.True()
			Expect(tx.DeleteFilter(ctx, getSchema("network"), deleteAllFilter)).To(Succeed())

			Expect(listEvents(tx)).To(HaveLen(6))
		})
	})

})
