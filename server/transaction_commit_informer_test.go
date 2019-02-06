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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	gohan_sync "github.com/cloudwan/gohan/sync"
	gohan_etcd "github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/cloudwan/gohan/sync/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transaction Commit Informer", func() {
	var (
		ctx      context.Context
		cancel   context.CancelFunc
		done     sync.WaitGroup
		sync     *gohan_etcd.Sync
		mockCtrl *gomock.Controller
		syncedDb db.DB
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

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		// go vet complains about cancel(), but it's called in AfterEach
		ctx, cancel = context.WithCancel(context.Background())

		syncedDb = srv.NewDbSyncWrapper(testDB)

		var err error
		sync, err = gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
		Expect(err).ToNot(HaveOccurred())

		Expect(sync.Delete(ctx, srv.SyncKeyTxCommitted, false)).To(Succeed())
	})

	AfterEach(func() {
		cancel()
		done.Wait()
		sync.Close()

		deleteAllEvents()

		mockCtrl.Finish()
	})

	startInformer := func(sync gohan_sync.Sync) {
		informer := srv.NewTransactionCommitInformer(sync)

		done.Add(1)
		go func() {
			defer GinkgoRecover()
			Expect(informer.Run(ctx, &done)).To(Equal(context.Canceled))
		}()
	}

	shouldReceiveExactlyOnce := func(ch <-chan *gohan_sync.Event) {
		Eventually(ch).Should(Receive())
		Consistently(ch).ShouldNot(Receive())
	}

	syncKeyShouldExist := func(key string) {
		var (
			node *gohan_sync.Node
			err  error
		)
		Eventually(func() error {
			node, err = sync.Fetch(ctx, key)
			return err
		}).Should(Succeed())

		Expect(node).ToNot(BeNil())
	}

	syncKeyShouldNotExist := func(key string) {
		Consistently(func() error {
			_, err := sync.Fetch(ctx, key)
			return err
		}).Should(MatchError(ContainSubstring("Key not found")))
	}

	fetchNewestEventId := func() (id int) {
		eventSchema, _ := schema.GetManager().Schema("event")

		Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
			paginator, err := pagination.NewPaginator(
				pagination.OptionKey(eventSchema, "id"),
				pagination.OptionOrder(pagination.DESC),
				pagination.OptionLimit(1),
			)
			Expect(err).NotTo(HaveOccurred())
			events, count, err := tx.List(ctx, eventSchema, transaction.Filter{}, nil, paginator)
			Expect(count).To(BeNumerically(">", 1))
			id = events[0].Get("id").(int)

			return nil
		})).To(Succeed())

		return
	}

	shouldStoreEventId := func(id int) {
		node, err := sync.Fetch(ctx, srv.SyncKeyTxCommitted)
		Expect(err).NotTo(HaveOccurred())

		result := struct {
			EventId int `json:"event_id"`
		}{}
		Expect(json.Unmarshal([]byte(node.Value), &result)).To(Succeed())

		Expect(result.EventId).To(Equal(id))
	}

	It("should update ETCD key on commit", func() {
		startInformer(sync)

		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
		})

		syncKeyShouldExist(srv.SyncKeyTxCommitted)
	})

	It("should update ETCD key once per transaction", func() {
		startInformer(sync)
		respCh := sync.Watch(ctx, srv.SyncKeyTxCommitted, gohan_sync.RevisionCurrent)

		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
			createNetwork(ctx, tx, "green")
			createNetwork(ctx, tx, "blue")
		})

		shouldReceiveExactlyOnce(respCh)

		shouldStoreEventId(fetchNewestEventId())
	})

	It("should update ETCD key once per a batch of transactions", func() {
		respCh := sync.Watch(ctx, srv.SyncKeyTxCommitted, gohan_sync.RevisionCurrent)

		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
		})
		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "green")
		})
		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "blue")
		})

		startInformer(sync)

		shouldReceiveExactlyOnce(respCh)
	})

	It("should not update ETCD key on rollback", func() {
		startInformer(sync)

		Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
			createNetwork(ctx, tx, "red")
			return fmt.Errorf("test error, should rollback")
		})).NotTo(Succeed())

		syncKeyShouldNotExist(srv.SyncKeyTxCommitted)
	})

	It("should not update ETCD key on non-synced resources", func() {
		startInformer(sync)

		withinTx(func(tx transaction.Transaction) {
			createNotSyncedResource(ctx, tx, "red")
		})

		syncKeyShouldNotExist(srv.SyncKeyTxCommitted)
	})

	It("should retry failed calls", func() {
		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
		})

		mockSync := mock_sync.NewMockSync(mockCtrl)
		failedCall := mockSync.EXPECT().Update(ctx, srv.SyncKeyTxCommitted, gomock.Any()).Return(fmt.Errorf("tested error: etcd update failed"))

		syncUpdated := make(chan struct{}, 1)
		mockSync.EXPECT().Update(ctx, srv.SyncKeyTxCommitted, gomock.Any()).DoAndReturn(func(interface{}, interface{}, interface{}) error {
			syncUpdated <- struct{}{}
			return nil
		}).After(failedCall)

		startInformer(mockSync)

		Eventually(syncUpdated).Should(Receive())
	})

	It("should update ETCD key once per after connection to ETCD restored", func() {
		withinTx(func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "firstResource")
		})

		mockSync := mock_sync.NewMockSync(mockCtrl)
		failedCall := mockSync.EXPECT().Update(ctx, srv.SyncKeyTxCommitted, gomock.Any()).DoAndReturn(func(interface{}, interface{}, interface{}) error {
			withinTx(func(tx transaction.Transaction) {
				createNetwork(ctx, tx, "secondResource")
			})
			return fmt.Errorf("tested error: etcd update failed")
		})

		syncUpdated := make(chan struct{}, 1)
		mockSync.EXPECT().Update(ctx, srv.SyncKeyTxCommitted, gomock.Any()).DoAndReturn(func(interface{}, interface{}, interface{}) error {
			syncUpdated <- struct{}{}
			return nil
		}).After(failedCall)

		startInformer(mockSync)

		Eventually(syncUpdated).Should(Receive())
	})
})
