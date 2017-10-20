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

package set

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("element tests", func() {
	Describe("len tests", func() {
		It("Should return correct length", func() {
			var array byName = []Element{test("a"), test("b")}
			Expect(array.Len()).To(Equal(len(array)))
		})
	})

	Describe("swap tests", func() {
		It("Should swap elements for array", func() {
			var array byName = []Element{test("a"), test("b")}
			array.Swap(0, 1)
			Expect(array[0]).To(Equal(test("b")))
			Expect(array[1]).To(Equal(test("a")))
		})
	})

	Describe("less tests", func() {
		It("Should compare elements", func() {
			var array byName = []Element{test("a"), test("b")}
			Expect(array.Less(0, 1)).To(BeTrue())
		})
	})
})
