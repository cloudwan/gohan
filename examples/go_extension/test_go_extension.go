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
	"github.com/cloudwan/gohan/extension/goext"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Returns list of schemas that this test depends on
// @optional
func Schemas() []string {
	return []string{
		"todo.yaml",
		"todo_related.yaml",
	}
}

// Returns path to the plugin's binary that this test depends on
// @required
func Binary() string {
	return "go_extension.so"
}

// Defines the test suite
// @required
func Test(env *goext.Environment) {
	env.Logger.Notice("Running go_extension test suite")

	Describe("Tests extension", func() {
		It("Should react on events", func() {
			context := make(map[string]interface{})

			env.Core.HandleEvent(goext.POST_UPDATE, context) //this will run extension's event handler

			handled, ok := context["example_event_handled"]

			Expect(handled.(bool)).To(Equal(true))
			Expect(ok).To(Equal(true))
		})
	})
}
