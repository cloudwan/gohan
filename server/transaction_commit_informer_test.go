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
	"fmt"
	"sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
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
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		ctx, cancel = context.WithCancel(context.Background())

		var err error
		sync, err = gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
		Expect(err).ToNot(HaveOccurred())

		Expect(sync.Delete(srv.SyncKeyTxCommitted, false)).To(Succeed())
	})

	AfterEach(func() {
		cancel()
		done.Wait()
		sync.Close()
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

	withinTx := func(database db.DB, fn func(transaction.Transaction)) {
		Expect(db.WithinTx(database, func(tx transaction.Transaction) error {
			fn(tx)
			return nil
		})).To(Succeed())
	}

	syncKeyShouldExist := func(key string) {
		var (
			node *gohan_sync.Node
			err  error
		)
		Eventually(func() error {
			node, err = sync.Fetch(key)
			return err
		}).Should(Succeed())

		Expect(node).ToNot(BeNil())
	}

	syncKeyShouldNotExist := func(key string) {
		Consistently(func() error {
			_, err := sync.Fetch(key)
			return err
		}).Should(MatchError(ContainSubstring("Key not found")))
	}

	It("should update ETCD key on commit", func() {
		startInformer(sync)

		syncedDb := srv.NewDbSyncWrapper(testDB)
		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
		})

		syncKeyShouldExist(srv.SyncKeyTxCommitted)
	})

	It("should update ETCD key once per transaction", func() {
		startInformer(sync)
		respCh := sync.WatchContext(ctx, srv.SyncKeyTxCommitted, gohan_sync.RevisionCurrent)

		syncedDb := srv.NewDbSyncWrapper(testDB)

		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
			createNetwork(ctx, tx, "green")
			createNetwork(ctx, tx, "blue")
		})

		shouldReceiveExactlyOnce(respCh)
	})

	It("should update ETCD key once per a batch of transactions", func() {
		respCh := sync.WatchContext(ctx, srv.SyncKeyTxCommitted, gohan_sync.RevisionCurrent)

		syncedDb := srv.NewDbSyncWrapper(testDB)

		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
		})
		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "green")
		})
		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "blue")
		})

		startInformer(sync)

		shouldReceiveExactlyOnce(respCh)
	})

	It("should not update ETCD key on rollback", func() {
		startInformer(sync)

		syncedDb := srv.NewDbSyncWrapper(testDB)

		Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
			createNetwork(ctx, tx, "red")
			return fmt.Errorf("test error, should rollback")
		})).NotTo(Succeed())

		syncKeyShouldNotExist(srv.SyncKeyTxCommitted)
	})

	It("should not update ETCD key on non-synced resources", func() {
		startInformer(sync)

		syncedDb := srv.NewDbSyncWrapper(testDB)

		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNotSyncedResource(ctx, tx, "red")
		})

		syncKeyShouldNotExist(srv.SyncKeyTxCommitted)
	})

	It("should retry failed calls", func() {
		syncedDb := srv.NewDbSyncWrapper(testDB)

		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "red")
		})

		mockSync := mock_sync.NewMockSync(mockCtrl)
		failedCall := mockSync.EXPECT().Update(srv.SyncKeyTxCommitted, gomock.Any()).Return(fmt.Errorf("etcd update failed"))

		syncUpdated := make(chan struct{}, 1)
		mockSync.EXPECT().Update(srv.SyncKeyTxCommitted, gomock.Any()).DoAndReturn(func(interface{}, interface{}) error {
			syncUpdated <- struct{}{}
			return nil
		}).After(failedCall)

		startInformer(mockSync)

		Eventually(syncUpdated).Should(Receive())
	})

	It("should update ETCD key once per after connection to ETCD restored", func() {
		syncedDb := srv.NewDbSyncWrapper(testDB)

		withinTx(syncedDb, func(tx transaction.Transaction) {
			createNetwork(ctx, tx, "firstResource")
		})

		mockSync := mock_sync.NewMockSync(mockCtrl)
		failedCall := mockSync.EXPECT().Update(srv.SyncKeyTxCommitted, gomock.Any()).DoAndReturn(func(interface{}, interface{}) error {
			withinTx(syncedDb, func(tx transaction.Transaction) {
				createNetwork(ctx, tx, "secondResource")
			})
			return fmt.Errorf("etcd update failed")
		})

		syncUpdated := make(chan struct{}, 1)
		mockSync.EXPECT().Update(srv.SyncKeyTxCommitted, gomock.Any()).DoAndReturn(func(interface{}, interface{}) error {
			syncUpdated <- struct{}{}
			return nil
		}).After(failedCall)

		startInformer(mockSync)

		Eventually(syncUpdated).Should(Receive())
	})
})
