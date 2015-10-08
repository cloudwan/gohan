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

package db

import (
	"os"

	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func makePluralKey(resource *schema.Resource) string {
	s := resource.Schema()
	key := s.Prefix + "/" + s.Plural
	return key
}

var _ = Describe("Database operation test", func() {
	var conn, dbType string
	if os.Getenv("MYSQL_TEST") == "true" {
		conn = "root@/gohan_test"
		dbType = "mysql"
	} else {
		conn = "./test.db"
		dbType = "sqlite3"
	}

	BeforeEach(func() {
		if os.Getenv("MYSQL_TEST") != "true" {
			os.Remove(conn)
		}
	})
	AfterEach(func() {
		schema.ClearManager()
		if os.Getenv("MYSQL_TEST") != "true" {
			os.Remove(conn)
		}
	})
	Describe("Database operation test", func() {
		It("should be CRUD db operation works", func() {
			manager := schema.GetManager()
			db, err := ConnectDB(dbType, conn)
			Expect(err).ToNot(HaveOccurred())
			Expect(manager.LoadSchemasFromFiles(
				"../etc/schema/gohan.json", "../etc/apps/example.yaml")).To(Succeed())

			InitDBWithSchemas(dbType, conn, true, false)

			networkSchema, ok := manager.Schema("network")
			Expect(ok).To(Equal(true))
			subnetSchema, ok := manager.Schema("network")
			Expect(ok).To(Equal(true))

			network := map[string]interface{}{
				"id":            "networkRed",
				"name":          "NetworkRed",
				"description":   "A crimson network",
				"tenant_id":     "red",
				"shared":        false,
				"route_targets": []string{"1000:10000", "2000:20000"},
				"providor_networks": map[string]interface{}{
					"segmentation_id": 10, "segmentation_type": "vlan"}}

			networkResource, err := manager.LoadResource("network", network)
			Expect(err).ToNot(HaveOccurred())

			tx, err := db.Begin()
			Expect(err).ToNot(HaveOccurred())
			list, _, err := tx.List(networkSchema, nil, nil)
			tx.Commit()
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(BeEmpty())

			tx, err = db.Begin()
			err = tx.Create(networkResource)
			tx.Commit()
			Expect(err).ToNot(HaveOccurred())

			tx, err = db.Begin()
			list, _, err = tx.List(networkSchema, nil, nil)
			tx.Commit()
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))

			tx, err = db.Begin()
			Expect(err).ToNot(HaveOccurred())
			networkResource, err = tx.Fetch(networkSchema, networkResource.ID(), nil)
			Expect(networkResource).ToNot(BeNil())
			tx.Commit()

			tx, err = db.Begin()
			Expect(err).ToNot(HaveOccurred())
			Expect(networkResource.Update(map[string]interface{}{"id": "new_id"})).ToNot(Succeed())

			Expect(networkResource.Update(map[string]interface{}{"name": "new_name"})).To(Succeed())

			tx.Update(networkResource)
			tx.Commit()

			subnet := map[string]interface{}{
				"id":          "subnetRed",
				"name":        "SubnetRed",
				"description": "A crimson subnet",
				"tenant_id":   "red",
				"cidr":        "10.0.0.0/24"}
			subnetResource, err := manager.LoadResource("subnet", subnet)

			Expect(err).ToNot(HaveOccurred())
			subnetResource.SetParentID("networkRed")
			tx, err = db.Begin()
			err = tx.Create(subnetResource)
			tx.Commit()
			Expect(err).ToNot(HaveOccurred())
			Expect(subnetResource.Path()).To(Equal("/v2.0/subnets/subnetRed"))

			Expect(makePluralKey(subnetResource)).To(Equal("/v2.0/subnets"))

			tx, err = db.Begin()
			Expect(err).ToNot(HaveOccurred())
			By(subnetResource.ID())
			tx.Delete(subnetSchema, subnetResource.ID())
			tx.Delete(networkSchema, networkResource.ID())
			tx.Commit()

			list, _, err = tx.List(networkSchema, nil, nil)
			Expect(list).To(BeEmpty())
		})
		It("should be relation works", func() {
			manager := schema.GetManager()
			os.Remove(conn)
			db, err := ConnectDB(dbType, conn)
			manager.LoadSchemasFromFiles(
				"../etc/schema/gohan.json", "../etc/apps/example.yaml")

			InitDBWithSchemas(dbType, conn, true, false)
			networkSchema, _ := manager.Schema("network")
			serverSchema, _ := manager.Schema("server")

			network := map[string]interface{}{
				"id":            "networkRed",
				"name":          "NetworkRed",
				"description":   "A crimson network",
				"tenant_id":     "red",
				"shared":        false,
				"route_targets": []string{"1000:10000", "2000:20000"},
				"providor_networks": map[string]interface{}{
					"segmentation_id": 10, "segmentation_type": "vlan"}}

			networkResource, _ := manager.LoadResource("network", network)

			tx, _ := db.Begin()
			Expect(tx.Create(networkResource)).To(Succeed())
			tx.Commit()

			server := map[string]interface{}{
				"id":         "serverRed",
				"name":       "serverRed",
				"tenant_id":  "red",
				"network_id": "networkRed",
				"cidr":       "10.0.0.0/24"}
			serverResource, err := manager.LoadResource("server", server)

			tx, err = db.Begin()
			err = tx.Create(serverResource)
			tx.Commit()
			Expect(err).ToNot(HaveOccurred())

			tx, err = db.Begin()
			Expect(err).ToNot(HaveOccurred())
			list, _, err := tx.List(serverSchema, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Data()["network"].(map[string]interface{})["name"]).To(Equal("NetworkRed"))

			tx, err = db.Begin()
			Expect(err).ToNot(HaveOccurred())
			tx.Delete(serverSchema, serverResource.ID())
			tx.Delete(networkSchema, networkResource.ID())
			tx.Commit()
		})
	})
	It("Should convert yaml to sqlite3", func() {
		manager := schema.GetManager()
		Expect(manager.LoadSchemasFromFiles("../etc/schema/gohan.json", "test_data/conv_in.yaml")).To(Succeed())
		inDB, err := ConnectDB("yaml", "test_data/conv_in.yaml")
		Expect(err).ToNot(HaveOccurred())

		InitDBWithSchemas("sqlite3", "test_data/conv_out.db", false, false)
		outDB, err := ConnectDB("sqlite3", "test_data/conv_out.db")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove("test_data/conv_out.db")

		InitDBWithSchemas("yaml", "test_data/conv_verify.yaml", false, false)
		verifyDB, err := ConnectDB("yaml", "test_data/conv_verify.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove("test_data/conv_verify.yaml")

		Expect(CopyDBResources(inDB, outDB)).To(Succeed())

		Expect(CopyDBResources(outDB, verifyDB)).To(Succeed())

		inTx, err := inDB.Begin()
		Expect(err).ToNot(HaveOccurred())
		defer inTx.Close()

		// SQL returns different types than JSON/YAML Database
		// So we need to move it back again so that DeepEqual would work correctly
		verifyTx, err := verifyDB.Begin()
		Expect(err).ToNot(HaveOccurred())
		defer verifyTx.Close()

		for _, s := range manager.OrderedSchemas() {
			if s.Metadata["type"] == "metaschema" {
				continue
			}
			resources, _, err := inTx.List(s, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			for _, inResource := range resources {
				outResource, err := verifyTx.Fetch(s, inResource.ID(), nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(outResource).To(Equal(inResource))
			}
		}
	})
})
