package server_test

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/schema"
)

const (
	adminTokenID      = "admin_token"
	memberTokenID     = "member_token"
	powerUserTokenID  = "power_user_token"
	adminTenantID     = "fc394f2ab2df4114bde39905f800dc57"
	memberTenantID    = "fc394f2ab2df4114bde39905f800dc57"
	powerUserTenantID = "acf5662bbff44060b93ac3db3c25a590"
)

var (
	testDB    db.DB
	whitelist = map[string]bool{
		"schema":    true,
		"policy":    true,
		"extension": true,
		"namespace": true,
	}
)

func testJSONEquality(actual, expected interface{}) {
	actualJSON, err := json.Marshal(actual)
	Expect(err).ToNot(HaveOccurred())
	expectedJSON, err := json.Marshal(expected)
	Expect(err).ToNot(HaveOccurred())
	Expect(actualJSON).To(MatchJSON(expectedJSON))
}

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("Suit set up and tear down", func() {
	var conn, dbType string
	if os.Getenv("MYSQL_TEST") == "true" {
		conn = "root@/gohan_test"
		dbType = "mysql"
	} else {
		conn = "./test.db"
		dbType = "sqlite3"
	}

	var _ = BeforeSuite(func() {
		var err error
		testDB, err = db.ConnectDB(dbType, conn)
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		if os.Getenv("MYSQL_TEST") == "true" {
			err = startTestServer("./server_test_mysql_config.yaml")
		} else {
			err = startTestServer("./server_test_config.yaml")
		}
		Expect(err).ToNot(HaveOccurred(), "Failed to start test server.")
	})

	var _ = AfterSuite(func() {
		schema.ClearManager()
		os.Remove(conn)
	})
})
