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

package main

import (
	"github.com/cloudwan/gohan/examples/goext_example/todo"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

// Schemas returns a list of extra schemas that will be loaded with this test
func Schemas() []string {
	return []string{
		"../todo/entry.yaml",
		"../todo/link.yaml",
	}
}

// Binaries returns a path to the plugin's binary that this test depends on
func Binaries() []string {
	return []string{"../example.so"}
}

// Test is an entry point of this test
func Test(env goext.IEnvironment) {
	env.Logger().Notice("Running example test suite")

	ginkgo.Describe("Example tests", func() {
		var (
			schema goext.ISchema
			entry  *todo.Entry
		)

		ginkgo.BeforeEach(func() {
			schema = env.Schemas().Find("entry")
			gomega.Expect(schema).To(gomega.Not(gomega.BeNil()))

			entry = &todo.Entry{
				ID:          env.Util().NewUUID(),
				Name:        "example name",
				Description: "example description",
				TenantID:    "admin",
			}
		})

		ginkgo.AfterEach(func() {
			env.Reset()
		})

		ginkgo.It("Smoke test CRUD", func() {
			gomega.Expect(schema.CreateRaw(entry, nil)).To(gomega.Succeed())
			gomega.Expect(schema.UpdateRaw(entry, nil)).To(gomega.Succeed())
			gomega.Expect(schema.DeleteRaw(goext.Filter{"id": entry.ID}, nil)).To(gomega.Succeed())
		})

		ginkgo.It("Should change name in pre_update event", func() {
			gomega.Expect(schema.CreateRaw(entry, nil)).To(gomega.Succeed())
			entry.Name = "random name"
			gomega.Expect(schema.UpdateRaw(entry, nil)).To(gomega.Succeed())
			gomega.Expect(entry.Name).To(gomega.Equal("name changed in pre_update event"))
		})
	})
}
