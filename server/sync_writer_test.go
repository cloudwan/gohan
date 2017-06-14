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
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	gohan_etcd "github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/cloudwan/gohan/util"
)

var _ = Describe("Server package test", func() {
	Describe("Sync", func() {
		It("should work", func() {
			manager := schema.GetManager()
			networkSchema, _ := manager.Schema("network")
			network := getNetwork("Red", "red")
			networkResource, err := manager.LoadResource("network", network)
			Expect(err).ToNot(HaveOccurred())
			testDB1 := &srv.DbSyncWrapper{DB: testDB}
			tx, err := testDB1.Begin()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Create(networkResource)).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			writer := srv.NewSyncWriterFromServer(server)
			Expect(writer.Sync()).To(Succeed())

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

			tx, err = testDB1.Begin()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Delete(networkSchema, networkResource.ID())).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			Expect(writer.Sync()).To(Succeed())

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
				testDB1 := &srv.DbSyncWrapper{DB: testDB}
				tx, err := testDB1.Begin()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Succeed())

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

				tx, err = testDB1.Begin()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Succeed())

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
				testDB1 := &srv.DbSyncWrapper{DB: testDB}
				tx, err := testDB1.Begin()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Succeed())

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch("/config" + resource.Path())
				Expect(err).ToNot(HaveOccurred())

				var configContentsRaw map[string]interface{}
				Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
				Expect(configContentsRaw).To(HaveKeyWithValue("id", "r0"))
				Expect(configContentsRaw).To(HaveKeyWithValue("p0", "property0"))

				tx, err = testDB1.Begin()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Succeed())

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
				testDB1 := &srv.DbSyncWrapper{DB: testDB}
				tx, err := testDB1.Begin()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Create(resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				writer := srv.NewSyncWriterFromServer(server)
				Expect(writer.Sync()).To(Succeed())

				sync, err := gohan_etcd.NewSync([]string{"http://127.0.0.1:2379"}, time.Second)
				Expect(err).ToNot(HaveOccurred())

				writtenConfig, err := sync.Fetch("/config" + resource.Path())
				Expect(err).ToNot(HaveOccurred())
				Expect(writtenConfig.Value).To(BeEquivalentTo("property0"))

				tx, err = testDB1.Begin()
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Delete(schema, resource.ID())).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
				tx.Close()

				Expect(writer.Sync()).To(Succeed())

				_, err = sync.Fetch(resource.Path())
				Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
			})
		})
	})
})
