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

	"github.com/cloudwan/gohan/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testDB1 db.DB
	testDB2 db.DB
)

func TestExtension(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extension Suite")
}

var _ = Describe("Suite set up and tear down", func() {
	const (
		testDBFile1 = "./extensionTest1.db"
		testDBFile2 = "./extensionTest2.db"
	)

	var _ = BeforeSuite(func() {
		var err error
		testDB1, err = db.ConnectDB("sqlite3", testDBFile1)
		Expect(err).NotTo(HaveOccurred())
		testDB2, err = db.ConnectDB("sqlite3", testDBFile2)
		Expect(err).NotTo(HaveOccurred())
	})

	var _ = AfterSuite(func() {
		os.Remove(testDBFile1)
		os.Remove(testDBFile2)
	})
})
