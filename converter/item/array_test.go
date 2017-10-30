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

	"github.com/cloudwan/gohan/converter/set"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("array tests", func() {
	Describe("hash tests", func() {
		Describe("to string tests", func() {
			It("Should return a correct string representation of an array", func() {
				array := Array{&PlainItem{itemType: "test"}}

				result := array.ToString()

				expected := "#[]"
				Expect(result).To(Equal(expected))
			})
		})

		Describe("compress tests", func() {
			It("Should compress if destination is owned by the array", func() {
				source := &PlainItem{itemType: "1"}
				destination := &PlainItem{itemType: "2"}
				array := Array{destination}

				array.Compress(source, destination)

				Expect(source).ToNot(BeIdenticalTo(destination))
				Expect(array.arrayItem).To(BeIdenticalTo(source))
			})

			It("Should not compress if destination is not owned by the array", func() {
				source := &PlainItem{itemType: "1"}
				destination := &PlainItem{itemType: "2"}
				array := Array{destination}

				array.Compress(destination, source)

				Expect(source).ToNot(BeIdenticalTo(destination))
				Expect(array.arrayItem).To(BeIdenticalTo(destination))
			})
		})

		Describe("get children tests", func() {
			It("Should return a correct children set", func() {
				plainItem := &PlainItem{itemType: "test"}
				array := Array{plainItem}

				result := array.GetChildren()

				Expect(len(result)).To(Equal(1))
				Expect(result[0]).To(BeIdenticalTo(plainItem))
			})
		})
	})

	Describe("copy tests", func() {
		It("Should copy an array", func() {
			array := &Array{&Array{&Object{}}}

			copy := array.Copy()

			Expect(copy).ToNot(BeIdenticalTo(array))
			Expect(copy).To(Equal(array))
		})
	})

	Describe("make required tests", func() {
		It("Should do nothing", func() {
			array := &Array{&Array{&Object{}}}
			old := array

			array.MakeRequired()

			Expect(array).To(Equal(old))
		})
	})

	Describe("contains object tests", func() {
		It("Should return true for an array of objects", func() {
			properties := set.New()
			properties.Insert(&Property{})
			array := Array{&Object{properties: properties}}

			Expect(array.ContainsObject()).To(BeTrue())
		})

		It("Should return false for an array of plain items", func() {
			array := Array{&PlainItem{}}

			Expect(array.ContainsObject()).To(BeFalse())
		})
	})

	Describe("default value tests", func() {
		It("Should return a correct default value", func() {
			typeOfItem := "string"
			array := Array{&PlainItem{itemType: typeOfItem}}

			result := array.Default("")

			expected := "[]" + typeOfItem + "{}"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("type tests", func() {
		It("Should return a correct array type", func() {
			typeOfItem := "int64"
			array := Array{&PlainItem{itemType: typeOfItem}}

			result := array.Type("")

			expected := "[]" + typeOfItem
			Expect(result).To(Equal(expected))
		})

		It("Should return correct array type for nested arrays", func() {
			typeOfItem := "string"
			nested := Array{&PlainItem{itemType: typeOfItem}}
			array := Array{&nested}

			result := array.Type("")

			expected := "[][]" + typeOfItem
			Expect(result).To(Equal(expected))
		})
	})

	Describe("add properties tests", func() {
		It("Should return an error", func() {
			array := &Array{}

			err := array.AddProperties(nil, false)

			expected := fmt.Errorf("cannot add properties to an array")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})
	})

	Describe("parse tests", func() {
		var (
			prefix = "abc"
			array  *Array
			data   map[interface{}]interface{}
		)

		BeforeEach(func() {
			array = &Array{}
		})

		It("Should return an error for an object with no items", func() {
			data = map[interface{}]interface{}{}

			err := array.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"array %s does not have items",
				prefix,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return an error for an item with no type", func() {
			data = map[interface{}]interface{}{
				"items": map[interface{}]interface{}{
					"a": 1,
				},
			}

			err := array.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"items of array %s do not have a type",
				prefix,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return an error for an invalid item", func() {
			data = map[interface{}]interface{}{
				"items": map[interface{}]interface{}{
					"type": 1,
				},
			}

			err := array.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"array %s: unsupported type: %T",
				prefix,
				data["items"].(map[interface{}]interface{})["type"],
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should parse a valid array", func() {
			data = map[interface{}]interface{}{
				"items": map[interface{}]interface{}{
					"type": "string",
				},
			}

			err := array.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			typeOfItem := data["items"].(map[interface{}]interface{})["type"]
			expected := "[]" + typeOfItem.(string)
			Expect(err).ToNot(HaveOccurred())
			Expect(array.Type("")).To(Equal(expected))
		})
	})

	Describe("collect object tests", func() {
		It("Should return nil for an array of plain items", func() {
			array := &Array{&PlainItem{}}

			Expect(array.CollectObjects(1, 0)).To(BeNil())
		})

		It("Should return object for an array of objects", func() {
			name := "Test"
			array := &Array{&Object{objectType: name}}

			result, err := array.CollectObjects(1, 0)

			Expect(err).ToNot(HaveOccurred())
			objects := result.ToArray()
			Expect(len(objects)).To(Equal(1))
			Expect(objects[0].(*Object).Type("")).To(Equal("interface{}"))
		})
	})

	Describe("collect properties tests", func() {
		It("Should return nil for an array of plain items", func() {
			array := &Array{&PlainItem{}}

			Expect(array.CollectProperties(1, 0)).To(BeNil())
		})
	})

	Describe("generate getter tests", func() {
		const (
			variable = "variable"
			argument = "argument"
		)

		It("Should generate a correct getter for an array of plain items", func() {
			name := "string"
			array := &Array{&PlainItem{itemType: name}}

			result := array.GenerateGetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				"\treturn %s",
				variable,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct getter for an array of empty objects", func() {
			name := "Type"
			array := &Array{&Object{objectType: name}}

			result := array.GenerateGetter(variable, argument, "", 1)
			expected := fmt.Sprintf(
				"\treturn %s",
				variable,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct getter for an array of objects", func() {
			name := "Type"
			properties := set.New()
			properties.Insert(&Property{})
			array := &Array{&Object{objectType: name, properties: properties}}

			result := array.GenerateGetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				`	%s := make([]I%s, len(%s))
	for i := range %s {
		%s[i] = %s[i]
	}
	return %s`,
				argument,
				name,
				variable,
				variable,
				argument,
				variable,
				argument,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct getter for nested array of plain items", func() {
			name := "string"
			array := &Array{&Array{&PlainItem{itemType: name}}}

			result := array.GenerateGetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				"\treturn %s",
				variable,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct getter for nested array of objects", func() {
			name := "Type"
			properties := set.New()
			properties.Insert(&Property{})
			array := &Array{&Array{&Object{objectType: name, properties: properties}}}

			result := array.GenerateGetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				`	%s := make([][]I%s, len(%s))
	for i := range %s {
		%s[i] = make([]I%s, len(%s[i]))
		for j := range %s[i] {
			%s[i][j] = %s[i][j]
		}
	}
	return %s`,
				argument,
				name,
				variable,
				variable,
				argument,
				name,
				variable,
				variable,
				argument,
				variable,
				argument,
			)
			Expect(result).To(Equal(expected))
		})
	})

	Describe("generate setter tests", func() {
		const (
			variable = "variable"
			argument = "argument"
		)

		It("Should generate a correct setter for an array of plain items", func() {
			name := "string"
			array := &Array{&PlainItem{itemType: name}}

			result := array.GenerateSetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				"\t%s = %s",
				variable,
				argument,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter for an array of objects", func() {
			name := "Type"
			properties := set.New()
			properties.Insert(&Property{})
			array := &Array{&Object{objectType: name, properties: properties}}

			result := array.GenerateSetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				`	%s = make([]*%s, len(%s))
	for i := range %s {
		%s[i], _ = %s[i].(*%s)
	}`,
				variable,
				name,
				argument,
				argument,
				variable,
				argument,
				name,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter for nested array of plain items", func() {
			name := "string"
			array := &Array{&Array{&PlainItem{itemType: name}}}

			result := array.GenerateSetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				"\t%s = %s",
				variable,
				argument,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter for array of empty objects", func() {
			name := "Type"
			array := &Array{&Array{&Object{objectType: name}}}

			result := array.GenerateSetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				"\t%s = %s",
				variable,
				argument,
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter for nested array of objects", func() {
			name := "Type"
			properties := set.New()
			properties.Insert(&Property{})
			array := &Array{&Array{&Object{objectType: name, properties: properties}}}

			result := array.GenerateSetter(variable, argument, "", 1)

			expected := fmt.Sprintf(
				`	%s = make([][]*%s, len(%s))
	for i := range %s {
		%s[i] = make([]*%s, len(%s[i]))
		for j := range %s[i] {
			%s[i][j], _ = %s[i][j].(*%s)
		}
	}`,
				variable,
				name,
				argument,
				argument,
				variable,
				name,
				argument,
				argument,
				variable,
				argument,
				name,
			)
			Expect(result).To(Equal(expected))
		})
	})
})
