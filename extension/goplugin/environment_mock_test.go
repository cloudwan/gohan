// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package goplugin_test

import (
	"os"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
	"github.com/cloudwan/gohan/schema"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mocks", func() {
	var (
		env        *goplugin.MockIEnvironment
		dbFile     string
		testSchema goext.ISchema
		context    goext.Context
	)

	const (
		SchemaPath = "test_data/test_schema.yaml"
	)

	BeforeEach(func() {
		var dbType string
		if os.Getenv("MYSQL_TEST") == "true" {
			dbFile = "root@/gohan_test"
			dbType = "mysql"
		} else {
			dbFile = "test.db"
			dbType = "sqlite3"
		}

		schemaManager := schema.GetManager()
		Expect(schemaManager.LoadSchemaFromFile(SchemaPath)).To(Succeed())
		rawDB, err := dbutil.ConnectDB(dbType, dbFile, db.DefaultMaxOpenConn, options.Default())
		Expect(err).ToNot(HaveOccurred())
		envReal := goplugin.NewEnvironment("test", nil, nil)

		Expect(envReal.Load("test_data/ext_good/ext_good.so")).To(Succeed())
		Expect(envReal.Start()).To(Succeed())
		envReal.SetDatabase(rawDB)
		env = goplugin.NewMockIEnvironment(envReal, GinkgoT())
		env.Reset()
		testSchema = env.Schemas().Find("test")
		Expect(testSchema).To(Not(BeNil()))
		Expect(dbutil.InitDBWithSchemas(dbType, dbFile, db.DefaultTestInitDBParams())).To(Succeed())
		context = goext.MakeContext()
	})

	AfterEach(func() {
		env.Reset()
		os.Remove(dbFile)
		schema.ClearManager()
	})

	Context("Mocking module", func() {
		It("should mock sync module", func() {
			env.SetMockModules(goext.MockModules{Sync: true})
			env.MockSync().EXPECT().Fetch("testKey").Return(&goext.Node{Value: "42"}, nil)

			resp, err := env.Sync().Fetch("testKey")

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Value).To(Equal("42"))
		})

		It("should not affect another module", func() {
			createdResource := test.Test{
				ID:          "some-id",
				Description: "description",
				TestSuiteID: goext.MakeNullString(),
				Name:        goext.MakeNullString(),
			}
			env.Database().Within(context, func(tx goext.ITransaction) error {
				Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
				return nil
			})

			env.SetMockModules(goext.MockModules{Sync: true})

			env.Database().Within(context, func(tx goext.ITransaction) error {
				returnedResources, err := testSchema.ListRaw(goext.Filter{}, nil, context)
				Expect(err).ToNot(HaveOccurred())
				Expect(returnedResources).To(HaveLen(1))
				returnedResource := returnedResources[0]
				Expect(&createdResource).To(Equal(returnedResource))
				return nil
			})
		})
	})

	Context("Reset mock env", func() {
		It("should create new controller", func() {
			env.SetMockModules(goext.MockModules{Sync: true})
			env.MockSync().EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			env.Reset()
			env.GetController().Finish()
		})
	})
})
