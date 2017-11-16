// Copyright (C) 2016  Juniper Networks, Inc.
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

package gohanscript_test

import (
	"os"
	"testing"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	configDir         = "."
	configFile        = "./server_test_config.yaml"
	adminTokenID      = "admin_token"
	memberTokenID     = "demo_token"
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

func clearTable(tx transaction.Transaction, s *schema.Schema) error {
	if s.IsAbstract() {
		return nil
	}
	for _, schema := range schema.GetManager().Schemas() {
		if schema.ParentSchema == s {
			err := clearTable(tx, schema)
			if err != nil {
				return err
			}
		}
	}
	resources, _, err := tx.List(s, nil, nil, nil)
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

func TestGohanScriptExtension(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gohan script Extension Suite")
}

var _ = Describe("Suite set up and tear down", func() {
	var _ = BeforeSuite(func() {
		Expect(os.Chdir(configDir)).To(Succeed())
		var err error
		testDB, err = db.ConnectLocal()
		Expect(err).To(Succeed())
		manager := schema.GetManager()
		schema.DefaultExtension = "gohanscript"
		config := util.GetConfig()
		Expect(config.ReadConfig(configFile)).To(Succeed())
		schemaFiles := config.GetStringList("schemas", nil)
		Expect(schemaFiles).NotTo(BeNil())
		Expect(manager.LoadSchemasFromFiles(schemaFiles...)).To(Succeed())
		Expect(db.InitSchemaConn(testDB, db.DefaultTestSchemaParams())).To(Succeed())
	})

	var _ = AfterSuite(func() {
		schema.ClearManager()
		testDB.Purge()
	})
})
