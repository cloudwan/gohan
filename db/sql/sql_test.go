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

package sql_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwan/gohan/db"
	. "github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sql", func() {

	const testFixtures = "test_fixture.json"

	var (
		conn    string
		tx      transaction.Transaction
		sqlConn *DB
	)

	BeforeEach(func() {
		var dbType string
		if os.Getenv("MYSQL_TEST") == "true" {
			conn = "gohan:gohan@/gohan_test"
			dbType = "mysql"
		} else {
			conn = "./test.db"
			dbType = "sqlite3"
		}

		manager := schema.GetManager()
		dbc, err := db.ConnectDB(dbType, conn, db.DefaultMaxOpenConn)
		sqlConn = dbc.(*DB)
		Expect(err).ToNot(HaveOccurred())
		Expect(manager.LoadSchemasFromFiles(
			"../../etc/schema/gohan.json", "../../tests/test_abstract_schema.yaml", "../../tests/test_schema.yaml")).To(Succeed())
		db.InitDBWithSchemas(dbType, conn, true, false, false)

		// Insert fixture data
		fixtureDB, err := db.ConnectDB("json", testFixtures, db.DefaultMaxOpenConn)
		Expect(err).ToNot(HaveOccurred())
		db.CopyDBResources(fixtureDB, dbc, true)

		tx, err = dbc.Begin()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		tx.Close()
		sqlConn.Close()

		schema.ClearManager()
		if os.Getenv("MYSQL_TEST") != "true" {
			os.Remove(conn)
		}
	})

	Describe("MakeColumns", func() {
		var s *schema.Schema

		BeforeEach(func() {
			manager := schema.GetManager()
			var ok bool
			s, ok = manager.Schema("test")
			Expect(ok).To(BeTrue())
		})

		Context("Without fields", func() {
			It("Returns all colums", func() {
				cols := MakeColumns(s, s.GetDbTableName(), nil, false)
				Expect(cols).To(HaveLen(6))
			})
		})

		Context("With fields", func() {
			It("Returns selected colums", func() {
				cols := MakeColumns(s, s.GetDbTableName(), []string{"test.id", "test.tenant_id"}, false)
				Expect(cols).To(HaveLen(2))
			})
		})
	})

	Describe("Query", func() {
		type testRow struct {
			ID          string  `json:"id"`
			TenantID    string  `json:"tenant_id"`
			TestString  string  `json:"test_string"`
			TestNumber  float64 `json:"test_number"`
			TestInteger int     `json:"test_integer"`
			TestBool    bool    `json:"test_bool"`
		}

		var (
			s        *schema.Schema
			expected []*testRow
		)

		var v map[string][]*testRow
		readFixtures(testFixtures, &v)
		expected = v["tests"]

		BeforeEach(func() {
			manager := schema.GetManager()
			var ok bool
			s, ok = manager.Schema("test")
			Expect(ok).To(BeTrue())
		})

		Context("Without place holders", func() {
			It("Returns resources", func() {
				query := fmt.Sprintf(
					"SELECT %s FROM %s",
					strings.Join(MakeColumns(s, s.GetDbTableName(), nil, false), ", "),
					s.GetDbTableName(),
				)
				results, err := tx.Query(s, query, []interface{}{})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(results)).To(Equal(4))

				for i, r := range results {
					Expect(r.Data()).To(Equal(map[string]interface{}{
						"id":           expected[i].ID,
						"tenant_id":    expected[i].TenantID,
						"test_string":  expected[i].TestString,
						"test_number":  expected[i].TestNumber,
						"test_integer": expected[i].TestInteger,
						"test_bool":    expected[i].TestBool,
					}))
				}
			})
		})

		Context("With a place holder", func() {
			It("Replace the place holder and returns resources", func() {
				query := fmt.Sprintf(
					"SELECT %s FROM %s WHERE tenant_id = ?",
					strings.Join(MakeColumns(s, s.GetDbTableName(), nil, false), ", "),
					s.GetDbTableName(),
				)
				results, err := tx.Query(s, query, []interface{}{"tenant0"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(results)).To(Equal(2))

				for i, r := range results {
					Expect(r.Data()).To(Equal(map[string]interface{}{
						"id":           expected[i].ID,
						"tenant_id":    expected[i].TenantID,
						"test_string":  expected[i].TestString,
						"test_number":  expected[i].TestNumber,
						"test_integer": expected[i].TestInteger,
						"test_bool":    expected[i].TestBool,
					}))
				}
			})
		})

		Context("With place holders", func() {
			It("Replace the place holders and returns resources", func() {
				query := fmt.Sprintf(
					"SELECT %s FROM %s WHERE tenant_id = ? AND test_string = ?",
					strings.Join(MakeColumns(s, s.GetDbTableName(), nil, false), ", "),
					s.GetDbTableName(),
				)
				results, err := tx.Query(s, query, []interface{}{"tenant0", "obj1"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(results)).To(Equal(1))

				Expect(results[0].Data()).To(Equal(map[string]interface{}{
					"id":           expected[1].ID,
					"tenant_id":    expected[1].TenantID,
					"test_string":  expected[1].TestString,
					"test_number":  expected[1].TestNumber,
					"test_integer": expected[1].TestInteger,
					"test_bool":    expected[1].TestBool,
				}),
				)
			})
		})
	})

	Describe("Generate Table", func() {
		var server *schema.Schema
		var subnet *schema.Schema
		var test *schema.Schema

		BeforeEach(func() {
			manager := schema.GetManager()
			var ok bool
			server, ok = manager.Schema("server")
			Expect(ok).To(BeTrue())
			subnet, ok = manager.Schema("subnet")
			Expect(ok).To(BeTrue())
			test, ok = manager.Schema("test")
			Expect(ok).To(BeTrue())
		})

		Context("Index on multiple columns", func() {
			It("Should create unique index on tenant_id and id", func() {
				_, indices := sqlConn.GenTableDef(test, false)
				Expect(indices).To(HaveLen(2))
				Expect(indices[1]).To(ContainSubstring("CREATE UNIQUE INDEX unique_id_and_tenant_id ON `tests`(`id`,`tenant_id`);"))
			})
		})

		Context("Index in schema", func() {
			It("Should create index, if schema property should be indexed", func() {
				_, indices := sqlConn.GenTableDef(test, false)
				Expect(indices).To(HaveLen(2))
				Expect(indices[0]).To(ContainSubstring("CREATE INDEX tests_tenant_id_idx ON `tests`(`tenant_id`(255));"))
			})
		})

		Context("Relation column name", func() {
			It("Generate foreign key with default column name when relationColumn not available", func() {
				table, _ := sqlConn.GenTableDef(server, false)
				Expect(table).To(ContainSubstring("REFERENCES `networks`(id)"))
			})

			It("Generate foreign key with given column same as relationColumn from property", func() {
				server.Properties = append(server.Properties, schema.NewProperty(
					"test",
					"test",
					"",
					"test",
					"string",
					"subnet",
					"cidr",
					"",
					"varchar(255)",
					false,
					false,
					false,
					nil,
					nil,
					false,
				))
				table, _, err := sqlConn.AlterTableDef(server, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).To(ContainSubstring("REFERENCES `subnets`(cidr)"))
			})
		})

		Context("With default cascade option", func() {
			It("Generate proper table with cascade delete", func() {
				table, _ := sqlConn.GenTableDef(server, true)
				Expect(table).To(ContainSubstring("REFERENCES `networks`(id) on delete cascade);"))
				table, _ = sqlConn.GenTableDef(subnet, true)
				Expect(table).To(ContainSubstring("REFERENCES `networks`(id) on delete cascade);"))
			})
		})

		Context("Without default cascade option", func() {
			It("Generate proper table with cascade delete", func() {
				table, _ := sqlConn.GenTableDef(server, false)
				Expect(table).To(ContainSubstring("REFERENCES `networks`(id) on delete cascade);"))
				table, _ = sqlConn.GenTableDef(subnet, false)
				Expect(table).ToNot(ContainSubstring("REFERENCES `networks`(id) on delete cascade);"))
			})
		})

		Context("Properties modifed", func() {
			It("Generate proper alter table statements", func() {
				server.Properties = append(server.Properties, schema.NewProperty(
					"test",
					"test",
					"",
					"test",
					"string",
					"",
					"",
					"",
					"varchar(255)",
					false,
					false,
					false,
					nil,
					nil,
					false,
				))
				table, _, err := sqlConn.AlterTableDef(server, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).To(ContainSubstring("alter table`servers` add (`test` varchar(255) not null);"))
			})

			It("Create index if property should be indexed", func() {
				server.Properties = append(server.Properties, schema.NewProperty(
					"test",
					"test",
					"",
					"test",
					"string",
					"",
					"",
					"",
					"varchar(255)",
					false,
					false,
					false,
					nil,
					nil,
					true,
				))
				_, indices, err := sqlConn.AlterTableDef(server, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(indices).To(HaveLen(1))
				Expect(indices[0]).To(ContainSubstring("CREATE INDEX servers_test_idx ON `servers`(`test`);"))
			})
		})
	})
})

func readFixtures(path string, v interface{}) {
	f, err := os.Open(path)
	if err != nil {
		panic("failed to open test fixtures")
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&v)
	if err != nil {
		panic("failed parse test fixtures")
	}
}
