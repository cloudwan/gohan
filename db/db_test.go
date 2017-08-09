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

package db_test

import (
	"os"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database operation test", func() {
	var (
		err error
		ok  bool

		conn   string
		dbType string

		manager          *schema.Manager
		networkSchema    *schema.Schema
		serverSchema     *schema.Schema
		networkResource1 *schema.Resource
		networkResource2 *schema.Resource
		subnetResource   *schema.Resource
		serverResource   *schema.Resource

		dataStore db.DB
	)

	BeforeEach(func() {
		manager = schema.GetManager()
		Expect(manager.LoadSchemaFromFile("../etc/schema/gohan.json")).To(Succeed())
	})

	AfterEach(func() {
		schema.ClearManager()
		if os.Getenv("MYSQL_TEST") != "true" {
			os.Remove(conn)
		}
	})

	Describe("Base operations", func() {
		var (
			tx transaction.Transaction
		)

		BeforeEach(func() {
			Expect(manager.LoadSchemaFromFile("../tests/test_abstract_schema.yaml")).To(Succeed())
			Expect(manager.LoadSchemaFromFile("../tests/test_schema.yaml")).To(Succeed())
			networkSchema, ok = manager.Schema("network")
			Expect(ok).To(BeTrue())
			_, ok = manager.Schema("subnet")
			Expect(ok).To(BeTrue())
			serverSchema, ok = manager.Schema("server")
			Expect(ok).To(BeTrue())

			network1 := map[string]interface{}{
				"id":                "networkRed",
				"name":              "NetworkRed",
				"description":       "A crimson network",
				"tenant_id":         "red",
				"shared":            false,
				"route_targets":     []string{"1000:10000", "2000:20000"},
				"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"}}
			networkResource1, err = manager.LoadResource("network", network1)
			Expect(err).ToNot(HaveOccurred())

			network2 := map[string]interface{}{
				"id":                "networkBlue",
				"name":              "NetworkBlue",
				"description":       "A crimson network",
				"tenant_id":         "blue",
				"shared":            false,
				"route_targets":     []string{"1000:10000", "2000:20000"},
				"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"}}
			networkResource2, err = manager.LoadResource("network", network2)
			Expect(err).ToNot(HaveOccurred())

			subnet := map[string]interface{}{
				"id":          "subnetRed",
				"name":        "SubnetRed",
				"description": "A crimson subnet",
				"tenant_id":   "red",
				"cidr":        "10.0.0.0/24"}
			subnetResource, err = manager.LoadResource("subnet", subnet)
			Expect(err).ToNot(HaveOccurred())
			subnetResource.SetParentID("networkRed")
			Expect(subnetResource.Path()).To(Equal("/v2.0/subnets/subnetRed"))

			server := map[string]interface{}{
				"id":          "serverRed",
				"name":        "serverRed",
				"tenant_id":   "red",
				"network_id":  "networkRed",
				"description": "red server",
				"cidr":        "10.0.0.0/24"}
			serverResource, err = manager.LoadResource("server", server)
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			os.Remove(conn)
			dataStore, err = db.ConnectDB(dbType, conn, db.DefaultMaxOpenConn, options.Default())
			Expect(err).ToNot(HaveOccurred())

			for _, s := range manager.Schemas() {
				Expect(dataStore.RegisterTable(s, false, true)).To(Succeed())
			}

			tx, err = dataStore.Begin()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			tx.Close()
		})

		Describe("Using sql", func() {
			BeforeEach(func() {
				if os.Getenv("MYSQL_TEST") == "true" {
					conn = "root@/gohan_test"
					dbType = "mysql"
				} else {
					conn = "./test.db"
					dbType = "sqlite3"
				}
			})

			Context("When the database is empty", func() {
				It("Returns an empty list", func() {
					list, num, err := tx.List(networkSchema, nil, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(0)))
					Expect(list).To(BeEmpty())
					Expect(tx.Commit()).To(Succeed())
				})

				It("Creates a resource", func() {
					Expect(tx.Create(networkResource1)).To(Succeed())

					Expect(tx.Commit()).To(Succeed())
				})
			})

			Describe("When the database is not empty", func() {
				JustBeforeEach(func() {
					Expect(tx.Create(networkResource1)).To(Succeed())
					Expect(tx.Create(networkResource2)).To(Succeed())
					Expect(tx.Create(serverResource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
					tx.Close()
					tx, err = dataStore.Begin()
					Expect(err).ToNot(HaveOccurred())
				})

				It("Returns the expected list", func() {
					list, num, err := tx.List(networkSchema, nil, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(2)))
					Expect(list).To(HaveLen(2))
					Expect(list[0]).To(util.MatchAsJSON(networkResource1))
					Expect(list[1]).To(util.MatchAsJSON(networkResource2))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Locks the expected list", func() {
					list, num, err := tx.LockList(networkSchema, nil, nil, nil, schema.LockRelatedResources)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(2)))
					Expect(list).To(HaveLen(2))
					Expect(list[0]).To(util.MatchAsJSON(networkResource1))
					Expect(list[1]).To(util.MatchAsJSON(networkResource2))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Returns the expected list with filter", func() {
					filter := map[string]interface{}{
						"tenant_id": []string{"red"},
					}
					list, num, err := tx.List(networkSchema, filter, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0]).To(util.MatchAsJSON(networkResource1))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Locks the expected list with filter", func() {
					filter := map[string]interface{}{
						"tenant_id": []string{"red"},
					}
					list, num, err := tx.LockList(networkSchema, filter, nil, nil, schema.LockRelatedResources)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0]).To(util.MatchAsJSON(networkResource1))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Returns the error with invalid filter in List", func() {
					filter := map[string]interface{}{
						"bad_filter": []string{"red"},
					}
					_, _, err := tx.List(networkSchema, filter, nil, nil)
					Expect(err).To(HaveOccurred())
				})

				It("Returns the error with invalid filter in LockList", func() {
					filter := map[string]interface{}{
						"bad_filter": []string{"red"},
					}
					_, _, err := tx.LockList(networkSchema, filter, nil, nil, schema.LockRelatedResources)
					Expect(err).To(HaveOccurred())
				})

				It("Shows related resources", func() {
					list, num, err := tx.List(serverSchema, nil, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", networkResource1.Data()["name"])))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Locks related resources when requested", func() {
					list, num, err := tx.LockList(serverSchema, nil, nil, nil, schema.LockRelatedResources)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", networkResource1.Data()["name"])))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Doesn't lock related resources when requested", func() {
					list, num, err := tx.LockList(serverSchema, nil, nil, nil, schema.SkipRelatedResources)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", BeNil())))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Doesn't show related resources when details is false", func() {
					list, num, err := tx.List(serverSchema, nil, &transaction.ViewOptions{Details: false}, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).NotTo(HaveKey("network"))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Doesn't show related resources when fields is set and nothing is selected", func() {
					list, num, err := tx.List(serverSchema, nil, &transaction.ViewOptions{
						Details: true,
						Fields:  []string{"id"},
					}, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).To(HaveKey("id"))
					Expect(list[0].Data()).NotTo(HaveKey("network"))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Show related resources when fields is set and something is selected", func() {
					list, num, err := tx.List(serverSchema, nil, &transaction.ViewOptions{
						Details: true,
						Fields:  []string{"id", "network.name"},
					}, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).To(HaveKey("id"))
					Expect(list[0].Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", "NetworkRed")))
					Expect(list[0].Data()).To(HaveKeyWithValue("network", Not(HaveKey("id"))))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Fetches an existing resource", func() {
					networkResourceFetched, err := tx.Fetch(networkSchema, transaction.IDFilter(networkResource1.ID()), nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(networkResourceFetched).To(util.MatchAsJSON(networkResource1))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Fetches and locks an existing resource", func() {
					networkResourceFetched, err := tx.LockFetch(networkSchema, transaction.IDFilter(networkResource1.ID()), schema.LockRelatedResources, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(networkResourceFetched).To(util.MatchAsJSON(networkResource1))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Fetches and locks related resources when requested", func() {
					networkResourceFetched, err := tx.LockFetch(serverSchema, nil, schema.LockRelatedResources, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(networkResourceFetched.Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", networkResource1.Data()["name"])))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Fetches and doesn't lock related resources when requested", func() {
					networkResourceFetched, err := tx.LockFetch(serverSchema, nil, schema.SkipRelatedResources, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(networkResourceFetched.Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", BeNil())))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Updates the resource properly", func() {
					By("Not allowing to update some fields")
					Expect(networkResource1.Update(map[string]interface{}{"id": "new_id"})).ToNot(Succeed())

					By("Updating other fields")
					Expect(networkResource1.Update(map[string]interface{}{"name": "new_name"})).To(Succeed())
					Expect(tx.Update(networkResource1)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
				})

				It("Creates a dependent resource", func() {
					Expect(tx.Create(subnetResource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
				})

				It("Deletes the resource", func() {
					Expect(tx.Delete(serverSchema, serverResource.ID())).To(Succeed())
					Expect(tx.Delete(networkSchema, networkResource1.ID())).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
				})

				Context("Using StateFetch", func() {
					It("Returns the defaults", func() {
						beforeState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource1.ID()))
						Expect(err).ToNot(HaveOccurred())
						Expect(tx.Commit()).To(Succeed())
						Expect(beforeState.ConfigVersion).To(Equal(int64(1)))
						Expect(beforeState.StateVersion).To(Equal(int64(0)))
						Expect(beforeState.State).To(Equal(""))
						Expect(beforeState.Error).To(Equal(""))
						Expect(beforeState.Monitoring).To(Equal(""))
					})
				})
			})
		})
	})

	Context("Initialization", func() {
		BeforeEach(func() {
			conn = "test.db"
			dbType = "sqlite3"
			Expect(manager.LoadSchemaFromFile("../tests/test_abstract_schema.yaml")).To(Succeed())
			Expect(manager.LoadSchemaFromFile("../tests/test_schema.yaml")).To(Succeed())
		})

		It("Should initialize the database without error", func() {
			Expect(db.InitDBWithSchemas(dbType, conn, false, false, true)).To(Succeed())
		})
	})

	Context("Converting", func() {
		BeforeEach(func() {
			Expect(manager.LoadSchemaFromFile("test_data/conv_in.yaml")).To(Succeed())
		})

		It("Should do it properly", func() {
			inDB, err := db.ConnectDB("yaml", "test_data/conv_in.yaml", db.DefaultMaxOpenConn, options.Default())
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_in.db")

			db.InitDBWithSchemas("sqlite3", "test_data/conv_out.db", false, false, true)
			outDB, err := db.ConnectDB("sqlite3", "test_data/conv_out.db", db.DefaultMaxOpenConn, options.Default())
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_out.db")

			db.InitDBWithSchemas("yaml", "test_data/conv_verify.yaml", false, false, true)
			verifyDB, err := db.ConnectDB("yaml", "test_data/conv_verify.yaml", db.DefaultMaxOpenConn, options.Default())
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_verify.yaml")

			Expect(db.CopyDBResources(inDB, outDB, true)).To(Succeed())

			Expect(db.CopyDBResources(outDB, verifyDB, true)).To(Succeed())

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
				resources, _, err := inTx.List(s, nil, nil, nil)
				Expect(err).ToNot(HaveOccurred())
				for _, inResource := range resources {
					outResource, err := verifyTx.Fetch(s, transaction.Filter{"id": inResource.ID()}, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(outResource).To(Equal(inResource))
				}
			}
		})

		It("Should not override existing rows", func() {
			inDB, err := db.ConnectDB("yaml", "test_data/conv_in.yaml", db.DefaultMaxOpenConn, options.Default())
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_in.db")

			db.InitDBWithSchemas("sqlite3", "test_data/conv_out.db", false, false, true)
			outDB, err := db.ConnectDB("sqlite3", "test_data/conv_out.db", db.DefaultMaxOpenConn, options.Default())
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_out.db")

			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_verify.yaml")

			Expect(db.CopyDBResources(inDB, outDB, false)).To(Succeed())
			subnetSchema, _ := manager.Schema("subnet")

			// Update some data
			tx, err := outDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			list, _, err := tx.List(subnetSchema, map[string]interface{}{
				"name": "subnetRedA",
			}, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			subnet := list[0]
			subnet.Data()["description"] = "Updated description"
			err = tx.Update(subnet)
			Expect(err).ToNot(HaveOccurred())
			tx.Commit()
			tx.Close()

			Expect(db.CopyDBResources(inDB, outDB, false)).To(Succeed())
			// check description of subnetRedA
			tx, err = outDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			list, _, err = tx.List(subnetSchema, map[string]interface{}{
				"name": "subnetRedA",
			}, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			subnet = list[0]
			Expect(subnet.Data()["description"]).To(Equal("Updated description"))
			tx.Close()
		})
	})

	Context("Transaction retry after a deadlock", func() {
		var (
			firstConn  db.DB
			secondConn db.DB

			connOpts = options.Options{
				RetryTxCount:    5,
				RetryTxInterval: 100,
			}
		)

		const (
			deadlockDbSchema  = "test_data/test_schema.yaml"
			deadlockDbInitial = "test_data/test_initial.yaml"
			deadlockDbName    = "test_data/deadlock_db.db"
			deadlockDbType    = "sqlite3"
		)

		BeforeEach(func() {
			os.Remove(deadlockDbName)
			Expect(manager.LoadSchemaFromFile(deadlockDbSchema)).To(Succeed())
			Expect(db.InitDBWithSchemas(deadlockDbType, deadlockDbName, false, false, true)).To(Succeed())
			firstConn, err = db.ConnectDB(deadlockDbType, deadlockDbName, db.DefaultMaxOpenConn, connOpts)
			Expect(err).ToNot(HaveOccurred())
			secondConn, err = db.ConnectDB(deadlockDbType, deadlockDbName, db.DefaultMaxOpenConn, connOpts)
			Expect(err).ToNot(HaveOccurred())
			initDB, err := db.ConnectDB("yaml", deadlockDbInitial, db.DefaultMaxOpenConn, options.Default())
			defer initDB.Close()
			Expect(err).ToNot(HaveOccurred())
			Expect(db.CopyDBResources(initDB, firstConn, false)).To(Succeed())
		})

		AfterEach(func() {
			firstConn.Close()
			secondConn.Close()
		})

		It("Within() should retry a few times after a deadlock", func() {
			deadlockCount := 0
			Expect(db.Within(firstConn, func(firstTx transaction.Transaction) error {
				err := db.Within(secondConn, func(secondTx transaction.Transaction) error {
					Expect(firstTx.Exec("update todos set name = 'other_name' where id = 'first'")).To(Succeed())
					deadlockCount++
					if deadlockCount == 4 {
						return nil
					}
					err := secondTx.Exec("update todos set description = 'other_description' where id = 'second'")
					Expect(db.IsDeadlock(err)).To(BeTrue())
					return err
				})
				Expect(err).To(Succeed())
				return nil
			})).To(Succeed())
		})
	})
})
