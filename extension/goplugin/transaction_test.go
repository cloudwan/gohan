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
	"context"
	"os"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
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
		db, err := dbutil.ConnectDB(DbType, DbFile, db.DefaultMaxOpenConn, options.Default())
		Expect(err).To(BeNil())
		env = goplugin.NewEnvironment("test", nil, nil)
		env.SetDatabase(db)
	})

	AfterEach(func() {
		os.Remove(DbFile)
		schema.ClearManager()
	})

	Describe("CRUD", func() {
		var (
			testSchema      goext.ISchema
			tx              goext.ITransaction
			createdResource map[string]interface{}
		)

		BeforeEach(func() {
			err := env.Load("test_data/ext_good/ext_good.so")
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))
			Expect(dbutil.InitDBWithSchemas(DbType, DbFile, db.DefaultTestInitDBParams())).To(Succeed())

			tx, err = env.Database().Begin()
			Expect(err).To(BeNil())

			createdResource = map[string]interface{}{
				"id":            "some-id",
				"description":   "description",
				"name":          nil,
				"subobject":     map[string]interface{}{},
				"test_suite_id": nil,
				"enumerations":  nil,
			}
		})

		AfterEach(func() {
			env.Stop()
		})

		It("Should create resource", func() {
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
		})

		It("Lists previously created resource", func() {
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
			res, total, err := tx.List(context.Background(), testSchema, goext.Filter{}, nil, nil)
			Expect(err).To(BeNil())
			Expect(total).To(Equal(uint64(1)))
			Expect(res).To(HaveLen(1))
			returnedResource := res[0]
			Expect(createdResource).To(Equal(returnedResource))
		})

		It("Fetch previously created resource", func() {
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
			returnedResource, err := tx.Fetch(context.Background(), testSchema, goext.Filter{"id": createdResource["id"]})
			Expect(err).To(BeNil())
			Expect(createdResource).To(Equal(returnedResource))
		})

		It("Fetch previously created resource with subobject", func() {
			createdResource["subobject"] = map[string]interface{}{"subproperty": "subproperty"}
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
			returnedResource, err := tx.Fetch(context.Background(), testSchema, goext.Filter{"id": createdResource["id"]})
			Expect(err).To(BeNil())
			Expect(createdResource).To(Equal(returnedResource))
		})

		It("Delete previously created resource", func() {
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
			Expect(tx.Delete(context.Background(), testSchema, createdResource["id"])).To(Succeed())
			returnedResource, err := tx.Fetch(context.Background(), testSchema, goext.Filter{"id": createdResource["id"]})
			Expect(err).ToNot(BeNil())
			Expect(returnedResource).To(BeNil())
		})

		It("Update previously created resource", func() {
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
			createdResource["description"] = "other-description"
			Expect(tx.Update(context.Background(), testSchema, createdResource)).To(Succeed())
			returnedResource, err := tx.Fetch(context.Background(), testSchema, goext.Filter{"id": createdResource["id"]})
			Expect(err).To(BeNil())
			Expect(returnedResource["description"]).To(Equal("other-description"))
		})

		It("Exec update is effective", func() {
			Expect(tx.Create(context.Background(), testSchema, createdResource)).To(Succeed())
			Expect(tx.Exec(context.Background(), "UPDATE `tests` SET `description`=? WHERE `id`=?", "updated description", createdResource["id"].(string)))
			returnedResource, err := tx.Fetch(context.Background(), testSchema, goext.Filter{"id": createdResource["id"]})
			Expect(err).To(BeNil())
			Expect(returnedResource["description"]).To(Equal("updated description"))
		})
	})
})
