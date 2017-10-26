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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type parsing tests", func() {
	Describe("TryToAddName tests", func() {
		It("Should return a empty string for an empty prefix", func() {
			Expect(TryToAddName("", "suffix")).To(BeEmpty())
		})

		It("Should add names", func() {
			Expect(TryToAddName("prefix", "suffix")).To(Equal("prefix_suffix"))
		})
	})

	Describe("AddName tests", func() {
		It("Should not add _ to an empty prefix", func() {
			name := "abc"

			result := AddName("", name)

			Expect(result).To(Equal(name))
		})

		It("Should add _ to a nonempty prefix", func() {
			prefix := "a"
			suffix := "b"

			result := AddName(prefix, suffix)

			expected := prefix + "_" + suffix
			Expect(result).To(Equal(expected))
		})
	})

	Describe("ToGoName tests", func() {
		It("Should return correct go name", func() {
			name := "aa-bb_cc-dd-eE"

			result := ToGoName(name, "")

			expected := "AaBbCcDdEE"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("mapType tests", func() {
		It("Should return the mapped type", func() {
			typeName := "number"

			result := mapType(typeName)

			expected := typeMapping[typeName]
			Expect(result).To(Equal(expected))
		})

		It("Should return given type for a type with no match", func() {
			typeName := "string"

			result := mapType(typeName)

			expected := typeName
			Expect(result).To(Equal(expected))
		})
	})

	Describe("ParseType tests", func() {
		var itemType interface{}

		It("Should return error for an unsupported argument type", func() {
			itemType = 1

			_, _, err := ParseType(itemType)

			expected := fmt.Errorf("unsupported type: %T", itemType)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for an array with no string", func() {
			itemType = []interface{}{1, false}

			_, _, err := ParseType(itemType)

			expected := fmt.Errorf("unsupported type: %T", itemType)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for an array with nulls", func() {
			itemType = []interface{}{"null", "null"}

			_, _, err := ParseType(itemType)

			expected := fmt.Errorf("unsupported type: %T", itemType)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return the mapped type for a string input", func() {
			itemType = "number"

			result, null, err := ParseType(itemType)

			expected := typeMapping[itemType.(string)]
			Expect(err).ToNot(HaveOccurred())
			Expect(null).To(BeFalse())
			Expect(result).To(Equal(expected))
		})

		It("Should return the mapped type for an array input", func() {
			itemType = []interface{}{"null", "number", "boolean"}

			result, null, err := ParseType(itemType)

			expected := typeMapping[itemType.([]interface{})[1].(string)]
			Expect(err).ToNot(HaveOccurred())
			Expect(null).To(BeTrue())
			Expect(result).To(Equal(expected))
		})
	})
})
