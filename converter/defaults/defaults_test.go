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

package defaults

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("defaults tests", func() {
	Describe("defaults creation tests", func() {
		It("Should create nil defaults for a complex type", func() {
			Expect(CreatePlainDefaults([]string{})).To(BeNil())
		})

		It("Should create string defaults", func() {
			Expect(CreatePlainDefaults("test")).To(Equal(&StringDefault{value: "test"}))
		})

		It("Should create int defaults", func() {
			Expect(CreatePlainDefaults(1)).To(Equal(&IntDefault{value: 1}))
		})

		It("Should create float defaults", func() {
			Expect(CreatePlainDefaults(1.5)).To(Equal(&FloatDefault{value: 1.5}))
		})

		It("Should create bool defaults", func() {
			Expect(CreatePlainDefaults(true)).To(Equal(&BoolDefault{value: true}))
		})
	})
})
