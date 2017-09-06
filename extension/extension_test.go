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

package extension_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/server/middleware"
)

var _ = Describe("Environment manager", func() {
	const (
		schemaID1 = "Wormtongue"
		schemaID2 = "Dumbledore"
	)
	var (
		env1    extension.Environment
		env2    extension.Environment
		manager *extension.Manager
	)

	BeforeEach(func() {
		env1 = otto.NewEnvironment("extension_test1", testDB1, &middleware.FakeIdentity{}, testSync)
		env2 = otto.NewEnvironment("extension_test2", testDB2, &middleware.FakeIdentity{}, testSync)
	})

	JustBeforeEach(func() {
		manager = extension.GetManager()
		Expect(manager.RegisterEnvironment(schemaID1, env1)).To(Succeed())
	})

	AfterEach(func() {
		extension.ClearManager()
	})

	Describe("Registering environments", func() {
		Context("When it isn't registered", func() {
			It("Should register it", func() {
				Expect(manager.RegisterEnvironment(schemaID2, env2)).To(Succeed())
			})
		})

		Context("When it is registered", func() {
			It("Should return an error", func() {
				err := manager.RegisterEnvironment(schemaID1, env2)
				Expect(err).To(MatchError("Environment already registered for schema 'Wormtongue'"))
			})
		})
	})

	Describe("Getting an environment", func() {
		Context("When it is registered", func() {
			It("Should return it", func() {
				env, ok := manager.GetEnvironment(schemaID1)
				Expect(ok).To(BeTrue())
				Expect(env).NotTo(BeNil())
			})
		})

		Context("When it isn't registered", func() {
			It("Should return a false ok", func() {
				_, ok := manager.GetEnvironment(schemaID2)
				Expect(ok).To(BeFalse())
			})
		})
	})

	Describe("Unregistering an environment", func() {
		Context("When it is registered", func() {
			It("Should unregister it", func() {
				Expect(manager.UnRegisterEnvironment(schemaID1)).To(Succeed())
			})
		})

		Context("When it isn't registered", func() {
			It("Should return an error", func() {
				err := manager.UnRegisterEnvironment(schemaID2)
				Expect(err).To(MatchError("No environment registered for this schema"))
			})
		})
	})

	Describe("Executing a sequence of operations", func() {
		Context("A typical sequence", func() {
			It("Should do what expected", func() {
				By("Getting a registered environment")
				env, ok := manager.GetEnvironment(schemaID1)
				Expect(ok).To(BeTrue())
				Expect(env).NotTo(BeNil())

				By("Successfully unregistering it")
				Expect(manager.UnRegisterEnvironment(schemaID1)).To(Succeed())

				By("No longer returning it")
				_, ok = manager.GetEnvironment(schemaID1)
				Expect(ok).To(BeFalse())

				By("Registering another environment")
				Expect(manager.RegisterEnvironment(schemaID2, env2)).To(Succeed())

				By("Getting it when requested")
				_, ok = manager.GetEnvironment(schemaID2)
				Expect(ok).To(BeTrue())

				By("Clearing the whole manager")
				extension.ClearManager()
				manager = extension.GetManager()

				By("No longer returning the environment")
				_, ok = manager.GetEnvironment(schemaID2)
				Expect(ok).To(BeFalse())
			})
		})
	})
})
