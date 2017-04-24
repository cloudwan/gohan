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
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transaction", func() {
	var (
		env *goplugin.Environment
	)

	const (
		DbFile     = "test.db"
		DbType     = "sqlite3"
		SchemaPath = "test_data/test_schema.yaml"
	)

	BeforeEach(func() {
		manager := schema.GetManager()
		Expect(manager.LoadSchemaFromFile(SchemaPath)).To(Succeed())
		db, err := db.ConnectDB(DbType, DbFile, db.DefaultMaxOpenConn, options.Default())
		Expect(err).To(BeNil())
		env = goplugin.NewEnvironment("test", db, nil, nil)
	})

	AfterEach(func() {
		os.Remove(DbFile)

	})

	Describe("CRUD", func() {
		var (
			testSchema      goext.ISchema
			createdResource test.Test
			context         goext.Context
		)

		BeforeEach(func() {
			loaded, err := env.Load("test_data/ext_good/ext_good.so", nil)
			Expect(loaded).To(BeTrue())
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))

			err = db.InitDBWithSchemas(DbType, DbFile, true, true, false)
			Expect(err).To(BeNil())

			tx, err := env.Database().Begin()
			Expect(err).To(BeNil())

			context = goext.MakeContext().WithTransaction(tx)

			createdResource = test.Test{
				ID:          "some-id",
				Description: "description",
			}
		})

		AfterEach(func() {
			env.Stop()
		})

		It("Should create resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
		})

		It("Lists previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			returnedResources, err := testSchema.ListRaw(goext.Filter{}, nil, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedResources).To(HaveLen(1))
			returnedResource := returnedResources[0]
			Expect(&createdResource).To(Equal(returnedResource))
		})

		It("Fetch previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			returnedResource, err := testSchema.FetchRaw(createdResource.ID, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(&createdResource).To(Equal(returnedResource))
		})

		It("DeleteRaw previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			Expect(testSchema.DeleteRaw(goext.Filter{"id": createdResource.ID}, context)).To(Succeed())
			_, err := testSchema.FetchRaw(createdResource.ID, context)
			Expect(err).To(HaveOccurred())
		})

		It("UpdateRaw previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			createdResource.Description = "other-description"
			Expect(testSchema.UpdateRaw(&createdResource, context)).To(Succeed())
			returnedResource, err := testSchema.FetchRaw(createdResource.ID, context)
			Expect(err).ToNot(HaveOccurred())
			returnedTest := returnedResource.(*test.Test)
			Expect(returnedTest.Description).To(Equal("other-description"))
		})
	})
})
