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

package golang_test

import (
	"fmt"
	"reflect"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/golang"
	"github.com/cloudwan/gohan/extension/golang/test_data/ext_good/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment", func() {
	var (
		env *golang.Environment
	)

	BeforeEach(func() {
		env = golang.NewEnvironment("test", nil, nil, nil)
	})

	Describe("Loading an extension", func() {
		Context("File paths are corrupted", func() {
			It("should not load plugin with wrong file extension", func() {
				loaded, err := env.Load("/wrong/extension.not-so", nil)
				Expect(loaded).To(BeFalse())
				Expect(err).To(Equal(fmt.Errorf("golang extensions source code must be a *.so file, source: /wrong/extension.not-so")))
			})

			It("should not load plugin from non-existing file", func() {
				loaded, err := env.Load("/non/existing-plugin.so", nil)
				Expect(loaded).To(BeFalse())
				Expect(err).To(Equal(fmt.Errorf("failed to load golang extension: plugin.Open(/non/existing-plugin.so): realpath failed")))
			})
		})

		Context("File paths are valid", func() {
			It("should load, start and stop plugin which does export Init and Schemas functions", func() {
				loaded, err := env.Load("test_data/ext_good/ext_good.so", nil)
				Expect(loaded).To(BeTrue())
				Expect(err).To(BeNil())
				Expect(env.Start()).To(Succeed())
				env.Stop()
			})

			It("should not load plugin which does not export Init function", func() {
				loaded, err := env.Load("test_data/ext_no_init/ext_no_init.so", nil)
				Expect(loaded).To(BeFalse())
				Expect(err.Error()).To(ContainSubstring("symbol Init not found"))
			})

			It("should not load plugin which does not export Schemas function", func() {
				loaded, err := env.Load("test_data/ext_no_schemas/ext_no_schemas.so", nil)
				Expect(loaded).To(BeFalse())
				Expect(err.Error()).To(ContainSubstring("symbol Schemas not found"))
			})
		})
	})

	Describe("Registering event handlers", func() {
		var (
			testSchema goext.ISchema
		)

		BeforeEach(func() {
			loaded, err := env.Load("test_data/ext_good/ext_good.so", nil)
			Expect(loaded).To(BeTrue())
			Expect(err).To(BeNil())
			Expect(env.Start()).To(Succeed())
			testSchema = env.Schemas().Find("test")
			Expect(testSchema).To(Not(BeNil()))
		})

		AfterEach(func() {
			env.Stop()
		})

		It("should register event handler on environment", func() {
			handler := func(context goext.Context, environment goext.IEnvironment) error {
				return nil
			}

			Expect(len(golang.GlobHandlers)).To(Equal(0))
			env.RegisterEventHandler("some_event", handler, goext.PriorityDefault)
			Expect(len(golang.GlobHandlers["some_event"][goext.PriorityDefault])).To(Equal(1))

			p1 := reflect.ValueOf(handler).Pointer()
			p2 := reflect.ValueOf(golang.GlobHandlers["some_event"][goext.PriorityDefault][0]).Pointer()

			Expect(p1).To(Equal(p2))
		})

		It("should register event handler on schema", func() {
			handler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error {
				return nil
			}

			testSchema.RegisterEventHandler("some_event", handler, goext.PriorityDefault)
			Expect(len(golang.GlobSchemaHandlers["some_event"]["test"][goext.PriorityDefault])).To(Equal(1))

			p1 := reflect.ValueOf(handler).Pointer()
			p2 := reflect.ValueOf(golang.GlobSchemaHandlers["some_event"]["test"][goext.PriorityDefault][0]).Pointer()

			Expect(p1).To(Equal(p2))
		})
	})

	Describe("Running event handlers", func() {
		var (
			testSchema goext.ISchema
		)

		BeforeEach(func() {
			loaded, err := env.Load("test_data/ext_good/ext_good.so", nil)
			Expect(loaded).To(BeTrue())
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

			someEventHandler := func(context goext.Context, environment goext.IEnvironment) error {
				someEventRunCount++
				return nil
			}

			someOtherEventHandler := func(context goext.Context, environment goext.IEnvironment) error {
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

			someEventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error {
				someEventRunCount++
				return nil
			}

			someOtherEventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error {
				someOtherEventRunCount++
				return nil
			}

			testSchema.RegisterEventHandler("some_event", someEventHandler, goext.PriorityDefault)
			testSchema.RegisterEventHandler("some_other_event", someOtherEventHandler, goext.PriorityDefault)

			Expect(env.HandleEvent("some_event", goext.MakeContext())).To(Succeed())

			Expect(someEventRunCount).To(Equal(1))
			Expect(someOtherEventRunCount).To(Equal(0))

			Expect(env.HandleEvent("some_other_event", goext.MakeContext())).To(Succeed())

			Expect(someEventRunCount).To(Equal(1))
			Expect(someOtherEventRunCount).To(Equal(1))
		})

		It("should pass data from context to handler", func() {
			var returnedResource *test.Test

			eventHandler := func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error {
				returnedResource = resource.(*test.Test)
				return nil
			}

			testSchema.RegisterEventHandler("some_event", eventHandler, goext.PriorityDefault)

			context := goext.MakeContext()
			resource := make(map[string]interface{})
			resource["id"] = "some-id"
			resource["description"] = "some description"
			context = context.WithResource(resource)

			Expect(env.HandleEvent("some_event", context)).To(Succeed())

			Expect(returnedResource).To(Not(BeNil()))
			Expect(returnedResource.ID).To(Equal("some-id"))
			Expect(returnedResource.Description).To(Equal("some description"))
		})
	})
})
