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

	"github.com/cloudwan/gohan/converter/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("db kind tests", func() {
	var dbKind = &DBKind{}

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

			result := dbKind.Type("", newItem)

			expected := "goext." + util.ToGoName("maybe", typeOfItem)
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

			result := dbKind.Type("", newItem)

			expected := typeOfItem
			Expect(result).To(Equal(expected))
		})
	})

	Describe("interface type tests", func() {
		It("Should return a correct interface type for a null item", func() {
			newItem, err := CreateItem("int64")
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Required: false,
				Data:     map[interface{}]interface{}{"type": "int64"},
			})
			Expect(err).ToNot(HaveOccurred())

			result := dbKind.InterfaceType("", newItem)

			expected := "goext.MaybeInt"
			Expect(result).To(Equal(expected))
		})

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

			result := dbKind.InterfaceType("", newItem)

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

			result := dbKind.Annotation(name, newItem)

			expected := fmt.Sprintf(
				"`db:\"%s\" json:\"%s,omitempty\"`",
				name, name,
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

			result := dbKind.Annotation(name, newItem)

			expected := fmt.Sprintf(
				"`db:\"%s\" json:\"%s\"`",
				name, name,
			)
			Expect(result).To(Equal(expected))
		})
	})

	Describe("default value tests", func() {
		It("Should return a correct default value for a null item", func() {
			value := 1
			typeOfItem := "int"
			newItem, err := CreateItem(typeOfItem)
			Expect(err).ToNot(HaveOccurred())

			err = newItem.Parse(ParseContext{
				Defaults: value,
				Data:     map[interface{}]interface{}{"type": typeOfItem},
			})
			Expect(err).ToNot(HaveOccurred())

			result := dbKind.Default("", newItem)

			expected := fmt.Sprintf(
				"goext.MakeInt(%d)",
				value,
			)
			Expect(result).To(Equal(expected))
		})

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

			result := dbKind.Default("", newItem)

			expected := "1"
			Expect(result).To(Equal(expected))
		})
	})
})
