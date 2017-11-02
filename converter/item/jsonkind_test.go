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

var _ = Describe("json kind tests", func() {
	var jsonKind = &JSONKind{}

	Describe("type tests", func() {
		It("Should return a correct type for a null item", func() {
			typeOfItem := "string"
			list := []interface{}{typeOfItem, "null"}
			newItem, err := CreateItem(list)
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Required: true,
				Data:     map[interface{}]interface{}{"type": list},
			})
			Expect(err).ToNot(HaveOccurred())

			result := jsonKind.Type("", newItem)

			expected := "goext.MaybeString"
			Expect(result).To(Equal(expected))
		})

		It("Should return a correct type for a not null item", func() {
			typeOfItem := "string"
			newItem, err := CreateItem(typeOfItem)
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Required: true,
				Data:     map[interface{}]interface{}{"type": typeOfItem},
			})
			Expect(err).ToNot(HaveOccurred())

			result := jsonKind.Type("", newItem)

			expected := typeOfItem
			Expect(result).To(Equal(expected))
		})
	})

	Describe("interface type tests", func() {
		It("Should return a correct interface type for an object", func() {
			newItem, err := CreateItem("object")
			Expect(err).ToNot(HaveOccurred())

			name := "Test"
			err = newItem.Parse(ParseContext{
				Prefix:   name,
				Required: false,
				Data: map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						"a": map[interface{}]interface{}{
							"type": "string",
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			result := jsonKind.InterfaceType("", newItem)

			expected := "I" + name
			Expect(result).To(Equal(expected))
		})
	})

	Describe("annotation tests", func() {
		It("Should return a correct annotation for a null item", func() {
			name := "name"
			typeOfItem := "string"
			list := []interface{}{typeOfItem, "null"}
			newItem, err := CreateItem(list)
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Required: true,
				Data:     map[interface{}]interface{}{"type": list},
			})
			Expect(err).ToNot(HaveOccurred())

			result := jsonKind.Annotation(name, newItem)

			expected := fmt.Sprintf(
				"`json:\"%s,omitempty\"`",
				name,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should return a correct annotation for a not null item", func() {
			name := "name"
			typeOfItem := "string"
			newItem, err := CreateItem(typeOfItem)
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Required: true,
				Data:     map[interface{}]interface{}{"type": typeOfItem},
			})
			Expect(err).ToNot(HaveOccurred())

			result := jsonKind.Annotation(name, newItem)

			expected := fmt.Sprintf(
				"`json:\"%s\"`",
				name,
			)
			Expect(result).To(Equal(expected))
		})
	})

	Describe("default value tests", func() {
		It("Should return a correct default value for a not null item", func() {
			typeOfItem := "int"
			newItem, err := CreateItem(typeOfItem)
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Defaults: 1,
				Required: true,
				Data:     map[interface{}]interface{}{"type": typeOfItem},
			})
			Expect(err).ToNot(HaveOccurred())

			result := jsonKind.Default("", newItem)

			expected := "1"
			Expect(result).To(Equal(expected))
		})
	})
})
