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

package extension_test

import (
	"os"
	"testing"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/sync/etcdv3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testDB1  db.DB
	testDB2  db.DB
	testSync *etcdv3.Sync
)

func TestExtension(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extension Suite")
}

var _ = Describe("Suite set up and tear down", func() {
	const (
		testDBFile1      = "./extensionTest1.db"
		testDBFile2      = "./extensionTest2.db"
		testSyncEndpoint = "localhost:2379"
	)

	var _ = BeforeSuite(func() {
		var err error
		testDB1, err = db.ConnectDB("sqlite3", testDBFile1, db.DefaultMaxOpenConn, options.Default())
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		testDB2, err = db.ConnectDB("sqlite3", testDBFile2, db.DefaultMaxOpenConn, options.Default())
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		testSync, err = etcdv3.NewSync([]string{testSyncEndpoint}, time.Second)
		Expect(err).ToNot(HaveOccurred(), "Failed to connect to etcd")
	})

	var _ = AfterSuite(func() {
		testSync.Close()
		os.Remove(testDBFile1)
		os.Remove(testDBFile2)
	})
})
