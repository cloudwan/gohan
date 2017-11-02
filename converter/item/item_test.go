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

package item

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("item tests", func() {
	Describe("item creation tests", func() {
		Describe("createItemFromString tests", func() {
			It("Should create an array", func() {
				typeName := "array"

				result := createItemFromString(typeName)

				expected := &Array{}
				Expect(result).To(Equal(expected))
			})

			It("Should create an object", func() {
				typeName := "object"

				result := createItemFromString(typeName)

				expected := &Object{}
				Expect(result).To(Equal(expected))
			})

			It("Should create a plain item", func() {
				typeName := "string"

				result := createItemFromString(typeName)

				expected := &PlainItem{}
				Expect(result).To(Equal(expected))
			})
		})

		Describe("CreateItem", func() {
			var typeOfItem interface{}

			It("Should return error for an invalid type", func() {
				typeOfItem = 1

				_, err := CreateItem(typeOfItem)

				expected := fmt.Errorf("unsupported type: %T", typeOfItem)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expected))
			})

			It("Should create item with a correct type", func() {
				typeOfItem = []interface{}{"null", 1, "object", "array"}

				result, err := CreateItem(typeOfItem)

				expected := &Object{}
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(expected))
			})
		})
	})
})
