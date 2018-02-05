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

package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("package util tests", func() {
	Describe("collect data tests", func() {
		It("Should generate an empty string for an empty data", func() {
			name := "name"
			data := []string{}

			result := CollectData(name, data)

			Expect(result).To(BeEmpty())
		})

		It("Should collect data", func() {
			name := "name"
			data := []string{"a", "", "b"}

			result := CollectData(name, data)

			expected := `package name

a
b`
			Expect(result).To(Equal(expected))
		})
	})

	Describe("const tests", func() {
		It("Should generate const", func() {
			data := []string{"a", "b"}

			result := Const(data)

			expected := `const (
	a
	b
)
`
			Expect(result).To(Equal(expected))
		})
	})
})
