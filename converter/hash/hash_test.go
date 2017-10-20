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

var _ = Describe("hash tests", func() {
	Describe("mod add tests", func() {
		It("Should add numbers with sum less than mod", func() {
			Expect(AddMod(1, 2, 4)).To(Equal(uint32(3)))
		})

		It("Should add numbers with sum greater than mod", func() {
			Expect(AddMod(2, 3, 4)).To(Equal(uint32(1)))
		})

		It("Should add numbers with overflow", func() {
			Expect(AddMod(0xFFFFFFFE, 2, 0xFFFFFFFF)).To(Equal(uint32(1)))
		})
	})

	Describe("mod mul tests", func() {
		It("Should multiply numbers with product less than mod", func() {
			Expect(MulMod(2, 3, 7)).To(Equal(uint32(6)))
		})

		It("Should multiply numbers with product greater than mod", func() {
			Expect(MulMod(10, 20, 3)).To(Equal(uint32(2)))
		})

		It("Should multiply numbers with overflow", func() {
			Expect(MulMod(0xFFFFFFFE, 5, 0xFFFFFFFF)).To(Equal(uint32(0xFFFFFFFA)))
		})
	})

	Describe("calc hash tests", func() {
		It("Should calc correct empty hash", func() {
			hash := Hash{}

			result := hash.Calc("")

			Expect(result.length).To(Equal(0))
			Expect(result.value).To(Equal(uint32(0)))
		})

		It("Should calc correct hash", func() {
			hash := Hash{}

			result := hash.Calc("a")

			Expect(result.length).To(Equal(1))
			Expect(result.value).To(Equal(uint32('a')))
		})
	})

	Describe("join tests", func() {
		It("Should join hash with zero value", func() {
			hash := Hash{}
			first := Node{1, 5}
			second := Node{0, 4}

			result := hash.Join(first, second)

			Expect(result.length).To(Equal(9))
			Expect(result.value).To(Equal(uint32(1)))
		})

		It("Should join hashes", func() {
			hash := Hash{}
			first := Node{1, 2}
			second := Node{3, 4}

			result := hash.Join(first, second)

			Expect(result.length).To(Equal(6))
			Expect(result.value).To(Equal(uint32(198148)))
		})
	})

	Describe("general tests", func() {
		It("Should compare equal strings", func() {
			hash := Hash{}
			strings := []string{"a", "bc", "ab", "c"}
			node := make([]Node, len(strings))

			for i, string := range strings {
				node[i] = hash.Calc(string)
			}

			first := hash.Join(node[0], node[1])
			second := hash.Join(node[2], node[3])

			Expect(first.value).To(Equal(second.value))
		})

		It("Should compare equal strings", func() {
			hash := Hash{}
			strings := []string{"abc", "def", "abc", "a", "bcdefabc"}
			node := make([]Node, len(strings))

			for i, string := range strings {
				node[i] = hash.Calc(string)
			}

			first := hash.Join(hash.Join(node[0], node[1]), node[2])
			second := hash.Join(node[3], node[4])

			Expect(first.value).To(Equal(second.value))
		})

		It("Should compare non equal strings", func() {
			hash := Hash{}
			strings := []string{"aaaaaaaaaaa", "aaaaaaaaaa"}

			first := hash.Calc(strings[0])
			second := hash.Calc(strings[1])

			Expect(first.value).ToNot(Equal(second.value))
		})

		It("Should compare non equal strings", func() {
			hash := Hash{}
			strings := []string{"abc", "def", "abc", "aa", "bcdefabc"}
			node := make([]Node, len(strings))

			for i, string := range strings {
				node[i] = hash.Calc(string)
			}

			first := hash.Join(hash.Join(node[0], node[1]), node[2])
			second := hash.Join(node[3], node[4])

			Expect(first.value).ToNot(Equal(second.value))
		})
	})

	Describe("sorting tests", func() {
		Describe("len tests", func() {
			It("Should return a correct length", func() {
				array := byHash{&treeNode{}, &treeNode{}}

				Expect(array.Len()).To(Equal(len(array)))
			})
		})

		Describe("swap tests", func() {
			It("Should swap items on an array", func() {
				first := &treeNode{index: 0}
				second := &treeNode{index: 1}
				array := byHash{first, second}

				array.Swap(0, 1)

				Expect(array[0]).To(BeIdenticalTo(second))
				Expect(array[1]).To(BeIdenticalTo(first))
			})
		})

		Describe("less tests", func() {
			var (
				first  *treeNode
				second *treeNode
				array  byHash
			)

			BeforeEach(func() {
				first = &treeNode{}
				second = &treeNode{}
				array = byHash{first, second}
			})

			It("Should be less when ancestor of the first is nil", func() {
				first.ancestor = nil

				Expect(array.Less(0, 1)).To(BeTrue())
			})

			It("Should be greater when ancestor of the second is nil", func() {
				first.ancestor = first
				second.ancestor = nil

				Expect(array.Less(0, 1)).To(BeFalse())
			})

			It("Should compare by index of an ancestors if hashes are equal", func() {
				first.ancestor = &treeNode{index: 1}
				second.ancestor = &treeNode{index: 0}
				first.value = Node{}
				second.value = Node{}

				Expect(array.Less(0, 1)).To(BeFalse())
			})

			It("Should compare by length if values are equal", func() {
				first.ancestor = first
				second.ancestor = first
				first.value = Node{length: 1}
				second.value = Node{length: 0}

				Expect(array.Less(0, 1)).To(BeFalse())
			})

			It("Should compare by value", func() {
				first.ancestor = first
				second.ancestor = first
				first.value = Node{value: 0, length: 1}
				second.value = Node{value: 1, length: 0}

				Expect(array.Less(0, 1)).To(BeTrue())
			})
		})
	})
})
