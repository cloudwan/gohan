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

	"sync"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Schemas", func() {
	var (
		env                    *goplugin.Environment
		dbFile                 string
		dbType                 string
		testSchema             goext.ISchema
		testSuiteSchema        goext.ISchema
		testSchemaNoExtensions goext.ISchema
		rawDB                  db.DB
		schemaManager          *schema.Manager
	)

	const (
		SchemaPath = "test_data/test_schema.yaml"
	)

	BeforeEach(func() {
		if os.Getenv("MYSQL_TEST") == "true" {
			dbFile = "root@/gohan_test"
			dbType = "mysql"
		} else {
			dbFile = "test.db"
			dbType = "sqlite3"
		}

		schemaManager = schema.GetManager()
		Expect(schemaManager.LoadSchemaFromFile(SchemaPath)).To(Succeed())
		var err error
		rawDB, err = db.ConnectDB(dbType, dbFile, db.DefaultMaxOpenConn, options.Default())
		Expect(err).To(BeNil())
		env = goplugin.NewEnvironment("test", nil, nil)
		env.SetDatabase(rawDB)

		Expect(env.Load("test_data/ext_good/ext_good.so")).To(Succeed())
		Expect(env.Start()).To(Succeed())
		testSchema = env.Schemas().Find("test")
		Expect(testSchema).To(Not(BeNil()))
		testSuiteSchema = env.Schemas().Find("test_suite")
		Expect(testSuiteSchema).To(Not(BeNil()))
		testSchemaNoExtensions = env.Schemas().Find("test_schema_no_ext")
		Expect(testSchemaNoExtensions).To(Not(BeNil()))
		Expect(db.InitDBWithSchemas(dbType, dbFile, db.DefaultTestInitDBParams())).To(Succeed())
	})

	AfterEach(func() {
		os.Remove(dbFile)

	})

	Context("DerivedSchemas", func() {
		It("Should get all derived schemas", func() {
			base := env.Schemas().Find("base")
			Expect(base).ToNot(BeNil())

			derived := base.DerivedSchemas()

			Expect(derived).To(Equal([]goext.ISchema{env.Schemas().Find("derived")}))
		})
	})

	Context("Make columns", func() {
		It("Should get correct column names", func() {
			Expect(testSuiteSchema.ColumnNames()).To(Equal([]string{"test_suites.`id` as `test_suites__id`"}))
		})
	})

	Context("Properties", func() {
		It("Should get correct properties", func() {
			Expect(testSchema.Properties()).To(Equal(
				[]goext.Property{
					{
						ID:    "id",
						Title: "ID",
					},
					{
						ID:    "description",
						Title: "Description",
					},
					{
						ID:    "name",
						Title: "Name",
					},
					{
						ID:       "test_suite_id",
						Title:    "Test Suite ID",
						Relation: "test_suite",
					},
					{
						ID:    "subobject",
						Title: "Subobject",
					},
				},
			))
		})
	})

	Context("Relations", func() {
		It("Returns relation info if schema has relations", func() {
			relations := env.Schemas().Relations(testSuiteSchema.ID())
			Expect(relations).To(HaveLen(1))
			relation := relations[0]
			Expect(relation.OnDeleteCascade).To(BeTrue())
			Expect(relation.PropertyID).To(Equal("test_suite_id"))
			Expect(relation.SchemaID).To(Equal(testSchema.ID()))
		})

		It("Returns empty relation info if schema has not any relations", func() {
			relations := env.Schemas().Relations(testSchema.ID())
			Expect(relations).To(BeEmpty())
		})
	})

	Context("CRUD", func() {
		var (
			createdResource test.Test
			context         goext.Context
			tx              goext.ITransaction
		)

		BeforeEach(func() {
			var err error
			tx, err = env.Database().Begin()
			Expect(err).To(BeNil())

			context = goext.MakeContext().WithTransaction(tx)

			createdResource = test.Test{
				ID:          "some-id",
				Description: "description",
				TestSuiteID: goext.MakeNullString(),
				Name:        goext.MakeNullString(),
			}
		})

		AfterEach(func() {
			tx.Close()
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

		It("should fetch resource state", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())

			state, err := testSchema.StateFetchRaw(createdResource.ID, context)

			Expect(err).ToNot(HaveOccurred())
			expected := goext.ResourceState{Error: "", Monitoring: "", State: "", StateVersion: 0, ConfigVersion: 1}
			Expect(state).To(Equal(expected))
		})

		It("should update resource state", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			state := goext.ResourceState{Error: "1", Monitoring: "2", State: "3", StateVersion: 4}

			Expect(testSchema.DbStateUpdateRaw(&createdResource, context, &state)).To(Succeed())

			expected := goext.ResourceState{Error: "1", Monitoring: "2", State: "3", StateVersion: 4, ConfigVersion: 1}
			state, err := testSchema.StateFetchRaw(createdResource.ID, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(expected))
		})

		It("should not update resource state config version", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			state := goext.ResourceState{ConfigVersion: 42}

			Expect(testSchema.DbStateUpdateRaw(&createdResource, context, &state)).To(Succeed())

			expected := goext.ResourceState{ConfigVersion: 1}
			state, err := testSchema.StateFetchRaw(createdResource.ID, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(expected))
		})

		Context("Resource type not registered", func() {
			resourceID := "testId"
			filter := map[string]interface{}{
				"id": resourceID,
			}

			BeforeEach(func() {
				tx, err := rawDB.Begin()
				Expect(err).ShouldNot(HaveOccurred())
				resource, err := schemaManager.LoadResource("test_schema_no_ext", map[string]interface{}{
					"id":   resourceID,
					"name": "testName",
				})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(tx.Create(resource)).To(Succeed())
				Expect(tx.Commit()).To(Succeed())
			})

			It("should fail gracefully in ListRaw", func() {
				_, err := testSchemaNoExtensions.ListRaw(filter, nil, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))

			})

			It("should fail gracefully in List", func() {
				_, err := testSchemaNoExtensions.List(filter, nil, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in LockListRaw", func() {
				_, err := testSchemaNoExtensions.LockListRaw(filter, nil, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in LockList", func() {
				_, err := testSchemaNoExtensions.LockList(filter, nil, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in Fetch", func() {
				_, err := testSchemaNoExtensions.Fetch(resourceID, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))

			})

			It("should fail gracefully in FetchRaw", func() {
				_, err := testSchemaNoExtensions.FetchRaw(resourceID, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in LockFetch", func() {
				_, err := testSchemaNoExtensions.LockFetch(resourceID, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in LockFetchRaw", func() {
				_, err := testSchemaNoExtensions.LockFetchRaw(resourceID, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})
		})

		Context("Resource not found", func() {
			const unknownID = "unknown-id"

			It("Should return error in Fetch", func() {
				_, err := testSchema.Fetch(unknownID, context)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in FetchRaw", func() {
				_, err := testSchema.FetchRaw(unknownID, context)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in LockFetch", func() {
				_, err := testSchema.LockFetch(unknownID, context, goext.SkipRelatedResources)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in LockFetchRaw", func() {
				_, err := testSchema.LockFetchRaw(unknownID, context, goext.SkipRelatedResources)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should not return error in List", func() {
				_, err := testSchema.List(goext.Filter{"id": unknownID}, nil, context)
				Expect(err).To(Succeed())
			})

			It("Should not return error in ListRaw", func() {
				_, err := testSchema.ListRaw(goext.Filter{"id": unknownID}, nil, context)
				Expect(err).To(Succeed())
			})

			It("Should not return error in LockList", func() {
				_, err := testSchema.LockList(goext.Filter{"id": unknownID}, nil, context, goext.SkipRelatedResources)
				Expect(err).To(Succeed())
			})

			It("Should not return error in LockListRaw", func() {
				_, err := testSchema.LockListRaw(goext.Filter{"id": unknownID}, nil, context, goext.SkipRelatedResources)
				Expect(err).To(Succeed())
			})

			It("Should not return error in DeleteRaw", func() {
				Expect(testSchema.DeleteRaw(goext.Filter{"id": unknownID}, context)).To(Succeed())
			})
		})

		Context("Database functions without transaction", func() {
			const unknownID = "unknown-id"

			It("should panic when creating resource without transaction", func() {
				Expect(func() { testSchema.CreateRaw(&createdResource, goext.MakeContext()) }).To(Panic())
			})

			It("should panic when creating resource with closed transaction", func() {
				Expect(tx.Close()).To(Succeed())
				Expect(func() { testSchema.CreateRaw(&createdResource, context) }).To(Panic())
			})

			It("should panic when updating resource without transaction", func() {
				Expect(func() { testSchema.UpdateRaw(&createdResource, goext.MakeContext()) }).To(Panic())
			})

			It("should panic when updating resource with closed transaction", func() {
				Expect(tx.Close()).To(Succeed())
				Expect(func() { testSchema.UpdateRaw(&createdResource, context) }).To(Panic())
			})

			It("should panic when deleting resource without transaction", func() {
				Expect(func() { testSchema.DeleteRaw(goext.Filter{"id": unknownID}, goext.MakeContext()) }).To(Panic())
			})

			It("should panic when deleting resource with closed transaction", func() {
				Expect(tx.Close()).To(Succeed())
				Expect(func() { testSchema.DeleteRaw(goext.Filter{"id": unknownID}, context) }).To(Panic())
			})

			It("should panic when fetching resource without transaction", func() {
				Expect(func() { testSchema.FetchRaw(unknownID, goext.MakeContext()) }).To(Panic())
			})

			It("should panic when fetching resource with closed transaction", func() {
				Expect(tx.Close()).To(Succeed())
				Expect(func() { testSchema.FetchRaw(unknownID, context) }).To(Panic())
			})

			It("should panic when fetching resources without transaction", func() {
				Expect(func() { testSchema.ListRaw(goext.Filter{"id": createdResource.ID}, nil, goext.MakeContext()) }).To(Panic())
			})

			It("should panic when fetching resources with closed transaction", func() {
				Expect(tx.Close()).To(Succeed())
				Expect(func() { testSchema.ListRaw(goext.Filter{"id": unknownID}, nil, context) }).To(Panic())
			})
		})
	})

	Context("Locks", func() {
		var (
			testSchema      goext.ISchema
			createdResource test.Test
		)

		BeforeEach(func() {
			if os.Getenv("MYSQL_TEST") != "true" {
				Skip("Locks are only valid in MySQL")
			}

			createdResource = test.Test{
				ID:          "some-id",
				Description: "description",
			}
		})

		It("Locks single resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, nil)).To(Succeed())

			firstTx, err := env.Database().Begin()
			Expect(err).To(BeNil())
			defer firstTx.Close()

			context := goext.MakeContext().WithTransaction(firstTx)
			_, err = testSchema.LockFetchRaw(createdResource.ID, context, goext.SkipRelatedResources)
			Expect(err).To(Succeed())

			committed := make(chan bool, 1)

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer GinkgoRecover()

				secondTx, err := env.Database().Begin()
				Expect(err).To(Succeed())
				defer secondTx.Close()

				context := goext.MakeContext().WithTransaction(secondTx)
				wg.Done()
				_, err = testSchema.LockFetchRaw(createdResource.ID, context, goext.SkipRelatedResources)
				Expect(err).To(Succeed())
				Expect(committed).Should(Receive())
			}()

			wg.Wait()
			// we're just before the call to LockListRaw in the child goroutine, let's make the call happen
			time.Sleep(time.Millisecond * 10)
			// once we commit, the lock will be released. mark we're done
			committed <- true
			firstTx.Commit()
			// now the child goroutine should wake up and verify it executed after us
			// this test is not perfect, but this implementation may yield false positives, but not false negatives
		})

		It("Locks many resources", func() {
			Expect(testSchema.CreateRaw(&createdResource, nil)).To(Succeed())

			firstTx, err := env.Database().Begin()
			Expect(err).To(BeNil())
			defer firstTx.Close()

			context := goext.MakeContext().WithTransaction(firstTx)
			_, err = testSchema.LockListRaw(map[string]interface{}{"id": createdResource.ID}, nil, context, goext.SkipRelatedResources)
			Expect(err).To(Succeed())

			committed := make(chan bool, 1)

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer GinkgoRecover()

				secondTx, err := env.Database().Begin()
				Expect(err).To(Succeed())
				defer secondTx.Close()

				context := goext.MakeContext().WithTransaction(secondTx)
				wg.Done()
				_, err = testSchema.LockListRaw(map[string]interface{}{"id": createdResource.ID}, nil, context, goext.SkipRelatedResources)
				Expect(err).To(Succeed())
				Expect(committed).Should(Receive())
			}()

			wg.Wait()
			// we're just before the call to LockListRaw in the child goroutine, let's make the call happen
			time.Sleep(time.Millisecond * 10)
			// once we commit, the lock will be released. mark we're done
			committed <- true
			firstTx.Commit()
			// now the child goroutine should wake up and verify it executed after us
			// this test is not perfect, but this implementation may yield false positives, but not false negatives
		})
	})

	Context("Convert", func() {
		It("should convert context to resource", func() {
			context := map[string]interface{}{
				"id":          "42",
				"description": "test",
				"subobject": map[string]interface{}{
					"subproperty": "testproperty",
				},
			}
			expected := &test.Test{
				ID:          "42",
				Description: "test",
				Subobject: &test.Subobject{
					Subproperty: "testproperty",
				},
			}

			resource, err := testSchema.ResourceFromMap(context)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).To(Equal(expected))
		})
		It("should convert resource to context", func() {
			resource := &test.Test{
				ID:          "42",
				Description: "test",
				Subobject: &test.Subobject{
					Subproperty: "testproperty",
				},
			}

			context := env.Util().ResourceToMap(resource)

			Expect(context).To(HaveKeyWithValue("id", "42"))
			Expect(context).To(HaveKeyWithValue("description", "test"))
			Expect(context).To(HaveKey("subobject"))
			Expect(context["subobject"]).To(HaveKeyWithValue("subproperty", "testproperty"))
		})
		It("should not change object after marshal and unmarshal", func() {
			resource := &test.Test{
				ID:          "42",
				Description: "test",
				Subobject: &test.Subobject{
					Subproperty: "testproperty",
				},
			}

			context := env.Util().ResourceToMap(resource)

			result, err := testSchema.ResourceFromMap(context)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(resource))
		})
		It("should convert context to resource with nil suboject", func() {
			context := map[string]interface{}{
				"id":          "42",
				"description": "test",
				"subobject":   nil,
			}
			expected := &test.Test{
				ID:          "42",
				Description: "test",
				Subobject:   nil,
			}

			resource, err := testSchema.ResourceFromMap(context)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).To(Equal(expected))
		})
		It("should convert resource to context", func() {
			resource := &test.Test{
				ID:          "42",
				Description: "test",
				Subobject:   nil,
			}

			context := env.Util().ResourceToMap(resource)

			Expect(context).To(HaveKeyWithValue("id", "42"))
			Expect(context).To(HaveKeyWithValue("description", "test"))
			Expect(context).To(HaveKey("subobject"))
			Expect(context["subobject"]).To(BeNil())
		})
		It("should convert resource to context with null string defined", func() {
			resource := &test.Test{
				ID:          "42",
				Description: "test",
				Name:        goext.MakeString("testName"),
				Subobject:   nil,
			}

			context := env.Util().ResourceToMap(resource)

			Expect(context).To(HaveKeyWithValue("id", "42"))
			Expect(context).To(HaveKeyWithValue("description", "test"))
			Expect(context).To(HaveKeyWithValue("name", "testName"))
			Expect(context).To(HaveKey("subobject"))
			Expect(context["subobject"]).To(BeNil())
		})
		It("should convert resource to context with null string", func() {
			resource := &test.Test{
				ID:          "42",
				Description: "test",
				Name:        goext.MakeNullString(),
				Subobject:   nil,
			}

			context := env.Util().ResourceToMap(resource)

			Expect(context).To(HaveKeyWithValue("id", "42"))
			Expect(context).To(HaveKeyWithValue("description", "test"))
			Expect(context).To(HaveKey("name"))
			Expect(context["name"]).To(BeNil())
			Expect(context).To(HaveKey("subobject"))
			Expect(context["subobject"]).To(BeNil())
		})
		It("should not convert context to resource with int passed as string", func() {
			context := map[string]interface{}{
				"id":          42,
				"description": "test",
				"subobject":   nil,
			}

			_, err := testSchema.ResourceFromMap(context)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid type of 'id' field (int, expecting string)"))
		})
		It("should convert context to resource with missing fields", func() {
			context := map[string]interface{}{
				"id":        nil,
				"subobject": nil,
			}
			expected := &test.Test{
				ID:          "",
				Description: "",
				Subobject:   nil,
			}

			resource, err := testSchema.ResourceFromMap(context)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).To(Equal(expected))
		})
		It("should convert context to resource with null string", func() {
			context := map[string]interface{}{
				"id":        nil,
				"name":      nil,
				"subobject": nil,
			}
			expected := &test.Test{
				ID:          "",
				Description: "",
				Name:        goext.MakeNullString(),
				Subobject:   nil,
			}

			resource, err := testSchema.ResourceFromMap(context)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).To(Equal(expected))
		})
		It("should convert context to resource with null string", func() {
			context := map[string]interface{}{
				"id":        nil,
				"name":      "testname",
				"subobject": nil,
			}
			expected := &test.Test{
				ID:          "",
				Description: "",
				Name:        goext.MakeString("testname"),
				Subobject:   nil,
			}

			resource, err := testSchema.ResourceFromMap(context)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).To(Equal(expected))
		})
	})

	Context("Context passed to handlers should be copied from original", func() {
		var (
			context      goext.Context
			tx           goext.ITransaction
			testResource *test.Test
		)

		BeforeEach(func() {
			var err error
			tx, err = env.Database().Begin()
			Expect(err).ToNot(HaveOccurred())
			context = goext.MakeContext().WithTransaction(tx)
			context["test"] = 42
			testResource = &test.Test{ID: "13", Name: goext.MakeString("123"), TestSuiteID: goext.MakeNullString()}

			checkContext := func(ctx goext.Context, resource goext.Resource, environment goext.IEnvironment) error {
				Expect(&ctx).ToNot(Equal(&context))
				Expect(ctx).To(HaveKeyWithValue("transaction", tx))
				Expect(ctx).To(HaveKeyWithValue("test", 42))
				Expect(ctx).To(HaveKeyWithValue("schema_id", "test"))
				Expect(ctx).To(HaveKeyWithValue("id", "13"))
				Expect(ctx).To(HaveKeyWithValue("resource", env.Util().ResourceToMap(testResource)))
				Expect(resource).To(Equal(testResource))
				return nil
			}

			testSchema.RegisterEventHandler(goext.PreCreateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterEventHandler(goext.PostCreateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterEventHandler(goext.PreUpdateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterEventHandler(goext.PostUpdateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterEventHandler(goext.PreDeleteTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterEventHandler(goext.PostDeleteTx, checkContext, goext.PriorityDefault)
		})

		AfterEach(func() {
			Expect(tx.Close()).To(Succeed())
		})

		It("should copy context for create", func() {
			Expect(testSchema.CreateRaw(testResource, context)).To(Succeed())
		})

		It("should copy context for update", func() {
			Expect(testSchema.CreateRaw(testResource, context)).To(Succeed())
			Expect(testSchema.UpdateRaw(testResource, context)).To(Succeed())
		})

		It("should copy context for delete", func() {
			Expect(testSchema.CreateRaw(testResource, context)).To(Succeed())
			Expect(testSchema.DeleteRaw(goext.Filter{"id": testResource.ID}, context)).To(Succeed())
		})
	})
})
