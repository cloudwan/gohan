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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type test string

type test2 struct {
	a string
}

func (elem *test2) Name() string {
	return elem.a
}

func (test test) Name() string {
	return string(test)
}

var _ = Describe("set tests", func() {
	Describe("size tests", func() {
		It("Should return correct set size", func() {
			var set Set = map[string]Element{"a": test("a"), "b": test("b")}
			Expect(set.Size()).To(Equal(2))
		})

		It("Should return size for a nil set", func() {
			var set Set
			Expect(set.Size()).To(Equal(0))
		})
	})

	Describe("empty tests", func() {
		It("Should return true for an empty set", func() {
			var set Set = map[string]Element{}
			Expect(set.Empty()).To(BeTrue())
		})

		It("Should return false for a nonempty set", func() {
			var set Set = map[string]Element{"a": test("a")}
			Expect(set.Empty()).To(BeFalse())
		})

		It("Should return true for a nil set", func() {
			var set Set
			Expect(set.Empty()).To(BeTrue())
		})
	})

	Describe("any tests", func() {
		It("Should return nil for an empty set", func() {
			set := New()
			Expect(set.Any()).To(BeNil())
		})

		It("Should return first value of set with one value", func() {
			var elem test = "test"
			set := New()
			set.Insert(elem)
			result := set.Any()
			Expect(result).To(Equal(elem))
		})

		It("Should return any element of given set", func() {
			var set Set = map[string]Element{"a": test("a"), "b": test("b")}
			result := set.Any()
			Expect(set.Contains(result)).To(BeTrue())
		})
	})

	Describe("contains tests", func() {
		const elem test = "1"

		var set Set

		BeforeEach(func() {
			set = New()
			set[elem.Name()] = elem
		})

		It("Should return true for an existing element", func() {
			Expect(set.Contains(elem)).To(BeTrue())
		})

		It("Should return false for a non existing element", func() {
			Expect(set.Contains(test("abc"))).To(BeFalse())
		})

		It("Should return false for a nil set", func() {
			var set Set
			Expect(set.Contains(elem)).To(BeFalse())
		})
	})

	Describe("delete tests", func() {
		const elem test = "1"

		var set Set

		BeforeEach(func() {
			set = New()
			set[elem.Name()] = elem
		})

		It("Should delete elements in a set", func() {
			Expect(set.Contains(elem)).To(BeTrue())
			set.Delete(elem)
			Expect(set.Size()).To(Equal(0))
			Expect(set.Contains(elem)).To(BeFalse())
		})

		It("Should do nothing with non existing elements", func() {
			set.Delete(test("a"))
			Expect(set.Contains(elem)).To(BeTrue())
			Expect(set.Size()).To(Equal(1))
		})

		It("Should do nothing for a nil set", func() {
			var set Set
			set.Delete(test("a"))
			Expect(set).To(BeNil())
		})
	})

	Describe("insert tests", func() {
		var set Set

		BeforeEach(func() {
			set = New()
		})

		It("Should insert elements into a set", func() {
			var (
				a test = "a"
				b test = "b"
			)
			Expect(set.Empty()).To(BeTrue())
			set.Insert(a)
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
			set.Insert(b)
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(a)).To(BeTrue())
			Expect(set.Contains(b)).To(BeTrue())
		})

		It("Should override value of items with same name", func() {
			var (
				a test = "a"
				b test = "a"
			)
			Expect(set.Empty()).To(BeTrue())
			set.Insert(a)
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
			set.Insert(b)
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
		})

		It("Should do nothing for a nil set", func() {
			var (
				set Set
				a   test = "a"
			)
			set.Insert(a)
			Expect(set).To(BeNil())
		})
	})

	Describe("insert all tests", func() {
		var (
			set   Set
			other Set
		)

		BeforeEach(func() {
			set = New()
			other = New()
		})

		It("Should insert all elements", func() {
			var (
				a test = "a"
				b test = "b"
			)
			other.Insert(a)
			other.Insert(b)
			Expect(set.Empty()).To(BeTrue())
			Expect(other.Size()).To(Equal(2))
			Expect(other.Contains(a)).To(BeTrue())
			Expect(other.Contains(b)).To(BeTrue())
			set.InsertAll(other)
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(a)).To(BeTrue())
			Expect(set.Contains(b)).To(BeTrue())
		})

		It("Should override intersecting elements", func() {
			var (
				a test = "a"
				b test = "b"
				c test = "c"
			)
			set.Insert(a)
			set.Insert(b)
			other.Insert(b)
			other.Insert(c)
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(c)).To(BeFalse())
			Expect(other.Size()).To(Equal(2))
			Expect(other.Contains(a)).To(BeFalse())
			set.InsertAll(other)
			Expect(set.Size()).To(Equal(3))
			Expect(set.Contains(c)).To(BeTrue())
		})
	})

	Describe("safe insert tests", func() {
		var set Set

		BeforeEach(func() {
			set = New()
		})

		It("Should insert elements into a set", func() {
			var (
				a test = "a"
				b test = "b"
			)
			Expect(set.Empty()).To(BeTrue())
			err := set.SafeInsert(a)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
			err = set.SafeInsert(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(a)).To(BeTrue())
			Expect(set.Contains(b)).To(BeTrue())
		})

		It("Should return error for items with same name", func() {
			var (
				a test = "a"
				b      = &test2{"a"}
			)
			Expect(set.Empty()).To(BeTrue())
			err := set.SafeInsert(a)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
			expected := fmt.Errorf("the element with the name a already in the set")
			err = set.SafeInsert(b)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
		})

		It("Should do nothing when insearting item that already was in the set", func() {
			var (
				a test = "a"
				b test = "a"
			)
			Expect(set.Empty()).To(BeTrue())
			err := set.SafeInsert(a)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
			err = set.SafeInsert(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(1))
			Expect(set.Contains(a)).To(BeTrue())
		})
	})

	Describe("safe insert all tests", func() {
		var (
			set   Set
			other Set
		)

		BeforeEach(func() {
			set = New()
			other = New()
		})

		It("Should insert all elements", func() {
			var (
				a test = "a"
				b test = "b"
			)
			other.Insert(a)
			other.Insert(b)
			Expect(set.Empty()).To(BeTrue())
			Expect(other.Size()).To(Equal(2))
			Expect(other.Contains(a)).To(BeTrue())
			Expect(other.Contains(b)).To(BeTrue())
			err := set.SafeInsertAll(other)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(a)).To(BeTrue())
			Expect(set.Contains(b)).To(BeTrue())
		})

		It("Should return error for items with the same name", func() {
			var (
				a test = "a"
				b test = "b"
				c test = "c"
				d      = &test2{"b"}
			)
			set.Insert(a)
			set.Insert(b)
			other.Insert(d)
			other.Insert(c)
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(c)).To(BeFalse())
			Expect(other.Size()).To(Equal(2))
			Expect(other.Contains(a)).To(BeFalse())
			expected := fmt.Errorf("the element with the name b already in the set")
			err := set.SafeInsertAll(other)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(c)).To(BeFalse())
		})

		It("Should ignore the same items", func() {
			var (
				a test = "a"
				b test = "b"
				c test = "c"
			)
			set.Insert(a)
			set.Insert(b)
			other.Insert(b)
			other.Insert(c)
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains(c)).To(BeFalse())
			Expect(other.Size()).To(Equal(2))
			Expect(other.Contains(a)).To(BeFalse())
			err := set.SafeInsertAll(other)
			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(3))
			Expect(set.Contains(c)).To(BeTrue())
		})
	})

	Describe("to array tests", func() {
		It("Should return sorted array", func() {
			set := New()
			set.Insert(test("b"))
			set.Insert(test("c"))
			set.Insert(test("a"))
			expected := []Element{test("a"), test("b"), test("c")}
			result := set.ToArray()
			Expect(result).To(Equal(expected))
		})

		It("Should return nil for a nil set", func() {
			var set Set
			Expect(set.ToArray()).To(BeNil())
		})
	})
})
