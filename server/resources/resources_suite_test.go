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

package resources_test

import (
	"os"
	"testing"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	configDir           = ".."
	configFile          = "./server_test_config.yaml"
	dbType              = "sqlite3"
	dbFile              = "./test.db"
	testSyncEndpoint    = "localhost:2379"
	adminTenantID       = "adminTenantID"
	memberTenantID      = "memberTenantID"
	otherDomainTenantID = "otherDomainTenantID"
	domainAID           = "domainAID"
	domainBID           = "domainBID"
)

var (
	testDB   db.DB
	testSync *etcdv3.Sync

	whitelist = map[string]bool{
		"schema":    true,
		"policy":    true,
		"extension": true,
		"namespace": true,
	}
)

func TestResources(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resources Suite")
}

var _ = Describe("Suite set up and tear down", func() {
	var _ = BeforeSuite(func() {
		var err error
		Expect(os.Chdir(configDir)).To(Succeed())
		testDB, err = dbutil.ConnectDB(dbType, dbFile, db.DefaultMaxOpenConn, options.Default())
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		testSync, err = etcdv3.NewSync([]string{testSyncEndpoint}, time.Second)
		Expect(err).NotTo(HaveOccurred(), "Failed to connect to etcd")
		manager := schema.GetManager()
		config := util.GetConfig()
		Expect(config.ReadConfig(configFile)).To(Succeed())
		schemaFiles := config.GetStringList("schemas", nil)
		Expect(schemaFiles).NotTo(BeNil())
		Expect(manager.LoadSchemasFromFiles(schemaFiles...)).To(Succeed())
		Expect(dbutil.InitDBWithSchemas(dbType, dbFile, db.DefaultTestInitDBParams())).To(Succeed())
	})

	var _ = AfterSuite(func() {
		schema.ClearManager()
		testSync.Close()
		os.Remove(dbFile)
	})
})
