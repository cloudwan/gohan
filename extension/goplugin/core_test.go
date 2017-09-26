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
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Core", func() {
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

		err := env.Load("test_data/ext_good/ext_good.so")
		Expect(err).To(BeNil())
		Expect(env.Start()).To(Succeed())
		testSchema := env.Schemas().Find("test")
		Expect(testSchema).To(Not(BeNil()))
	})

	AfterEach(func() {
		env.Stop()
	})

	Context("Triggering an event", func() {
		It("should panic when schema_id not specified", func() {
			ctx := map[string]interface{}{}
			Expect(func() { env.Core().TriggerEvent("dummyEventName", ctx) }).To(Panic())
		})

		It("should restore original schema when already specified", func() {
			ctx := map[string]interface{}{
				"schema":    "testSchema",
				"schema_id": "test",
			}

			Expect(env.Core().TriggerEvent("dummyEventName", ctx)).To(Succeed())
			Expect(ctx).To(HaveKeyWithValue("schema", "testSchema"))
		})

		It("should clean schema field when none specified", func() {
			ctx := map[string]interface{}{
				"schema_id": "test",
			}

			Expect(env.Core().TriggerEvent("dummyEventName", ctx)).To(Succeed())
			Expect(ctx).NotTo(HaveKey("schema"))
		})
	})
})
