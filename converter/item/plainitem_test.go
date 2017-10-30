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

	"github.com/cloudwan/gohan/converter/defaults"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("plain item tests", func() {
	Describe("hash tests", func() {
		Describe("to string tests", func() {
			It("Should return a correct item type", func() {
				typeOfItem := "int64"
				plainItem := PlainItem{itemType: typeOfItem}

				result := plainItem.ToString()

				expected := "#int64,true,"
				Expect(result).To(Equal(expected))
			})

			It("Should return a correct item type for a null item", func() {
				typeOfItem := "string"
				plainItem := PlainItem{itemType: typeOfItem, required: true}

				result := plainItem.ToString()

				expected := "#string,false,"
				Expect(result).To(Equal(expected))
			})

			It("Should return a correct item type for an item with default value", func() {
				typeOfItem := "string"
				plainItem := PlainItem{
					itemType:     typeOfItem,
					required:     true,
					defaultValue: defaults.CreatePlainDefaults("test"),
				}

				result := plainItem.ToString()

				expected := `#string,false,"test"`
				Expect(result).To(Equal(expected))
			})
		})

		Describe("compress tests", func() {
			It("Should do nothing", func() {
				plainItem := PlainItem{itemType: "test", null: true}
				original := plainItem

				plainItem.Compress(&PlainItem{}, &plainItem)

				Expect(plainItem).To(Equal(original))
			})
		})

		Describe("get children tests", func() {
			It("Should return an empty children list", func() {
				plainItem := PlainItem{}

				result := plainItem.GetChildren()

				Expect(result).To(BeNil())
			})
		})
	})

	Describe("copy tests", func() {
		It("Should copy a plain item", func() {
			plainItem := &PlainItem{}

			copy := plainItem.Copy()

			Expect(copy).ToNot(BeIdenticalTo(plainItem))
			Expect(copy).To(Equal(plainItem))
		})
	})

	Describe("make required tests", func() {
		It("Should make an item required", func() {
			plainItem := &PlainItem{}

			plainItem.MakeRequired()

			Expect(plainItem.required).To(BeTrue())
		})
	})

	Describe("contains object tests", func() {
		It("Should return false", func() {
			plainItem := &PlainItem{}

			Expect(plainItem.ContainsObject()).To(BeFalse())
		})
	})

	Describe("default value tests", func() {
		It("Should return a correct default value", func() {
			plainItem := PlainItem{defaultValue: defaults.CreatePlainDefaults("test")}

			Expect(plainItem.Default("")).To(Equal(`"test"`))
		})
	})

	Describe("type tests", func() {
		It("Should return a correct item type", func() {
			typeOfItem := "int64"
			plainItem := PlainItem{itemType: typeOfItem}

			result := plainItem.Type("")

			expected := typeOfItem
			Expect(result).To(Equal(expected))
		})
	})

	Describe("add properties tests", func() {
		It("Should return an error", func() {
			plainItem := &PlainItem{}

			err := plainItem.AddProperties(nil, false)

			expected := fmt.Errorf("cannot add properties to a plain item")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})
	})

	Describe("parse tests", func() {
		var (
			prefix    = "abc"
			plainItem *PlainItem
			data      map[interface{}]interface{}
		)

		BeforeEach(func() {
			plainItem = &PlainItem{}
		})

		It("Should return an error for an object with no type", func() {
			data = map[interface{}]interface{}{}

			err := plainItem.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"item %s does not have a type",
				prefix,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return an error for an unsupported type", func() {
			data = map[interface{}]interface{}{"type": 1}

			err := plainItem.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"item %s: unsupported type: %T",
				prefix,
				data["type"],
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should parse a valid object", func() {
			data = map[interface{}]interface{}{"type": "number"}

			err := plainItem.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := "float64"
			Expect(err).ToNot(HaveOccurred())
			Expect(plainItem.Type("")).To(Equal(expected))
		})

		It("Should not be null when default value is provided", func() {
			data = map[interface{}]interface{}{
				"type":    "string",
				"default": "abc",
			}

			err := plainItem.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: false,
				Data:     data,
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(plainItem.IsNull()).To(BeFalse())
		})

		It("Should be null when neither required nor default value is provided", func() {
			data = map[interface{}]interface{}{"type": "string"}

			err := plainItem.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: false,
				Data:     data,
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(plainItem.IsNull()).To(BeTrue())
		})

		It("Should parse a correct default value", func() {
			value := "name"
			data = map[interface{}]interface{}{"type": "string"}

			err := plainItem.Parse(ParseContext{
				Defaults: value,
				Data:     data,
			})

			expected := `"` + value + `"`
			Expect(err).ToNot(HaveOccurred())
			Expect(plainItem.Default("")).To(Equal(expected))
		})
	})

	Describe("collect objects tests", func() {
		It("Should return nil for a plain item", func() {
			plainItem := &PlainItem{}

			Expect(plainItem.CollectObjects(1, 0)).To(BeNil())
		})
	})

	Describe("collect objects tests", func() {
		It("Should return nil for a plain item", func() {
			plainItem := &PlainItem{}

			Expect(plainItem.CollectProperties(1, 0)).To(BeNil())
		})
	})

	Describe("generate getter tests", func() {
		const (
			name     = "string"
			variable = "var"
			argument = "arg"
		)

		var plainItem *PlainItem

		BeforeEach(func() {
			plainItem = &PlainItem{itemType: name, null: true}
		})

		It("Should return a correct getter for a plain item depth 1", func() {
			result := plainItem.GenerateGetter(variable, argument, "", 1)

			expected := fmt.Sprintf("\treturn %s", variable)
			Expect(result).To(Equal(expected))
		})

		It("Should return a correct getter for a plain item depth >1", func() {
			result := plainItem.GenerateGetter(variable, argument, "", 2)

			expected := fmt.Sprintf("\t\t%s = %s", argument, variable)
			Expect(result).To(Equal(expected))
		})
	})

	Describe("generate setter tests", func() {
		It("Should return a correct setter for a plain item", func() {
			variable := "var"
			argument := "arg"

			plainItem := &PlainItem{}

			result := plainItem.GenerateSetter(variable, argument, "", 1)

			expected := fmt.Sprintf("\t%s = %s", variable, argument)
			Expect(result).To(Equal(expected))
		})
	})
})
