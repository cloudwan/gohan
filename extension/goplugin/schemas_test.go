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
	"github.com/cloudwan/gohan/db/dbimpl"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goext/filter"
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
		rawDB, err = dbimpl.ConnectDB(dbType, dbFile, db.DefaultMaxOpenConn, options.Default())
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
		Expect(dbimpl.InitDBWithSchemas(dbType, dbFile, db.DefaultTestInitDBParams())).To(Succeed())
	})

	AfterEach(func() {
		env.Stop()
		os.Remove(dbFile)
		schema.ClearManager()
	})

	It("Extends returns list of schema ids which schema extends", func() {
		derivedSchema := env.Schemas().Find("derived")
		Expect(derivedSchema).ToNot(BeNil())
		parents := derivedSchema.Extends()
		Expect(parents).To(HaveLen(1))
		Expect(parents[0]).To(Equal(goext.SchemaID("base")))
	})

	Context("DerivedSchemas", func() {
		It("Should get all derived schemas", func() {
			base := env.Schemas().Find(goext.SchemaID("base"))
			Expect(base).ToNot(BeNil())

			derived := env.Schemas().Find(goext.SchemaID("derived"))
			expected := []goext.ISchema{derived}
			actual := base.DerivedSchemas()

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Make columns", func() {
		It("Should get correct column names", func() {
			Expect(testSuiteSchema.ColumnNames()).To(Equal([]string{"test_suites.`id` as `test_suites__id`"}))
		})
	})

	Context("Register event handler", func() {
		var (
			eventHandler = func(goext.Context, goext.Resource, goext.IEnvironment) *goext.Error {
				return nil
			}
			customActionHandler = func(goext.Context, goext.IEnvironment) *goext.Error {
				return nil
			}
		)

		Context("RegisterResourceEventHandler", func() {
			It("Should panic when trying to register custom action as event", func() {
				Expect(func() {
					testSchema.RegisterResourceEventHandler("echo", eventHandler, 0)
				}).To(Panic())
			})

			It("Should not panic when trying to register event as event", func() {
				Expect(func() {
					testSchema.RegisterResourceEventHandler("pre_create", eventHandler, 0)
				}).NotTo(Panic())
			})
		})

		Context("RegisterCustomEventHandler", func() {
			It("Should not panic when trying to register event as custom action", func() {
				Expect(func() {
					testSchema.RegisterCustomEventHandler("pre_create", customActionHandler, 0)
				}).NotTo(Panic())
			})

			It("Should not panic when trying to register custom action as custom action", func() {
				Expect(func() {
					testSchema.RegisterCustomEventHandler("echo", customActionHandler, 0)
				}).NotTo(Panic())
			})
		})
	})

	Context("Properties", func() {
		It("Should get correct properties", func() {
			Expect(testSchema.Properties()).To(Equal(
				[]goext.Property{
					{
						ID:    "id",
						Title: "ID",
						Type:  "string",
					},
					{
						ID:    "description",
						Title: "Description",
						Type:  "string",
					},
					{
						ID:    "name",
						Title: "Name",
						Type:  "string",
					},
					{
						ID:       "test_suite_id",
						Title:    "Test Suite ID",
						Relation: goext.SchemaID("test_suite"),
						Type:     "string",
					},
					{
						ID:    "subobject",
						Title: "Subobject",
						Type:  "object",
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

		It("Should count resources", func() {
			c, err := testSchema.Count(goext.Filter{}, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(uint64(0)))
			for i := 0; i < 2; i++ {
				createdResource.ID = string(i)
				createdResource.Name = goext.MakeString("test1")
				Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			}
			for i := 2; i < 5; i++ {
				createdResource.ID = string(i)
				createdResource.Name = goext.MakeString("test2")
				Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			}
			c, err = testSchema.Count(goext.Filter{}, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(uint64(5)))
			c, err = testSchema.Count(goext.Filter{"name": "test2"}, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(uint64(3)))
			c, err = testSchema.Count(goext.Filter{"name": "test1"}, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(uint64(2)))
		})

		It("Fetch previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			returnedResource, err := testSchema.FetchRaw(createdResource.ID, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(&createdResource).To(Equal(returnedResource))
		})

		It("FetchFilter previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			returnedResource, err := testSchema.FetchFilterRaw(goext.Filter{"description": "description"}, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(&createdResource).To(Equal(returnedResource))
		})

		It("DeleteRaw previously created resource", func() {
			Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())
			Expect(testSchema.DeleteRaw(createdResource.ID, context)).To(Succeed())
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

		It("should return error when trying to delete with empty filter", func() {
			err := testSchema.DeleteFilterRaw(goext.Filter{}, context)
			Expect(err).To(MatchError("Cannot delete with empty filter"))
		})

		Context("Query filters", func() {
			It("Should list resources using And", func() {
				Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())

				f := filter.And(
					filter.Eq("id", "some-id"),
					filter.Eq("description", "description"),
				)
				returnedResources, err := testSchema.ListRaw(f, nil, context)
				Expect(err).ToNot(HaveOccurred())
				Expect(returnedResources).To(HaveLen(1))
				returnedResource := returnedResources[0]
				Expect(&createdResource).To(Equal(returnedResource))
			})

			It("Should list resources using Or", func() {
				Expect(testSchema.CreateRaw(&createdResource, context)).To(Succeed())

				f := filter.Or(
					filter.Eq("id", "invalid-id"),
					filter.Eq("id", "some-id"),
				)
				returnedResources, err := testSchema.ListRaw(f, nil, context)
				Expect(err).ToNot(HaveOccurred())
				Expect(returnedResources).To(HaveLen(1))
				returnedResource := returnedResources[0]
				Expect(&createdResource).To(Equal(returnedResource))
			})

		})

		Context("Resource type not registered", func() {
			resourceID := "testId"
			resourceName := "testName"
			filter := map[string]interface{}{
				"id": resourceID,
			}

			BeforeEach(func() {
				tx, err := rawDB.Begin()
				Expect(err).ShouldNot(HaveOccurred())
				resource, err := schemaManager.LoadResource("test_schema_no_ext", map[string]interface{}{
					"id":   resourceID,
					"name": resourceName,
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

			It("should fail gracefully in FetchFilter", func() {
				_, err := testSchemaNoExtensions.FetchFilter(goext.Filter{"name": resourceName}, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))

			})

			It("should fail gracefully in FetchRaw", func() {
				_, err := testSchemaNoExtensions.FetchRaw(resourceID, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in FetchFilterRaw", func() {
				_, err := testSchemaNoExtensions.FetchFilterRaw(goext.Filter{"name": resourceName}, context)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))

			})

			It("should fail gracefully in LockFetch", func() {
				_, err := testSchemaNoExtensions.LockFetch(resourceID, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in LockFetchFilter", func() {
				_, err := testSchemaNoExtensions.LockFetchFilter(goext.Filter{"name": resourceName}, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))

			})

			It("should fail gracefully in LockFetchRaw", func() {
				_, err := testSchemaNoExtensions.LockFetchRaw(resourceID, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))
			})

			It("should fail gracefully in LockFetchFilterRaw", func() {
				_, err := testSchemaNoExtensions.LockFetchFilterRaw(goext.Filter{"name": resourceName}, context, goext.SkipRelatedResources)
				Expect(err.Error()).To(ContainSubstring("test_schema_no_ext"))
				Expect(err.Error()).To(ContainSubstring("not registered"))

			})
		})

		Context("Resource not found", func() {
			const (
				unknownID   = "unknown-id"
				unknownName = "unknown-name"
			)

			It("Should return error in Fetch", func() {
				_, err := testSchema.Fetch(unknownID, context)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in FetchFilter", func() {
				_, err := testSchema.FetchFilter(goext.Filter{"name": unknownName}, context)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in FetchRaw", func() {
				_, err := testSchema.FetchRaw(unknownID, context)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in FetchFilterRaw", func() {
				_, err := testSchema.FetchFilterRaw(goext.Filter{"name": unknownName}, context)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in LockFetch", func() {
				_, err := testSchema.LockFetch(unknownID, context, goext.SkipRelatedResources)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in LockFetchFilter", func() {
				_, err := testSchema.LockFetchFilter(goext.Filter{"name": unknownName}, context, goext.SkipRelatedResources)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in LockFetchRaw", func() {
				_, err := testSchema.LockFetchRaw(unknownID, context, goext.SkipRelatedResources)
				Expect(err).To(Equal(goext.ErrResourceNotFound))
			})

			It("Should return error in LockFetchFilterRaw", func() {
				_, err := testSchema.LockFetchFilterRaw(goext.Filter{"name": unknownName}, context, goext.SkipRelatedResources)
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
				Expect(testSchema.DeleteRaw(unknownID, context)).To(Succeed())
			})

			It("Should not return error in DeleteFilterRaw", func() {
				Expect(testSchema.DeleteFilterRaw(goext.Filter{"name": unknownName}, context)).To(Succeed())
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
				Expect(func() { testSchema.DeleteRaw(unknownID, goext.MakeContext()) }).To(Panic())
			})

			It("should panic when deleting resource with closed transaction", func() {
				Expect(tx.Close()).To(Succeed())
				Expect(func() { testSchema.DeleteRaw(unknownID, context) }).To(Panic())
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

			checkContext := func(ctx goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				Expect(&ctx).ToNot(Equal(&context))
				Expect(ctx).To(HaveKeyWithValue("transaction", tx))
				Expect(ctx).To(HaveKeyWithValue("test", 42))
				Expect(ctx).To(HaveKeyWithValue("schema_id", goext.SchemaID("test")))
				Expect(ctx).To(HaveKeyWithValue("id", "13"))
				Expect(ctx).To(HaveKeyWithValue("resource", env.Util().ResourceToMap(testResource)))
				Expect(resource).To(Equal(testResource))
				return nil
			}

			testSchema.RegisterResourceEventHandler(goext.PreCreateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler(goext.PostCreateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler(goext.PreUpdateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler(goext.PostUpdateTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler(goext.PreDeleteTx, checkContext, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler(goext.PostDeleteTx, checkContext, goext.PriorityDefault)
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
			Expect(testSchema.DeleteRaw(testResource.ID, context)).To(Succeed())
		})
	})
})
