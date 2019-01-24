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
	"github.com/twinj/uuid"
)

var _ = Describe("Server package test", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Sync", func() {
		It("should work", func() {
			manager := schema.GetManager()
			networkSchema, _ := manager.Schema("network")
			network := getNetwork("Red", "red")
			networkResource, err := manager.LoadResource("network", network)
			Expect(err).ToNot(HaveOccurred())
			testDB1 := srv.NewDbSyncWrapper(testDB)
			tx, err := testDB1.BeginTx()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Create(ctx, networkResource)).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			writer := srv.NewSyncWriterFromServer(server)
			Expect(writer.Sync()).To(Equal(1))

			sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
			Expect(err).ToNot(HaveOccurred())

			writtenConfig, err := sync.Fetch("/config" + networkResource.Path())
			Expect(err).ToNot(HaveOccurred())

			var configContentsRaw interface{}
			Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
			configContents, ok := configContentsRaw.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(configContents).To(HaveKeyWithValue("version", float64(1)))
			var configNetworkRaw interface{}
			Expect(json.Unmarshal([]byte(configContents["body"].(string)), &configNetworkRaw)).To(Succeed())
			configNetwork, ok := configNetworkRaw.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(configNetwork).To(util.MatchAsJSON(network))

			tx, err = testDB1.BeginTx()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Delete(ctx, networkSchema, networkResource.ID())).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			Expect(writer.Sync()).To(Equal(1))

			_, err = sync.Fetch(networkResource.Path())
			Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
		})

		Context("With sync_property", func() {
			It("should write only speficied property", func() {
				manager := schema.GetManager()
				schema, _ := manager.Schema("with_sync_property")
				resource, err := manager.LoadResource(
					"with_sync_property", map[string]interface{}{
						"id": "r0", "p0": "property0",
					})
				Expect(err).ToNot(HaveOccurred())
				testDB1 := srv.NewDbSyncWrapper(testDB)
				tx, err := testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(ctx, resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Equal(1))

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch("/config" + resource.Path())
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

				tx, err = testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(ctx, schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Equal(1))

				_, err = sync.Fetch(resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("With sync_plain", func() {
			It("should write data without marshaling", func() {
				manager := schema.GetManager()
				schema, _ := manager.Schema("with_sync_plain")
				resource, err := manager.LoadResource(
					"with_sync_plain", map[string]interface{}{
						"id": "r0", "p0": "property0",
					})
				Expect(err).ToNot(HaveOccurred())
				testDB1 := srv.NewDbSyncWrapper(testDB)
				tx, err := testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(ctx, resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Equal(1))

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch("/config" + resource.Path())
				Expect(err).ToNot(HaveOccurred())

				var configContentsRaw map[string]interface{}
				Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
				Expect(configContentsRaw).To(HaveKeyWithValue("id", "r0"))
				Expect(configContentsRaw).To(HaveKeyWithValue("p0", "property0"))

				tx, err = testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(ctx, schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Equal(1))

				_, err = sync.Fetch(resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("With sync_plain and sync_property in string", func() {
			It("should write data without marshaling", func() {
				manager := schema.GetManager()
				schema, _ := manager.Schema("with_sync_plain_string")
				resource, err := manager.LoadResource(
					"with_sync_plain_string", map[string]interface{}{
						"id": "r0", "p0": "property0",
					})
				Expect(err).ToNot(HaveOccurred())
				testDB1 := srv.NewDbSyncWrapper(testDB)
				tx, err := testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(ctx, resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Equal(1))

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch("/config" + resource.Path())
				Expect(err).ToNot(HaveOccurred())
				Expect(writtenConfig.Value).To(BeEquivalentTo("property0"))

				tx, err = testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(ctx, schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Equal(1))

				_, err = sync.Fetch(resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("With sync_skip_config_prefix and sync_key_template", func() {
			It("should write data to and to exact path specified in template ", func() {
				manager := schema.GetManager()
				schema, _ := manager.Schema("with_sync_skip_config_prefix")
				resource, err := manager.LoadResource(
					"with_sync_skip_config_prefix", map[string]interface{}{
						"id": "abcdef", "p0": "property0",
					})
				Expect(err).ToNot(HaveOccurred())
				testDB1 := srv.NewDbSyncWrapper(testDB)
				tx, err := testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(ctx, resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Equal(1))

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch("/prefix/abcdef")
				Expect(err).ToNot(HaveOccurred())
				Expect(writtenConfig.Value).To(BeEquivalentTo("property0"))

				tx, err = testDB1.BeginTx()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(ctx, schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Equal(1))

				_, err = sync.Fetch(resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})

		Context("many events", func() {
			var (
				cancel   context.CancelFunc
				writer   *srv.SyncWriter
				sync     *gohan_etcd.Sync
				syncedDb db.DB
				done     sync_lib.WaitGroup

				rawRed, rawBlue, rawGreen map[string]interface{}
				red, blue, green          *schema.Resource
			)

			BeforeEach(func() {
				ctx, cancel = context.WithCancel(ctx)

				done.Add(1)
				writer = srv.NewSyncWriterFromServer(server)
				go func() {
					defer GinkgoRecover()
					Expect(writer.Run(ctx, &done)).To(Equal(context.Canceled))
				}()

				var err error
				sync, err = gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				syncedDb = srv.NewDbSyncWrapper(testDB)
			})

			deleteNetwork := func(network *schema.Resource) {
				Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
					schema, _ := schema.GetManager().Schema("network")
					return tx.Delete(ctx, schema, network.ID())
				})).To(Succeed())
			}

			AfterEach(func() {
				deleteNetwork(red)
				deleteNetwork(green)
				deleteNetwork(blue)

				sync.Close()

				cancel()
				done.Wait()
			})

			createNetwork := func(tx transaction.Transaction, label string) (map[string]interface{}, *schema.Resource) {
				label = label + uuid.NewV4().String()

				manager := schema.GetManager()
				network := getNetwork(label, label)
				networkResource, err := manager.LoadResource("network", network)
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(ctx, networkResource)).To(Succeed())

				return network, networkResource
			}

			checkIsSynced := func(rawResource map[string]interface{}, resource *schema.Resource) {
				var writtenConfig *gohan_sync.Node
				Eventually(func() error {
					var err error
					writtenConfig, err = sync.Fetch("/config" + resource.Path())
					return err
				}, 10*time.Second).Should(Succeed())

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

			It("writes all events from a single transaction", func() {

				err := db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
					rawRed, red = createNetwork(tx, "red")
					rawBlue, blue = createNetwork(tx, "blue")
					rawGreen, green = createNetwork(tx, "green")

					return nil
				})
				Expect(err).ToNot(HaveOccurred())

				checkIsSynced(rawRed, red)
				checkIsSynced(rawBlue, blue)
				checkIsSynced(rawGreen, green)
			})

			It("writes all events from many transactions", func() {
				Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
					rawRed, red = createNetwork(tx, "red")
					return nil
				})).To(Succeed())

				Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
					rawBlue, blue = createNetwork(tx, "blue")
					return nil
				})).To(Succeed())

				Expect(db.WithinTx(syncedDb, func(tx transaction.Transaction) error {
					rawGreen, green = createNetwork(tx, "green")
					return nil
				})).To(Succeed())

				checkIsSynced(rawRed, red)
				checkIsSynced(rawBlue, blue)
				checkIsSynced(rawGreen, green)
			})
		})

	})
})
