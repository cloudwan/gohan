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

package otto_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	db_mocks "github.com/cloudwan/gohan/db/mocks"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	tr_mocks "github.com/cloudwan/gohan/db/transaction/mocks"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func newEnvironmentWithExtension(extension *schema.Extension, db db.DB) (env extension.Environment) {
	timeLimit := time.Duration(1) * time.Second
	timeLimits := []*schema.PathEventTimeLimit{}
	extensions := []*schema.Extension{extension}
	env = otto.NewEnvironment("db_test", db, &middleware.FakeIdentity{}, testSync)
	Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
	return
}

var _ = Describe("GohanDb", func() {
	var (
		manager       *schema.Manager
		s             *schema.Schema
		ok            bool
		fakeResources []map[string]interface{}
		err           error
		r0, r1        *schema.Resource
		mockCtrl      *gomock.Controller
	)

	var ()

	BeforeEach(func() {
		manager = schema.GetManager()
		s, ok = manager.Schema("test")
		Expect(ok).To(BeTrue())
		mockCtrl = gomock.NewController(GinkgoT())

		fakeResources = []map[string]interface{}{
			{"tenant_id": "t0", "test_string": "str0", "test_bool": false},
			{"tenant_id": "t1", "test_string": "str1", "test_bool": true},
		}

		r0, err = schema.NewResource(s, fakeResources[0])
		Expect(err).ToNot(HaveOccurred())
		r1, err = schema.NewResource(s, fakeResources[1])
		Expect(err).ToNot(HaveOccurred())

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("gohan_db_transaction", func() {
		Context("When no argument is given", func() {
			It("doesn't run SetIsolationLevel method", func() {
				ext, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = gohan_db_transaction();
					    tx.Commit();
					    tx.Close();
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)
				gomock.InOrder(
					mockTx.EXPECT().Commit().Return(nil),
					mockTx.EXPECT().Close().Return(nil).Times(2),
				)

				mockDB := db_mocks.NewMockDB(mockCtrl)
				mockDB.EXPECT().Begin(gomock.Any()).Return(mockTx, nil)
				env := newEnvironmentWithExtension(ext, mockDB)

				context := map[string]interface{}{}

				Expect(env.HandleEvent("test_event", context)).To(Succeed())
			})
		})

		Context("When proper isolation level argument is given", func() {
			It("runs SetIsolationLevel method", func() {
				ext, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = gohan_db_transaction("SERIALIZABLE");
					    tx.Commit();
					    tx.Close();
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)
				gomock.InOrder(
					mockTx.EXPECT().Commit().Return(nil),
					mockTx.EXPECT().Close().Return(nil).Times(2),
				)

				mockDB := db_mocks.NewMockDB(mockCtrl)
				mockDB.EXPECT().Begin(gomock.Any()).DoAndReturn(func(opt ...transaction.OptionTxParams) (transaction.Transaction, error) {
					Expect(len(opt)).To(Equal(1))
					params := &transaction.TxParams{}
					opt[0](params)
					Expect(params.IsolationLevel).To(Equal(transaction.Serializable))
					return mockTx, nil
				})
				env := newEnvironmentWithExtension(ext, mockDB)

				context := map[string]interface{}{
					"transaction": mockTx,
				}

				Expect(env.HandleEvent("test_event", context)).To(Succeed())
			})
		})

	})

	Describe("gohan_db_(lock)list", func() {
		var listCall = func(tx *tr_mocks.MockTransaction, methodName string, s *schema.Schema, f transaction.Filter, pg *pagination.Paginator) *gomock.Call {
			if strings.Contains(methodName, "Lock") {
				return tx.EXPECT().LockList(s, f, nil, pg, schema.SkipRelatedResources)
			}
			return tx.EXPECT().List(s, f, nil, pg)
		}

		Context("When valid minimum parameters are given", func() {
			DescribeTable("returns the list ordered by id",
				func(function, methodName string) {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": fmt.Sprintf(`
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = %s(
					      tx,
					      "test",
					      {"tenant_id": "tenant0"}
					    );
					  });`, function),
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					env := newEnvironmentWithExtension(extension, testDB)

					mockTx := tr_mocks.NewMockTransaction(mockCtrl)
					pg, err := pagination.NewPaginator(pagination.OptionOrder(pagination.ASC))
					Expect(err).To(Succeed())
					listCall(mockTx, methodName, s, transaction.Filter{"tenant_id": "tenant0"}, pg).Return(
						[]*schema.Resource{r0, r1},
						uint64(2),
						nil,
					)

					context := map[string]interface{}{
						"transaction": mockTx,
					}

					Expect(env.HandleEvent("test_event", context)).To(Succeed())
					Expect(context["resp"]).To(
						Equal(
							fakeResources,
						),
					)
				},
				Entry("gohan_db_list", "gohan_db_list", "List"),
				Entry("gohan_db_lock_list", "gohan_db_lock_list", "LockList"),
			)
		})

		Context("When boolean parameter is given", func() {
			DescribeTable("returns the list test_bool is true",
				func(function, methodName string) {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": fmt.Sprintf(`
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = %s(
					      tx,
					      "test",
					      {"test_bool": true}
					    );
					  });`, function),
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					env := newEnvironmentWithExtension(extension, testDB)

					mockTx := tr_mocks.NewMockTransaction(mockCtrl)
					pg, err := pagination.NewPaginator(pagination.OptionOrder(pagination.ASC))
					Expect(err).To(Succeed())
					listCall(mockTx, methodName, s, transaction.Filter{"test_bool": true}, pg).Return(
						[]*schema.Resource{r1},
						uint64(1),
						nil,
					)

					context := map[string]interface{}{
						"transaction": mockTx,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())

					Expect(context["resp"]).To(
						Equal(
							[]map[string]interface{}{
								{"tenant_id": "t1", "test_string": "str1", "test_bool": true},
							},
						),
					)
				},
				Entry("gohan_db_list", "gohan_db_list", "List"),
				Entry("gohan_db_lock_list", "gohan_db_lock_list", "LockList"),
			)
		})

		Context("When order key parameter is not given", func() {
			DescribeTable("returns the list limited to given limit",
				func(function, methodName string) {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": fmt.Sprintf(`
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = %s(
					      tx,
					      "test",
					      {"tenant_id": "tenant0"},
					      "",
					      1
					    );
					  });`, function),
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					env := newEnvironmentWithExtension(extension, testDB)

					mockTx := tr_mocks.NewMockTransaction(mockCtrl)
					pg, _ := pagination.NewPaginator(pagination.OptionOrder(pagination.ASC), pagination.OptionLimit(1))
					listCall(mockTx, methodName, s, transaction.Filter{"tenant_id": "tenant0"}, pg).Return(
						[]*schema.Resource{r0},
						uint64(2),
						nil,
					)

					context := map[string]interface{}{
						"transaction": mockTx,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())

					Expect(context["resp"]).To(
						Equal(
							[]map[string]interface{}{fakeResources[0]},
						),
					)
				},
				Entry("gohan_db_list", "gohan_db_list", "List"),
				Entry("gohan_db_lock_list", "gohan_db_lock_list", "LockList"),
			)
		})

		Context("When 4 parameters are given", func() {
			DescribeTable("returns the list ordered by given column",
				func(function, methodName string) {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": fmt.Sprintf(`
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = %s(
					      tx,
					      "test",
					      {"tenant_id": "tenant0"},
					      "test_string"
					    );
					  });`, function),
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					env := newEnvironmentWithExtension(extension, testDB)

					mockTx := tr_mocks.NewMockTransaction(mockCtrl)
					pg, _ := pagination.NewPaginator(
						pagination.OptionKey(nil, "test_string"),
						pagination.OptionOrder(pagination.ASC))

					listCall(mockTx, methodName, s, transaction.Filter{"tenant_id": "tenant0"}, pg).Return(
						[]*schema.Resource{r0, r1},
						uint64(2),
						nil,
					)

					context := map[string]interface{}{
						"transaction": mockTx,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())

					Expect(context["resp"]).To(
						Equal(
							fakeResources,
						),
					)
				},
				Entry("gohan_db_list", "gohan_db_list", "List"),
				Entry("gohan_db_lock_list", "gohan_db_lock_list", "LockList"),
			)
		})

		Context("When 5 parameters are given", func() {
			DescribeTable("returns the list ordered by given column and limited",
				func(function, methodName string) {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": fmt.Sprintf(`
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = %s(
					      tx,
					      "test",
					      {"tenant_id": "tenant0"},
					      "test_string",
					      100
					    );
					  });`, function),
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					env := newEnvironmentWithExtension(extension, testDB)

					mockTx := tr_mocks.NewMockTransaction(mockCtrl)
					pg, _ := pagination.NewPaginator(
						pagination.OptionKey(nil, "test_string"),
						pagination.OptionOrder(pagination.ASC),
						pagination.OptionLimit(100))

					listCall(mockTx, methodName, s, transaction.Filter{"tenant_id": "tenant0"}, pg).Return(
						[]*schema.Resource{r0, r1},
						uint64(2),
						nil,
					)

					context := map[string]interface{}{
						"transaction": mockTx,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())

					Expect(context["resp"]).To(
						Equal(
							fakeResources,
						),
					)
				},
				Entry("gohan_db_list", "gohan_db_list", "List"),
				Entry("gohan_db_lock_list", "gohan_db_lock_list", "LockList"),
			)
		})

		Context("When 6 parameters are given", func() {
			DescribeTable("returns the list ordered by given column and limited with offset",
				func(function, methodName string) {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": fmt.Sprintf(`
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = %s(
					      tx,
					      "test",
					      {"tenant_id": "tenant0"},
					      "test_string",
					      100,
					      10
					    );
					  });`, function),
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					env := newEnvironmentWithExtension(extension, testDB)

					mockTx := tr_mocks.NewMockTransaction(mockCtrl)
					pg, _ := pagination.NewPaginator(
						pagination.OptionKey(nil, "test_string"),
						pagination.OptionOrder(pagination.ASC),
						pagination.OptionLimit(100),
						pagination.OptionOffset(10))

					listCall(mockTx, methodName, s, transaction.Filter{"tenant_id": "tenant0"}, pg).Return(
						[]*schema.Resource{r0, r1},
						uint64(2),
						nil,
					)

					context := map[string]interface{}{
						"transaction": mockTx,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())

					Expect(context["resp"]).To(
						Equal(
							fakeResources,
						),
					)
				},
				Entry("gohan_db_list", "gohan_db_list", "List"),
				Entry("gohan_db_lock_list", "gohan_db_lock_list", "LockList"),
			)
		})

	})

	Describe("gohan_db_state_fetch", func() {
		Context("When valid parameters are given", func() {
			It("returns a resource state object", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = gohan_db_state_fetch(
					      tx,
					      "test",
					      "resource_id",
					      "tenant0"
					    );
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)
				mockTx.EXPECT().StateFetch(s, transaction.Filter{"id": "resource_id", "tenant_id": "tenant0"}).Return(
					transaction.ResourceState{
						ConfigVersion: 30,
						StateVersion:  29,
						Error:         "e",
						State:         "s",
						Monitoring:    "m",
					},
					nil,
				)

				context := map[string]interface{}{
					"transaction": mockTx,
				}

				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["resp"]).To(
					Equal(
						map[string]interface{}{
							"config_version": int64(30),
							"state_version":  int64(29),
							"error":          "e",
							"state":          "s",
							"monitoring":     "m",
						},
					),
				)
			})
		})
	})

	Describe("gohan_db_sql_make_columns", func() {
		Context("when a valid schema ID is given", func() {
			It("returns column names in Gohan DB compatible format", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    context.resp = gohan_db_sql_make_columns("test");
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				context := map[string]interface{}{}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["resp"]).To(ContainElement("tests.`id` as `tests__id`"))
				Expect(context["resp"]).To(ContainElement("tests.`tenant_id` as `tests__tenant_id`"))
				Expect(context["resp"]).To(ContainElement("tests.`test_string` as `tests__test_string`"))
			})
		})

		Context("when an invalid schema ID is given", func() {
			It("returns an error", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    context.resp = gohan_db_sql_make_columns("NOT EXIST");
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				context := map[string]interface{}{}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Error: Unknown schema 'NOT EXIST'"))
			})
		})

	})

	Describe("gohan_db_query", func() {
		Context("when valid parameters are given", func() {
			It("returns resources in db", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = gohan_db_query(
					      tx,
					      "test",
					      "SELECT DUMMY",
					      ["tenant0", "obj1"]
					    );
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)
				mockTx.EXPECT().Query(s, "SELECT DUMMY", []interface{}{"tenant0", "obj1"}).Return(
					[]*schema.Resource{r0, r1}, nil,
				)

				context := map[string]interface{}{
					"transaction": mockTx,
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["resp"]).To(Equal(fakeResources))
			})
		})

		Context("When an invalid transaction is provided", func() {
			It("fails and return an error", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = gohan_db_query(
					      tx,
					      "test",
					      "SELECT DUMMY",
					      ["tenant0", "obj1"]
					    );
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				context := map[string]interface{}{
					"transaction": "not_a_transaction",
				}

				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Error: Argument 'not_a_transaction' should be of type 'Transaction'"))
			})
		})

		Context("When an invalid schema ID is provided", func() {
			It("fails and return an error", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = gohan_db_query(
					      tx,
					      "INVALID_SCHEMA_ID",
					      "SELECT DUMMY",
					      ["tenant0", "obj1"]
					    );
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)

				context := map[string]interface{}{
					"transaction": mockTx,
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Error: Unknown schema 'INVALID_SCHEMA_ID'"))
			})
		})

		Context("When an invalid array is provided to arguments", func() {
			It("fails and return an error", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = gohan_db_query(
					      tx,
					      "test",
					      "SELECT DUMMY",
					      "THIS IS NOT AN ARRAY"
					    );
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)

				context := map[string]interface{}{
					"transaction": mockTx,
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Error: Argument 'THIS IS NOT AN ARRAY' should be of type 'array'"))
			})
		})

		Context("When an error occurred while processing the query", func() {
			It("fails and return an error", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
					  gohan_register_handler("test_event", function(context){
					    var tx = context.transaction;
					    context.resp = gohan_db_query(
					      tx,
					      "test",
					      "SELECT DUMMY",
					      []
					    );
					  });`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				env := newEnvironmentWithExtension(extension, testDB)

				mockTx := tr_mocks.NewMockTransaction(mockCtrl)
				mockTx.EXPECT().Query(s, "SELECT DUMMY", []interface{}{}).Return(
					nil, errors.New("SOMETHING HAPPENED"),
				)

				context := map[string]interface{}{
					"transaction": mockTx,
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Error: Error during gohan_db_query: SOMETHING HAPPEN"))
			})
		})
	})
})
