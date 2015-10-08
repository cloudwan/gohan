package sql_test

import (
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

	var conn string
	var tx transaction.Transaction

	BeforeEach(func() {
		var dbType string
		if os.Getenv("MYSQL_TEST") == "true" {
			conn = "root@/gohan_test"
			dbType = "mysql"
		} else {
			conn = "./test.db"
			dbType = "sqlite3"
		}

		manager := schema.GetManager()
		dbc, err := db.ConnectDB(dbType, conn)
		Expect(err).ToNot(HaveOccurred())
		Expect(manager.LoadSchemasFromFiles(
			"../../etc/schema/gohan.json", "../../etc/apps/example.yaml")).To(Succeed())
		db.InitDBWithSchemas(dbType, conn, true, false)

		// Insert fixture data
		fixtureDB, err := db.ConnectDB("json", "test_fixture.json")
		Expect(err).ToNot(HaveOccurred())
		db.CopyDBResources(fixtureDB, dbc)

		tx, err = dbc.Begin()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		schema.ClearManager()
		if os.Getenv("MYSQL_TEST") != "true" {
			os.Remove(conn)
		}
	})

	Describe("Query", func() {
		var s *schema.Schema

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
					strings.Join(MakeColumns(s, false), ", "),
					s.GetDbTableName(),
				)
				results, err := tx.Query(s, query, []interface{}{})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0].Get("tenant_id")).To(Equal("tenant0"))
				Expect(results[0].Get("test_string")).To(Equal("obj0"))
				Expect(results[2].Get("tenant_id")).To(Equal("tenant1"))
				Expect(results[2].Get("test_string")).To(Equal("obj2"))
				Expect(len(results)).To(Equal(4))
			})
		})

		Context("With a place holder", func() {
			It("Replace the place holder and returns resources", func() {
				query := fmt.Sprintf(
					"SELECT %s FROM %s WHERE tenant_id = ?",
					strings.Join(MakeColumns(s, false), ", "),
					s.GetDbTableName(),
				)
				results, err := tx.Query(s, query, []interface{}{"tenant0"})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0].Get("tenant_id")).To(Equal("tenant0"))
				Expect(results[0].Get("test_string")).To(Equal("obj0"))
				Expect(results[1].Get("tenant_id")).To(Equal("tenant0"))
				Expect(results[1].Get("test_string")).To(Equal("obj1"))
				Expect(len(results)).To(Equal(2))

			})
		})

		Context("With place holders", func() {
			It("Replace the place holders and returns resources", func() {
				query := fmt.Sprintf(
					"SELECT %s FROM %s WHERE tenant_id = ? AND test_string = ?",
					strings.Join(MakeColumns(s, false), ", "),
					s.GetDbTableName(),
				)
				results, err := tx.Query(s, query, []interface{}{"tenant0", "obj1"})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0].Get("tenant_id")).To(Equal("tenant0"))
				Expect(results[0].Get("test_string")).To(Equal("obj1"))
				Expect(len(results)).To(Equal(1))
			})
		})
	})
})
