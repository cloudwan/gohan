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

package server_test

import (
	"context"
	"encoding/json"
	sync_lib "sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	gohan_sync "github.com/cloudwan/gohan/sync"
	gohan_etcd "github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server package test", func() {
	var (
		ctx                       context.Context
		rawRed, rawBlue, rawGreen map[string]interface{}
		red, blue, green          *schema.Resource
		syncedDb                  db.DB
		sync                      *gohan_etcd.Sync
	)

	BeforeEach(func() {
		ctx = context.Background()
		syncedDb = srv.NewDbSyncWrapper(testDB)

		var err error
		sync, err = gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		sync.Close()
	})

	checkIsSynced := func(rawResource map[string]interface{}, resource *schema.Resource) {
		var writtenConfig *gohan_sync.Node
		Eventually(func() error {
			var err error
			writtenConfig, err = sync.Fetch(ctx, "/config"+resource.Path())
			return err
		}, 1*time.Second).Should(Succeed())

		var configContentsRaw interface{}
		Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
		configContents, ok := configContentsRaw.(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(configContents).To(HaveKeyWithValue("version", float64(1)))
		var configNetworkRaw interface{}
		Expect(json.Unmarshal([]byte(configContents["body"].(string)), &configNetworkRaw)).To(Succeed())
		configNetwork, ok := configNetworkRaw.(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(configNetwork).To(util.MatchAsJSON(rawResource))
	}

	withinTx := func(fn func(transaction.Transaction)) {
		Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
			fn(tx)
			return nil
		})).To(Succeed())
	}

	deleteResource := func(schemaId string, resource *schema.Resource) {
		Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
			schema, _ := schema.GetManager().Schema(schemaId)
			return tx.Delete(ctx, schema, resource.ID())
		})).To(Succeed())
	}

	deleteNetwork := func(network *schema.Resource) {
		deleteResource("network", network)
	}

	Describe("Sync", func() {
		It("should work", func() {
			withinTx(func(tx transaction.Transaction) {
				rawRed, red = createNetwork(ctx, tx, "red")
			})

			writer := srv.NewSyncWriterFromServer(server)
			Expect(writer.Sync(ctx)).To(Equal(1))

			checkIsSynced(rawRed, red)

			deleteNetwork(red)

			Expect(writer.Sync(ctx)).To(Equal(1))

			_, err := sync.Fetch(ctx, red.Path())
			Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
		})

		create := func(schemaId string, rawResource map[string]interface{}) *schema.Resource {
			manager := schema.GetManager()
			resource, err := manager.LoadResource(schemaId, rawResource)
			Expect(err).ToNot(HaveOccurred())

			Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
				_, err := tx.Create(ctx, resource)
				return err
			})).To(Succeed())

			return resource
		}

		Context("With sync_property", func() {
			It("should write only speficied property", func() {
				resource := create("with_sync_property", map[string]interface{}{
					"id": "r0", "p0": "property0",
				})

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync(ctx)).To(Equal(1))

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch(ctx, "/config"+resource.Path())
				Expect(err).ToNot(HaveOccurred())

				var configContentsRaw interface{}
				Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
				configContents, ok := configContentsRaw.(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(configContents).To(HaveKeyWithValue("version", float64(1)))
				var p0If interface{}
				Expect(json.Unmarshal([]byte(configContents["body"].(string)), &p0If)).To(Succeed())
				p0, ok := p0If.(string)
				Expect(ok).To(BeTrue())
				Expect(p0).To(BeEquivalentTo("property0"))

				deleteResource("with_sync_property", resource)

				Expect(writer.Sync(ctx)).To(Equal(1))

				_, err = sync.Fetch(ctx, resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("With sync_plain", func() {
			It("should write data without marshaling", func() {
				resource := create("with_sync_plain", map[string]interface{}{
					"id": "r0", "p0": "property0",
				})

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync(ctx)).To(Equal(1))

				writtenConfig, err := sync.Fetch(ctx, "/config"+resource.Path())
				Expect(err).ToNot(HaveOccurred())

				var configContentsRaw map[string]interface{}
				Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
				Expect(configContentsRaw).To(HaveKeyWithValue("id", "r0"))
				Expect(configContentsRaw).To(HaveKeyWithValue("p0", "property0"))

				deleteResource("with_sync_plain", resource)

				Expect(writer.Sync(ctx)).To(Equal(1))

				_, err = sync.Fetch(ctx, resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("With sync_plain and sync_property in string", func() {
			It("should write data without marshaling", func() {
				resource := create("with_sync_plain_string", map[string]interface{}{
					"id": "r0", "p0": "property0",
				})

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync(ctx)).To(Equal(1))

				writtenConfig, err := sync.Fetch(ctx, "/config"+resource.Path())
				Expect(err).ToNot(HaveOccurred())
				Expect(writtenConfig.Value).To(BeEquivalentTo("property0"))

				deleteResource("with_sync_plain_string", resource)

				Expect(writer.Sync(ctx)).To(Equal(1))

				_, err = sync.Fetch(ctx, resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("With sync_skip_config_prefix and sync_key_template", func() {
			It("should write data to and to exact path specified in template ", func() {
				resource := create("with_sync_skip_config_prefix", map[string]interface{}{
					"id": "abcdef", "p0": "property0",
				})

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync(ctx)).To(Equal(1))

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch(ctx, "/prefix/abcdef")
				Expect(err).ToNot(HaveOccurred())
				Expect(writtenConfig.Value).To(BeEquivalentTo("property0"))

				deleteResource("with_sync_skip_config_prefix", resource)

				Expect(writer.Sync(ctx)).To(Equal(1))

				_, err = sync.Fetch(ctx, resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

	})

	Describe("Interactions with TransactionCommitInformer", func() {
		var (
			cancel context.CancelFunc
			done   sync_lib.WaitGroup
		)

		startInformer := func() {
			informer := srv.NewTransactionCommitInformer(sync)

			done.Add(1)
			go func() {
				defer GinkgoRecover()
				Expect(informer.Run(ctx, &done)).To(Equal(context.Canceled))
			}()
		}

		startWriter := func() {
			writer := srv.NewSyncWriterFromServer(server)

			done.Add(1)
			go func() {
				defer GinkgoRecover()
				Expect(writer.Run(ctx, &done)).To(Equal(context.Canceled))
			}()
		}

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(ctx)

			startWriter()
			startInformer()
		})

		AfterEach(func() {
			deleteNetwork(red)
			deleteNetwork(green)
			deleteNetwork(blue)

			cancel()
			done.Wait()
		})

		It("writes all events from a single transaction", func() {
			withinTx(func(tx transaction.Transaction) {
				rawRed, red = createNetwork(ctx, tx, "red")
				rawBlue, blue = createNetwork(ctx, tx, "blue")
				rawGreen, green = createNetwork(ctx, tx, "green")
			})

			checkIsSynced(rawRed, red)
			checkIsSynced(rawBlue, blue)
			checkIsSynced(rawGreen, green)
		})

		It("writes all events from many transactions", func() {
			withinTx(func(tx transaction.Transaction) {
				rawRed, red = createNetwork(ctx, tx, "red")
			})

			withinTx(func(tx transaction.Transaction) {
				rawBlue, blue = createNetwork(ctx, tx, "blue")
			})

			withinTx(func(tx transaction.Transaction) {
				rawGreen, green = createNetwork(ctx, tx, "green")
			})

			checkIsSynced(rawRed, red)
			checkIsSynced(rawBlue, blue)
			checkIsSynced(rawGreen, green)
		})
	})
})
