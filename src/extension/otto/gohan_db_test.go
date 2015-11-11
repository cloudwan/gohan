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

	"db/transaction"
	"db/transaction/mocks"
	"extension/otto"
	"schema"
	"server/middleware"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GohanDb", func() {

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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

				manager := schema.GetManager()
				s, ok := manager.Schema("test")
				Expect(ok).To(BeTrue())

				var fakeTx = new(mocks.Transaction)
				fakeTx.On(
					"StateFetch", s, "resource_id", []string{"tenant0"},
				).Return(
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
					"transaction": fakeTx,
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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

				context := map[string]interface{}{}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Unknown schema 'NOT EXIST'"))
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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

				manager := schema.GetManager()
				s, ok := manager.Schema("test")
				Expect(ok).To(BeTrue())

				fakeResources := []map[string]interface{}{
					map[string]interface{}{"tenant_id": "t0", "test_string": "str0"},
					map[string]interface{}{"tenant_id": "t1", "test_string": "str1"},
				}

				r0, err := schema.NewResource(s, fakeResources[0])
				Expect(err).ToNot(HaveOccurred())
				r1, err := schema.NewResource(s, fakeResources[1])
				Expect(err).ToNot(HaveOccurred())

				var fakeTx = new(mocks.Transaction)
				fakeTx.On(
					"Query", s, "SELECT DUMMY", []interface{}{"tenant0", "obj1"},
				).Return(
					[]*schema.Resource{r0, r1}, nil,
				)

				context := map[string]interface{}{
					"transaction": fakeTx,
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["resp"]).To(Equal(fakeResources))
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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"transaction": new(mocks.Transaction),
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Unknown schema 'INVALID_SCHEMA_ID'"))
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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"transaction": new(mocks.Transaction),
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Gievn arguments is not \\[\\]interface\\{\\}"))
			})
		})

		Context("When an error occured while processing the query", func() {
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
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment(testDB, &middleware.FakeIdentity{})
				Expect(env.LoadExtensionsForPath(extensions, "test_path")).To(Succeed())

				manager := schema.GetManager()
				s, ok := manager.Schema("test")
				Expect(ok).To(BeTrue())

				var fakeTx = new(mocks.Transaction)
				fakeTx.On(
					"Query", s, "SELECT DUMMY", []interface{}{},
				).Return(
					nil, errors.New("SOMETHING HAPPENED"),
				)

				context := map[string]interface{}{
					"transaction": fakeTx,
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(MatchRegexp("test_event: Error during gohan_db_query: SOMETHING HAPPEN"))
			})
		})

	})
})