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

package hash

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("hash util tests", func() {
	Describe("power tests", func() {
		It("Should get correct powers", func() {
			log, powers := powers(0)

			Expect(log).To(Equal(-1))
			Expect(powers).To(BeNil())
		})

		It("Should get correct powers", func() {
			log, powers := powers(1)

			Expect(log).To(Equal(0))
			Expect(powers).To(Equal([]int{0}))
		})

		It("Should get correct powers", func() {
			log, powers := powers(4)

			Expect(log).To(Equal(2))
			Expect(powers).To(Equal([]int{2}))
		})

		It("Should get correct powers", func() {
			log, powers := powers(7)

			Expect(log).To(Equal(2))
			Expect(powers).To(Equal([]int{0, 1, 2}))
		})

		It("Should get correct powers", func() {
			log, powers := powers(10)

			Expect(log).To(Equal(3))
			Expect(powers).To(Equal([]int{1, 3}))
		})
	})
})
