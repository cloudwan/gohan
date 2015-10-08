package resources_test

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

const (
	configDir         = ".."
	configFile        = "./server_test_config.yaml"
	dbType            = "sqlite3"
	dbFile            = "./test.db"
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

func clearTable(tx transaction.Transaction, s *schema.Schema) error {
	for _, schema := range schema.GetManager().Schemas() {
		if schema.ParentSchema == s {
			err := clearTable(tx, schema)
			if err != nil {
				return err
			}
		}
	}
	resources, _, err := tx.List(s, nil, nil)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		err = tx.Delete(s, resource.ID())
		if err != nil {
			return err
		}
	}
	return nil
}

func TestResources(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resources Suite")
}

var _ = Describe("Suit set up and tear down", func() {
	var _ = BeforeSuite(func() {
		var err error
		Expect(os.Chdir(configDir)).To(Succeed())
		testDB, err = db.ConnectDB(dbType, dbFile)
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		manager := schema.GetManager()
		config := util.GetConfig()
		Expect(config.ReadConfig(configFile)).To(Succeed())
		schemaFiles := config.GetStringList("schemas", nil)
		Expect(schemaFiles).NotTo(BeNil())
		Expect(manager.LoadSchemasFromFiles(schemaFiles...)).To(Succeed())
		Expect(db.InitDBWithSchemas(dbType, dbFile, false, false)).To(Succeed())
	})

	var _ = AfterSuite(func() {
		schema.ClearManager()
		os.Remove(dbFile)
	})
})
