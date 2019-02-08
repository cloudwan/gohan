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
	"reflect"
	"time"

	mock_db "github.com/cloudwan/gohan/db/mocks"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
	"github.com/cloudwan/gohan/schema"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

type MyRaw struct {
	ID          string `db:"id"`
	Description string `db:"description"`
}

var _ = Describe("Environment", func() {
	var (
		env     *goplugin.Environment
		manager *schema.Manager
	)

	const (
		schemaPath = "test_data/test_schema.yaml"
	)

	BeforeEach(func() {
		manager = schema.GetManager()
		Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
		env = goplugin.NewEnvironment("test", nil, nil)
	})

	Describe("Internal state consistency", func() {
		It("should be able to use logger when no plugins are loaded", func() {
			Expect(env.Start()).To(Succeed())
			env.Logger().Info("A message from logger just after environment Start()")
			env.Reset()
			env.Logger().Info("A message from logger just after environment Reset()")
			env.Stop()
		})
	})

	AfterEach(func() {
		schema.ClearManager()
	})

	Describe("Loading an extension", func() {
		Context("File paths are corrupted", func() {
			It("should not load plugin with wrong file extension", func() {
				err := env.Load("/wrong/extension.not-so")
				Expect(err.Error()).To(Equal("go extension must be a *.so file, file: /wrong/extension.not-so"))
			})

			It("should not load plugin from non-existing file", func() {
				err := env.Load("/non/existing-plugin.so")

				// Path can be surrounded by "" or not, depending on the GO version
				Expect(err.Error()).To(ContainSubstring("failed to load go extension: plugin.Open("))
				Expect(err.Error()).To(ContainSubstring("/non/existing-plugin.so"))
				Expect(err.Error()).To(ContainSubstring("): realpath failed"))
			})
		})

		Context("File paths are valid", func() {
			It("should load, start and stop plugin which does export Init and Schemas functions", func() {
				err := env.Load("test_data/ext_good/ext_good.so")
				Expect(err).To(BeNil())
				Expect(env.Start()).To(Succeed())
				env.Stop()
			})

			It("should not load plugin which does not export Init function", func() {
				err := env.Load("test_data/ext_no_init/ext_no_init.so")
				Expect(err.Error()).To(ContainSubstring("symbol Init not found"))
			})
		})
	})

	Describe("Registering event handlers", func() {
		var (
			testSchema goext.ISchema
		)

		BeforeEach(func() {
			err := env.Load("test_data/ext_good/ext_good.so")
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))
		})

		AfterEach(func() {
			env.Stop()
		})

		It("should register event handler on environment", func() {
			handler := func(context goext.Context, environment goext.IEnvironment) *goext.Error {
				return nil
			}

			Expect(len(env.Handlers())).To(Equal(0))
			env.RegisterEventHandler("some_event", handler, goext.PriorityDefault)
			Expect(len(env.Handlers()["some_event"][goext.PriorityDefault])).To(Equal(1))

			p1 := reflect.ValueOf(handler).Pointer()
			p2 := reflect.ValueOf(env.Handlers()["some_event"][goext.PriorityDefault][0]).Pointer()

			Expect(p1).To(Equal(p2))
		})

		It("should register event handler on schema", func() {
			handler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				return nil
			}

			testSchema.RegisterResourceEventHandler("some_event", handler, goext.PriorityDefault)
			Expect(len(env.SchemaHandlers()["some_event"]["test"][goext.PriorityDefault])).To(Equal(1))

			p1 := reflect.ValueOf(handler).Pointer()
			p2 := reflect.ValueOf(env.SchemaHandlers()["some_event"]["test"][goext.PriorityDefault][0]).Pointer()

			Expect(p1).To(Equal(p2))
		})
	})

	Describe("Running event handlers", func() {
		var (
			testSchema goext.ISchema
		)

		BeforeEach(func() {
			err := env.Load("test_data/ext_good/ext_good.so")
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))
		})

		AfterEach(func() {
			env.Stop()
		})

		It("should run event handlers registered on environment", func() {
			var someEventRunCount int = 0
			var someOtherEventRunCount int = 0

			someEventHandler := func(context goext.Context, environment goext.IEnvironment) *goext.Error {
				someEventRunCount++
				return nil
			}

			someOtherEventHandler := func(context goext.Context, environment goext.IEnvironment) *goext.Error {
				someOtherEventRunCount++
				return nil
			}

			env.RegisterEventHandler("some_event", someEventHandler, goext.PriorityDefault)
			env.RegisterEventHandler("some_other_event", someOtherEventHandler, goext.PriorityDefault)

			Expect(env.HandleEvent("some_event", make(map[string]interface{}))).To(Succeed())

			Expect(someEventRunCount).To(Equal(1))
			Expect(someOtherEventRunCount).To(Equal(0))

			Expect(env.HandleEvent("some_other_event", make(map[string]interface{}))).To(Succeed())

			Expect(someEventRunCount).To(Equal(1))
			Expect(someOtherEventRunCount).To(Equal(1))
		})

		It("should run event handlers registered on schema", func() {
			var someEventRunCount int = 0
			var someOtherEventRunCount int = 0

			someEventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				someEventRunCount++
				return nil
			}

			someOtherEventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				someOtherEventRunCount++
				return nil
			}

			testSchema.RegisterResourceEventHandler("some_event", someEventHandler, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler("some_other_event", someOtherEventHandler, goext.PriorityDefault)

			Expect(env.HandleEvent("some_event", goext.MakeContext())).To(Succeed())

			Expect(someEventRunCount).To(Equal(1))
			Expect(someOtherEventRunCount).To(Equal(0))

			Expect(env.HandleEvent("some_other_event", goext.MakeContext())).To(Succeed())

			Expect(someEventRunCount).To(Equal(1))
			Expect(someOtherEventRunCount).To(Equal(1))
		})

		It("should pass data from context to handler", func() {
			var returnedResource *test.Test

			eventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				returnedResource = resource.(*test.Test)
				return nil
			}

			testSchema.RegisterResourceEventHandler("some_event", eventHandler, goext.PriorityDefault)

			context := goext.MakeContext()
			resource := make(map[string]interface{})
			resource["id"] = "some-id"
			resource["description"] = "some description"
			resource["subobject"] = map[string]interface{}{"subproperty": "subvalue"}
			context = context.WithResource(resource)

			Expect(env.HandleEvent("some_event", context)).To(Succeed())

			Expect(returnedResource).To(Not(BeNil()))
			Expect(returnedResource.ID).To(Equal("some-id"))
			Expect(returnedResource.Description).To(Equal("some description"))
			Expect(returnedResource.Subobject.Subproperty).To(Equal("subvalue"))
		})

		It("should update resource from context after each event dispatched", func() {
			var returnedResource *test.Test

			eventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				returnedResource = resource.(*test.Test)
				return nil
			}

			modifingHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				res := resource.(*test.Test)
				res.ID = "other-id"
				return nil
			}

			testSchema.RegisterResourceEventHandler("some_event", eventHandler, goext.PriorityDefault)
			testSchema.RegisterResourceEventHandler("some_event", modifingHandler, goext.PriorityDefault+1)

			context := goext.MakeContext()
			resource := make(map[string]interface{})
			resource["id"] = "some-id"
			resource["description"] = "some description"
			context = context.WithResource(resource)

			Expect(env.HandleEvent("some_event", context)).To(Succeed())

			Expect(returnedResource).To(Not(BeNil()))
			Expect(returnedResource.ID).To(Equal("other-id"))
			Expect(returnedResource.Description).To(Equal("some description"))
		})

		It("handlers are executed in priority order", func() {
			var prioritizedCalled, defaultCalled bool
			errWrongOrder := errors.New("wrong order of execution. Prioritized handler should be called first")
			prioritizedEventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				if defaultCalled {
					return goext.NewErrorInternalServerError(errWrongOrder)
				}
				prioritizedCalled = true
				return nil
			}

			defaultPriorityEventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) *goext.Error {
				if !prioritizedCalled {
					return goext.NewErrorInternalServerError(errWrongOrder)
				}
				defaultCalled = true
				return nil
			}

			testSchema.RegisterResourceEventHandler("some_event", prioritizedEventHandler, goext.PriorityDefault-100)
			testSchema.RegisterResourceEventHandler("some_event", defaultPriorityEventHandler, goext.PriorityDefault)

			context := goext.MakeContext()
			Expect(env.HandleEvent("some_event", context)).To(Succeed())
			Expect(prioritizedCalled).To(BeTrue())
			Expect(defaultCalled).To(BeTrue())
		})

		Context("Cloning", func() {
			var (
				mockCtrl *gomock.Controller
				mockDB   *mock_db.MockDB
				env      *goplugin.Environment
			)

			BeforeEach(func() {
				mockCtrl = gomock.NewController(GinkgoT())
				mockDB = mock_db.NewMockDB(mockCtrl)

				env = goplugin.NewEnvironment("cloning_env", nil, nil)
			})

			It("should correctly clone runtime types", func() {
				env.RegisterRawType("my_raw", MyRaw{})

				clone := env.Clone().(*goplugin.Environment)

				Expect(len(clone.RawTypes())).To(Equal(1))
				Expect(clone.RawTypes()["my_raw"]).To(Equal(env.RawTypes()["my_raw"]))
			})

			It("should clone database options", func() {
				expectedOptions := goext.DbOptions{RetryTxCount: 1, RetryTxInterval: 2}
				mockDB.EXPECT().Options().Return(options.Options{
					RetryTxCount:    expectedOptions.RetryTxCount,
					RetryTxInterval: expectedOptions.RetryTxInterval,
				})

				env.SetDatabase(mockDB)
				clone := env.Clone().(*goplugin.Environment)

				Expect(clone.Database().Options()).To(Equal(expectedOptions))
			})
		})

		Context("execution termination", func() {
			It("should exit gracefully when HTTP peer disconnects", func() {
				ctx, cancel := context.WithCancel(context.Background())

				context := goext.MakeContext()
				context["context"] = ctx

				done := make(chan bool, 1)
				go func() {
					defer GinkgoRecover()
					Expect(env.HandleEvent("wait_for_context_cancel", context)).To(Succeed())
					done <- true
				}()

				cancel()
				Eventually(done).Should(Receive())
			})

			It("should exit gracefully on global execution timeout", func() {
				err := env.LoadExtensionsForPath(manager.Extensions, time.Millisecond*100, nil, "wait_for_context_cancel")
				Expect(err).To(Succeed())

				ctx := context.Background()
				context := goext.MakeContext()
				context["context"] = ctx

				done := make(chan bool, 1)
				go func() {
					defer GinkgoRecover()
					Expect(env.HandleEvent("wait_for_context_cancel", context)).To(Succeed())
					done <- true
				}()

				Eventually(done, time.Millisecond*500).Should(Receive())
			})

			It("should exit gracefully on path execution timeout", func() {
				timeLimits := []*schema.PathEventTimeLimit{
					schema.NewPathEventTimeLimit(".*", "wait_for_context_cancel", time.Millisecond*100),
				}

				err := env.LoadExtensionsForPath(manager.Extensions, 0, timeLimits, "wait_for_context_cancel")
				Expect(err).To(Succeed())

				ctx := context.Background()
				context := goext.MakeContext()
				context["context"] = ctx

				done := make(chan bool, 1)
				go func() {
					defer GinkgoRecover()
					Expect(env.HandleEvent("wait_for_context_cancel", context)).To(Succeed())
					done <- true
				}()

				Eventually(done, time.Millisecond*500).Should(Receive())
			})

			It("should not overwrite existing timeouts", func() {
				err := env.LoadExtensionsForPath(manager.Extensions, time.Hour, nil, "wait_for_context_cancel")
				Expect(err).To(Succeed())

				requestContext := goext.MakeContext()
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
				requestContext["context"] = ctx

				done := make(chan bool, 1)
				go func() {
					defer GinkgoRecover()
					Expect(env.HandleEvent("wait_for_context_cancel", requestContext)).To(Succeed())
					done <- true
				}()

				cancel()
				Eventually(done, time.Millisecond*500).Should(Receive())
			})
		})
	})
})

type SimpleCloseNotifier struct {
	closeCh chan bool
}

func (s SimpleCloseNotifier) CloseNotify() <-chan bool {
	return s.closeCh
}

func (s SimpleCloseNotifier) Close() {
	s.closeCh <- true
}
