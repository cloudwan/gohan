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
		env    *goplugin.Environment
		dbFile string
		dbType string
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

		manager := schema.GetManager()
		Expect(manager.LoadSchemaFromFile(SchemaPath)).To(Succeed())
		db, err := db.ConnectDB(dbType, dbFile, db.DefaultMaxOpenConn, options.Default())
		Expect(err).To(BeNil())
		env = goplugin.NewEnvironment("test", nil, nil)
		env.SetDatabase(db)
	})

	AfterEach(func() {
		os.Remove(dbFile)

	})

	Context("CRUD", func() {
		var (
			testSchema      goext.ISchema
			createdResource test.Test
			context         goext.Context
			tx              goext.ITransaction
		)

		BeforeEach(func() {
			err := env.Load("test_data/ext_good/ext_good.so")
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))

			err = db.InitDBWithSchemas(dbType, dbFile, true, true, false)
			Expect(err).To(BeNil())

			tx, err = env.Database().Begin()
			Expect(err).To(BeNil())

			context = goext.MakeContext().WithTransaction(tx)

			createdResource = test.Test{
				ID:          "some-id",
				Description: "description",
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

			err := env.Load("test_data/ext_good/ext_good.so")
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))

			err = db.InitDBWithSchemas(dbType, dbFile, true, true, false)
			Expect(err).To(BeNil())

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

})
