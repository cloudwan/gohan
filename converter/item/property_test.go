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
	"github.com/cloudwan/gohan/converter/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("property tests", func() {
	Describe("hash tests", func() {
		Describe("to string tests", func() {
			It("Should return a correct string representation of a property", func() {
				name := "name"
				property := CreateProperty(name)

				result := property.ToString()

				Expect(result).To(Equal(name))
			})
		})

		Describe("compress tests", func() {
			It("Should compress if destination is owned by the array", func() {
				source := &PlainItem{itemType: "1"}
				destination := &PlainItem{itemType: "2"}
				property := Property{item: destination}

				property.Compress(source, destination)

				Expect(source).ToNot(BeIdenticalTo(destination))
				Expect(property.item).To(BeIdenticalTo(source))
			})

			It("Should not compress if destination is not owned by the array", func() {
				source := &PlainItem{itemType: "1"}
				destination := &PlainItem{itemType: "2"}
				property := Property{item: destination}

				property.Compress(destination, source)

				Expect(source).ToNot(BeIdenticalTo(destination))
				Expect(property.item).To(BeIdenticalTo(destination))
			})
		})

		Describe("get children tests", func() {
			It("Should return a correct children set", func() {
				plainItem := &PlainItem{itemType: "test"}
				property := Property{item: plainItem}

				result := property.GetChildren()

				Expect(len(result)).To(Equal(1))
				Expect(result[0]).To(BeIdenticalTo(plainItem))
			})
		})
	})

	Describe("make required tests", func() {
		It("Should return false for a property with a non null item", func() {
			item := &Object{}
			property := &Property{item: item}

			result := property.MakeRequired()

			Expect(result).To(BeFalse())
			Expect(property.item).To(BeIdenticalTo(item))
		})

		It("Should return true for a property with a null item", func() {
			item := &PlainItem{}
			property := &Property{item: item}

			result := property.MakeRequired()

			Expect(result).To(BeTrue())
			Expect(property.item.IsNull()).To(BeFalse())
			Expect(property.item).ToNot(Equal(item))
		})

		It("Should copy a property item", func() {
			item := &PlainItem{null: true, required: true}
			property := &Property{item: item}

			result := property.MakeRequired()

			Expect(result).To(BeTrue())
			Expect(property.item.IsNull()).To(BeTrue())
			Expect(property.item).ToNot(BeIdenticalTo(item))
		})
	})

	Describe("creation tests", func() {
		It("Should create a property with a correct name", func() {
			name := "name"
			property := CreateProperty(name)

			Expect(property.name).To(Equal(name))
		})
	})

	Describe("is object tests", func() {
		It("Should return false for not an object", func() {
			property := CreateProperty("")
			property.item, _ = CreateItem("string")

			Expect(property.IsObject()).To(BeFalse())
		})

		It("Should return false for an object", func() {
			property := CreateProperty("")
			property.item, _ = CreateItem("object")

			Expect(property.IsObject()).To(BeTrue())
		})
	})

	Describe("add properties tests", func() {
		It("Should add properties for an item", func() {
			properties := set.New()
			properties.Insert(CreateProperty("a"))
			object := &Object{}
			property := &Property{
				name: "",
				item: object,
			}

			err := property.AddProperties(properties, true)

			Expect(err).ToNot(HaveOccurred())
			Expect(object.properties).To(Equal(properties))
		})
	})

	Describe("parse tests", func() {
		var (
			prefix   = "abc"
			property *Property
			data     map[interface{}]interface{}
		)

		BeforeEach(func() {
			property = &Property{name: "def"}
		})

		It("Should return an error for an object with no items", func() {
			data = map[interface{}]interface{}{}

			err := property.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"property %s does not have a type",
				util.AddName(prefix, property.name),
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return an error for a property with an invalid type", func() {
			data = map[interface{}]interface{}{
				"type": 1,
			}

			err := property.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"property %s: unsupported type: %T",
				util.AddName(prefix, property.name),
				data["type"],
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return an error for an invalid item", func() {
			data = map[interface{}]interface{}{
				"type": "array",
			}

			err := property.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			expected := fmt.Errorf(
				"array %s does not have items",
				util.AddName(prefix, property.name),
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should parse a valid property", func() {
			data = map[interface{}]interface{}{
				"type": "array",
				"items": map[interface{}]interface{}{
					"type": "string",
				},
			}

			err := property.Parse(ParseContext{
				Prefix:   prefix,
				Level:    0,
				Required: true,
				Data:     data,
			})

			typeOfItem := data["items"].(map[interface{}]interface{})["type"]
			expected := "[]" + typeOfItem.(string)
			Expect(err).ToNot(HaveOccurred())
			Expect(property.item.Type("")).To(Equal(expected))
		})

		Describe("default value tests", func() {
			var data map[interface{}]interface{}

			BeforeEach(func() {
				data = map[interface{}]interface{}{
					"type": "string",
				}
			})

			It("Should get a default value from context", func() {
				err := property.Parse(ParseContext{
					Defaults: "test",
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.hasDefault).To(BeTrue())
			})

			It("Should get a default value from data", func() {
				data["default"] = "test"
				err := property.Parse(ParseContext{
					Data: data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.hasDefault).To(BeTrue())
			})

			It("Should not get a nil default value", func() {
				data["default"] = nil
				err := property.Parse(ParseContext{
					Defaults: nil,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.hasDefault).To(BeFalse())
			})

			It("Should override a default value", func() {
				data["default"] = 2
				err := property.Parse(ParseContext{
					Defaults: 1,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.Default("")).To(Equal("2"))
			})
		})

		Describe("null item tests", func() {
			It("Should be null for non required, no default value, no null in type", func() {
				data = map[interface{}]interface{}{
					"type": "string",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: false,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeTrue())
			})

			It("Should not be null for required, no default value, no null in type", func() {
				data = map[interface{}]interface{}{
					"type": "string",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeFalse())
			})
			It("Should not be null for non required, default value, no null in type", func() {
				data = map[interface{}]interface{}{
					"type":    "string",
					"default": "test",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: false,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeFalse())
			})
			It("Should not be null for required, default value, no null in type", func() {
				data = map[interface{}]interface{}{
					"type":    "string",
					"default": "test",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeFalse())
			})
			It("Should be null for non required, no default value, null in type", func() {
				data = map[interface{}]interface{}{
					"type": []interface{}{
						"string",
						"null",
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: false,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeTrue())
			})
			It("Should be null for required, no default value, null in type", func() {
				data = map[interface{}]interface{}{
					"type": []interface{}{
						"string",
						"null",
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeTrue())
			})
			It("Should be null for non required, default value, null in type", func() {
				data = map[interface{}]interface{}{
					"type": []interface{}{
						"string",
						"null",
					},
					"default": "test",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: false,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeTrue())
			})
			It("Should be null for required, default value, null in type", func() {
				data = map[interface{}]interface{}{
					"type": []interface{}{
						"string",
						"null",
					},
					"default": "test",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(property.item.IsNull()).To(BeTrue())
			})
		})
	})

	Describe("collect objects tests", func() {
		It("Should collect object", func() {
			object := &Object{objectType: "abc"}
			property := &Property{
				name: "",
				item: object,
			}
			objects := set.New()
			objects.Insert(object)

			result, err := property.CollectObjects(1, 0)

			expected := objects
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Describe("generate constructor tests", func() {
		It("Should return empty string for a property with no default value", func() {
			property := &Property{}

			Expect(property.GenerateConstructor("")).To(BeEmpty())
		})

		It("Should generate a correct constructor for a property", func() {
			property := &Property{name: "test"}
			data := map[interface{}]interface{}{"type": "string"}

			err := property.Parse(ParseContext{
				Defaults: "",
				Data:     data,
			})
			Expect(err).ToNot(HaveOccurred())

			result := property.GenerateConstructor("")

			expected := "Test: goext.MakeString(\"\")"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("generate property tests", func() {
		var (
			prefix   = "abc"
			suffix   = "xyz"
			property *Property
			data     map[interface{}]interface{}
		)

		Describe("db property tests", func() {
			const dbAnnotation = "db"
			const jsonAnnotation = "json"

			BeforeEach(func() {
				property = &Property{
					name: "def_id",
				}
			})

			It("Should generate a correct property for a null item", func() {
				data = map[interface{}]interface{}{
					"type": []interface{}{
						"string",
						"null",
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID goext.MaybeString `%s:\"%s\" %s:\"%s\"`\n",
					dbAnnotation,
					property.name,
					jsonAnnotation,
					property.name+",omitempty",
				)
				Expect(result).To(Equal(expected))
			})

			It("Should generate a correct property for a plain item", func() {
				data = map[interface{}]interface{}{
					"type": "boolean",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID bool `%s:\"%s\" %s:\"%s\"`\n",
					dbAnnotation,
					property.name,
					jsonAnnotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})

			It("Should generate a correct property for an array", func() {
				data = map[interface{}]interface{}{
					"type": "array",
					"items": map[interface{}]interface{}{
						"type": "integer",
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID []int `%s:\"%s\" %s:\"%s\"`\n",
					dbAnnotation,
					property.name,
					jsonAnnotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})

			It("Should generate a correct property for an object", func() {
				data = map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						"test": map[interface{}]interface{}{
							"type": "string",
						},
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    0,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID *AbcDefIDXyz `%s:\"%s\" %s:\"%s\"`\n",
					dbAnnotation,
					property.name,
					jsonAnnotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})
		})

		Describe("json property tests", func() {
			const annotation = "json"

			BeforeEach(func() {
				property = &Property{
					name: "def_id",
				}
			})

			It("Should generate a correct property for a null item", func() {
				data = map[interface{}]interface{}{
					"type": []interface{}{
						"string",
						"null",
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    2,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID goext.MaybeString `%s:\"%s,omitempty\"`\n",
					annotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})

			It("Should generate a correct property for a plain item", func() {
				data = map[interface{}]interface{}{
					"type": "boolean",
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    2,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID bool `%s:\"%s\"`\n",
					annotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})

			It("Should generate a correct property for an array", func() {
				data = map[interface{}]interface{}{
					"type": "array",
					"items": map[interface{}]interface{}{
						"type": "integer",
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    2,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID []int `%s:\"%s\"`\n",
					annotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})

			It("Should generate a correct property for an object", func() {
				data = map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						"test": map[interface{}]interface{}{
							"type": "string",
						},
					},
				}

				err := property.Parse(ParseContext{
					Prefix:   prefix,
					Level:    2,
					Required: true,
					Data:     data,
				})
				Expect(err).ToNot(HaveOccurred())

				result := property.GenerateProperty(suffix)

				expected := fmt.Sprintf(
					"\tDefID *AbcDefIDXyz `%s:\"%s\"`\n",
					annotation,
					property.name,
				)
				Expect(result).To(Equal(expected))
			})
		})
	})

	Describe("getter header tests", func() {
		It("Should generate a correct getter header for a plain item", func() {
			property := &Property{
				name: "name",
				item: &PlainItem{itemType: "string", null: true},
				kind: &DBKind{},
			}

			result := property.GetterHeader("suffix")

			expected := "GetName() goext.MaybeString"
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct getter header for an object", func() {
			properties := set.New()
			properties.Insert(&Property{})
			property := &Property{
				name: "name",
				item: &Object{objectType: "test", properties: properties},
				kind: &DBKind{},
			}

			result := property.GetterHeader("suffix")

			expected := "GetName() ITestSuffix"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("setter header tests", func() {
		It("Should generate a correct setter header for a plain item", func() {
			property := &Property{
				name: "name",
				item: &PlainItem{itemType: "string", null: true},
				kind: &DBKind{},
			}

			result := property.SetterHeader("suffix", true)

			expected := "SetName(name goext.MaybeString)"
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter header for an object", func() {
			properties := set.New()
			properties.Insert(&Property{})
			property := &Property{
				name: "name",
				item: &Object{objectType: "test", properties: properties},
				kind: &DBKind{},
			}

			result := property.SetterHeader("suffix", false)

			expected := "SetName(ITestSuffix)"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("generate getter tests", func() {
		It("Should generate a correct getter for a plain item", func() {
			property := &Property{
				name: "def",
				item: &PlainItem{itemType: "int64", null: true},
				kind: &DBKind{},
			}

			result := property.GenerateGetter("var", "")

			expected := `GetDef() goext.MaybeInt {
	return var.Def
}`
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct getter for an object", func() {
			properties := set.New()
			properties.Insert(&Property{})
			property := &Property{
				name: "abc",
				item: &Object{objectType: "xyz", properties: properties},
				kind: &DBKind{},
			}

			result := property.GenerateGetter("var", "")

			expected := `GetAbc() IXyz {
	return var.Abc
}`
			Expect(result).To(Equal(expected))
		})
	})

	Describe("generate setter tests", func() {
		It("Should generate a correct setter for a plain item", func() {
			property := &Property{
				name: "def",
				item: &PlainItem{itemType: "int64", null: true},
				kind: &DBKind{},
			}

			result := property.GenerateSetter("var", "", "")

			expected := `SetDef(def goext.MaybeInt) {
	var.Def = def
}`
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter for an object", func() {
			properties := set.New()
			properties.Insert(&Property{})
			property := &Property{
				name: "range",
				item: &Object{objectType: "xyz", properties: properties},
				kind: &DBKind{},
			}

			result := property.GenerateSetter("var", "", "")

			expected := `SetRange(rangeObject IXyz) {
	var.Range, _ = rangeObject.(*Xyz)
}`
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct setter for an array", func() {
			properties := set.New()
			properties.Insert(&Property{})
			property := &Property{
				name: "a",
				item: &Array{&Array{&Object{objectType: "object", properties: properties}}},
				kind: &DBKind{},
			}

			result := property.GenerateSetter("var", "", "")

			expected := `SetA(a [][]IObject) {
	var.A = make([][]*Object, len(a))
	for i := range a {
		var.A[i] = make([]*Object, len(a[i]))
		for j := range a[i] {
			var.A[i][j], _ = a[i][j].(*Object)
		}
	}
}`
			Expect(result).To(Equal(expected))
		})
	})

	Describe("compression tests", func() {
		It("Should compress exactly identical objects", func() {
			data := map[interface{}]interface{}{
				"type": "object",
				"properties": map[interface{}]interface{}{
					"a": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"x": map[interface{}]interface{}{
									"type": "string",
								},
								"y": map[interface{}]interface{}{
									"type": "number",
								},
							},
						},
					},
					"b": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"x": map[interface{}]interface{}{
									"type": "string",
								},
								"y": map[interface{}]interface{}{
									"type": "number",
								},
							},
						},
					},
					"c": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"x": map[interface{}]interface{}{
									"type": "string",
								},
								"y": map[interface{}]interface{}{
									"type": "number",
								},
							},
						},
					},
					"d": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"x": map[interface{}]interface{}{
									"type": "string",
								},
								"y": map[interface{}]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			}
			property := &Property{name: "test"}
			err := property.Parse(ParseContext{
				Prefix:   "test",
				Level:    0,
				Required: true,
				Data:     data,
			})
			Expect(err).ToNot(HaveOccurred())

			property.CompressObjects()

			properties := property.item.GetChildren()
			Expect(len(properties)).To(Equal(4))
			Expect(properties[0].(*Property).item).To(
				BeIdenticalTo(properties[1].(*Property).item),
			)
			Expect(properties[1].(*Property).item).To(
				BeIdenticalTo(properties[2].(*Property).item),
			)
			Expect(properties[2].(*Property).item).ToNot(
				BeIdenticalTo(properties[3].(*Property).item),
			)

			objects, err := property.CollectObjects(-1, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(objects.Size()).To(Equal(3))
		})

		It("Should not compress objects with different defaults", func() {
			data := map[interface{}]interface{}{
				"type": "object",
				"properties": map[interface{}]interface{}{
					"a": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type":    "string",
											"default": "test",
										},
									},
								},
								"b": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type":    "string",
											"default": "test2",
										},
									},
								},
							},
						},
					},
					"b": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type":    "string",
											"default": "test",
										},
									},
								},
								"b": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type":    "string",
											"default": "test2",
										},
									},
									"required": []interface{}{
										"a",
									},
								},
							},
						},
					},
				},
			}
			property := &Property{name: "test"}
			err := property.Parse(ParseContext{
				Prefix:   "test",
				Level:    0,
				Required: true,
				Data:     data,
			})
			Expect(err).ToNot(HaveOccurred())

			property.CompressObjects()

			objects, err := property.CollectObjects(-1, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(objects.Size()).To(Equal(4))

			array := objects.ToArray()
			Expect(array[0].Name()).To(Equal("test_test"))
			Expect(array[1].Name()).To(Equal("test_test_common"))
			Expect(array[2].Name()).To(Equal("test_test_common_a"))
			Expect(array[3].Name()).To(Equal("test_test_common_b"))
		})

		It("Should change names of compressed objects", func() {
			data := map[interface{}]interface{}{
				"type": "object",
				"properties": map[interface{}]interface{}{
					"a": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "string",
										},
									},
								},
								"b": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": []interface{}{
												"string",
												"null",
											},
										},
									},
									"required": []interface{}{
										"a",
									},
								},
							},
						},
					},
					"b": map[interface{}]interface{}{
						"type": "array",
						"items": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "string",
										},
									},
								},
								"b": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": []interface{}{
												"string",
												"null",
											},
										},
									},
									"required": []interface{}{
										"a",
									},
								},
							},
						},
					},
				},
			}
			property := &Property{name: "test"}
			err := property.Parse(ParseContext{
				Prefix:   "test",
				Level:    0,
				Required: true,
				Data:     data,
			})
			Expect(err).ToNot(HaveOccurred())

			property.CompressObjects()

			objects, err := property.CollectObjects(-1, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(objects.Size()).To(Equal(3))

			array := objects.ToArray()
			Expect(array[0].Name()).To(Equal("test_test"))
			Expect(array[1].Name()).To(Equal("test_test_common"))
			Expect(array[2].Name()).To(Equal("test_test_common_common"))
		})
	})
})
