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

package schema

import (
	"fmt"

	"github.com/cloudwan/gohan/converter/item"
	"github.com/cloudwan/gohan/converter/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("schema tests", func() {
	Describe("get name tests", func() {
		var schema *Schema

		BeforeEach(func() {
			schema = &Schema{}
		})

		It("Should return error for schema with no name", func() {
			object := map[interface{}]interface{}{}

			err := schema.getName(object)

			expected := fmt.Errorf("schema does not have an id")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for schema with invalid name", func() {
			object := map[interface{}]interface{}{
				"id": 1,
			}

			err := schema.getName(object)

			expected := fmt.Errorf("schema does not have an id")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should get correct schema name", func() {
			name := "name"
			object := map[interface{}]interface{}{
				"id": name,
			}

			err := schema.getName(object)

			Expect(err).ToNot(HaveOccurred())
			Expect(schema.schema.Name()).To(Equal(name))
		})
	})

	Describe("get parent tests", func() {
		var schema *Schema

		BeforeEach(func() {
			schema = &Schema{}
		})

		It("Should get empty parent for schema with invalid parent", func() {
			object := map[interface{}]interface{}{
				"parent": 1,
			}

			schema.getParent(object)

			Expect(schema.parent).To(BeEmpty())
		})

		It("Should get correct parent", func() {
			name := "name"
			object := map[interface{}]interface{}{
				"parent": name,
			}

			schema.getParent(object)

			Expect(schema.parent).To(Equal(name))
		})
	})

	Describe("get base schemas tests", func() {
		var schema *Schema

		BeforeEach(func() {
			schema = &Schema{}
		})

		It("Should return error for schema with invalid base", func() {
			object := map[interface{}]interface{}{
				"extends": []interface{}{
					"a",
					1,
				},
			}

			err := schema.getBaseSchemas(object)

			expected := fmt.Errorf("one of the base schemas is not a string")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should get empty bases for schema with invalid base", func() {
			object := map[interface{}]interface{}{
				"extends": 1,
			}

			err := schema.getBaseSchemas(object)

			Expect(err).ToNot(HaveOccurred())
			Expect(schema.extends).To(BeNil())
		})

		It("Should get correct base schemas", func() {
			object := map[interface{}]interface{}{
				"extends": []interface{}{"a", "b", "c"},
			}

			err := schema.getBaseSchemas(object)

			Expect(err).To(BeNil())
			Expect(schema.extends).To(Equal([]string{"a", "b", "c"}))
		})
	})

	Describe("parse tests", func() {
		var schema *Schema

		BeforeEach(func() {
			schema = &Schema{}
		})

		It("Should return error for schema with no name", func() {
			object := map[interface{}]interface{}{}

			err := schema.parse(object)

			expected := fmt.Errorf("schema does not have an id")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for schema with invalid base schema", func() {
			name := "test"
			object := map[interface{}]interface{}{
				"id": name,
				"extends": []interface{}{
					1,
				},
			}

			err := schema.parse(object)

			expected := fmt.Errorf(
				"invalid schema %s: one of the base schemas is not a string",
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for schema with no schema", func() {
			name := "test"
			object := map[interface{}]interface{}{
				"id": name,
			}

			err := schema.parse(object)

			expected := fmt.Errorf(
				"invalid schema %s: schema does not have a \"schema\"",
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for invalid schema", func() {
			name := "test"
			object := map[interface{}]interface{}{
				"id": name,
				"schema": map[interface{}]interface{}{
					"properties": 1,
				},
			}

			err := schema.parse(object)

			expected := fmt.Errorf(
				"invalid schema %s: property %s does not have a type",
				name,
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for schema which is not an object", func() {
			name := "test"
			object := map[interface{}]interface{}{
				"id": name,
				"schema": map[interface{}]interface{}{
					"type": "string",
				},
			}

			err := schema.parse(object)

			expected := fmt.Errorf(
				"invalid schema %s: schema should be an object",
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for schema with invalid parent", func() {
			id := "test_schema"
			name := "test"
			object := map[interface{}]interface{}{
				"id":     id,
				"parent": name,
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						util.AddName(name, "id"): map[interface{}]interface{}{
							"type": "boolean",
						},
					},
				},
			}

			err := schema.parse(object)

			expected := fmt.Errorf(
				"invalid schema %s: object %s: multiple properties have the same name",
				id,
				id,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should create empty schema", func() {
			name := "test"
			object := map[interface{}]interface{}{
				"id":     name,
				"schema": map[interface{}]interface{}{},
			}

			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())
			Expect(schema.schema.IsObject()).To(BeTrue())

			result, err := schema.collectProperties(-1, 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Empty()).To(BeTrue())
		})

		It("Should parse correct schema", func() {
			name := "test"
			object := map[interface{}]interface{}{
				"id": name,
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						"a": map[interface{}]interface{}{
							"type": "string",
						},
					},
				},
			}

			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())
			Expect(schema.Name()).To(Equal(name))
			Expect(schema.schema.IsObject()).To(BeTrue())
		})
	})

	Describe("parse all tests", func() {
		It("Should return error for invalid schema", func() {
			name := "test"
			objects := []map[interface{}]interface{}{
				{
					"id": name + "1",
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"a": map[interface{}]interface{}{
								"type": "string",
							},
						},
					},
				},
				{
					"id": name,
					"schema": map[interface{}]interface{}{
						"type": "string",
					},
				},
			}

			_, err := parseAll(objects)

			expected := fmt.Errorf(
				"invalid schema %s: schema should be an object",
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for multiple schemas with the same name", func() {
			name := "test"
			objects := []map[interface{}]interface{}{
				{
					"id": name,
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"a": map[interface{}]interface{}{
								"type": "string",
							},
						},
					},
				},
				{
					"id": name,
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"b": map[interface{}]interface{}{
								"type": "number",
							},
						},
					},
				},
			}

			_, err := parseAll(objects)

			expected := fmt.Errorf(
				"multiple schemas with the same name: %s",
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should parse valid schemas", func() {
			name := "test"
			objects := []map[interface{}]interface{}{
				{
					"id": name + "0",
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"a": map[interface{}]interface{}{
								"type": "string",
							},
						},
					},
				},
				{
					"id": name + "1",
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"b": map[interface{}]interface{}{
								"type": "number",
							},
						},
					},
				},
			}

			set, err := parseAll(objects)

			Expect(err).ToNot(HaveOccurred())
			Expect(set.Size()).To(Equal(len(objects)))
		})
	})

	Describe("collect objects tests", func() {
		It("Should return error for schema with multiple objects with the same name", func() {
			name := "name"
			schema := &Schema{}
			object := map[interface{}]interface{}{
				"id": name,
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						"A_": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"B": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "number",
										},
									},
								},
							},
						},
						"A": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"_B": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "number",
										},
									},
								},
							},
						},
					},
				},
			}
			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())

			_, err = schema.collectObjects(-1, 0)

			expected := fmt.Errorf(
				"invalid schema %s: multiple objects with the same type at object %s",
				name,
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should collect all object", func() {
			schema := &Schema{}
			names := []string{"A", "B", "C"}
			object := map[interface{}]interface{}{
				"id": names[0],
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						names[1]: map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "string",
								},
								names[2]: map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "number",
										},
									},
								},
								names[0]: map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "number",
										},
									},
								},
							},
						},
						names[0]: map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "number",
								},
							},
						},
					},
				},
			}

			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())

			result, err := schema.collectObjects(-1, 0)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(result)).To(Equal(4))

			array := result.ToArray()

			Expect(util.ToGoName(array[0].Name(), "")).To(Equal(names[0]))
			Expect(util.ToGoName(array[1].Name(), "")).To(Equal(names[0] + names[0]))
			Expect(util.ToGoName(array[2].Name(), "")).To(Equal(names[0] + names[1]))
			Expect(util.ToGoName(array[3].Name(), "")).To(Equal(names[0] + names[1] + "Common"))
		})
	})

	Describe("collect properties tests", func() {
		It("Should return error for schema with multiple properties with the same name", func() {
			name := "name"
			schema := &Schema{}
			object := map[interface{}]interface{}{
				"id": name,
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						"A_": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"B": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "number",
										},
									},
								},
							},
						},
						"A": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"_B": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"a": map[interface{}]interface{}{
											"type": "number",
										},
									},
								},
							},
						},
					},
				},
			}

			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())

			_, err = schema.collectProperties(-1, 0)

			expected := fmt.Errorf(
				"invalid schema %s: multiple properties with the same name at object %s",
				name,
				name,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should collect all properties", func() {
			schema := &Schema{}
			names := []string{"A", "B", "C", "D"}
			object := map[interface{}]interface{}{
				"id": names[3],
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						names[1]: map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								names[0]: map[interface{}]interface{}{
									"type": "string",
								},
								names[2]: map[interface{}]interface{}{
									"type": "number",
								},
							},
						},
					},
				},
			}

			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())

			result, err := schema.collectProperties(-1, 0)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(result)).To(Equal(len(names)))

			array := result.ToArray()

			Expect(util.ToGoName(array[0].Name(), "")).To(Equal(names[0]))
			Expect(util.ToGoName(array[1].Name(), "")).To(Equal(names[1]))
			Expect(util.ToGoName(array[2].Name(), "")).To(Equal(names[2]))
			Expect(util.ToGoName(array[3].Name(), "")).To(Equal(names[3]))
		})
	})

	Describe("join tests", func() {
		var (
			schema *Schema
			names  = []string{"a", "b", "c", "d"}
		)

		BeforeEach(func() {
			schema = &Schema{}
			object := map[interface{}]interface{}{
				"id": names[0],
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						names[1]: map[interface{}]interface{}{
							"type": "string",
						},
						names[2]: map[interface{}]interface{}{
							"type": "number",
						},
					},
				},
			}

			err := schema.parse(object)

			Expect(err).ToNot(HaveOccurred())
		})

		It("Should return error when joining to invalid schema", func() {
			data := map[interface{}]interface{}{
				"type": "string",
			}
			other := &Schema{schema: item.CreateProperty(names[3])}
			other.schema.Parse(item.ParseContext{
				Prefix:   "",
				Level:    0,
				Required: true,
				Data:     data,
			})

			err := other.join([]*node{{value: schema}})

			expected := fmt.Errorf(
				"schema %s should be an object",
				other.Name(),
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should join schemas with the same properties", func() {
			nodes := []*node{{value: schema}, {value: schema}}

			err := schema.join(nodes)

			Expect(err).ToNot(HaveOccurred())
		})

		It("Should return error when schemas in nodes share properties", func() {
			other := &Schema{}
			object := map[interface{}]interface{}{
				"id": names[0],
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						names[1]: map[interface{}]interface{}{
							"type": "string",
						},
						names[2]: map[interface{}]interface{}{
							"type": "number",
						},
					},
				},
			}

			err := other.parse(object)

			Expect(err).ToNot(HaveOccurred())

			nodes := []*node{{value: schema}, {value: other}}

			err = schema.join(nodes)

			expected := fmt.Errorf(
				"multiple properties with the same name in bases of schema %s",
				names[0],
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should join correctly", func() {
			other := &Schema{}
			object := map[interface{}]interface{}{
				"id": names[0],
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						names[3]: map[interface{}]interface{}{
							"type": "boolean",
						},
						names[2]: map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"a": map[interface{}]interface{}{
									"type": "string",
								},
								"b": map[interface{}]interface{}{
									"type": "boolean",
								},
							},
						},
					},
				},
			}

			err := other.parse(object)

			Expect(err).ToNot(HaveOccurred())

			err = schema.join([]*node{{value: other}})

			Expect(err).ToNot(HaveOccurred())

			properties, err := schema.collectProperties(-1, 1)

			Expect(err).ToNot(HaveOccurred())

			array := properties.ToArray()

			Expect(len(array)).To(Equal(3))
			Expect(array[2].(*item.Property).IsObject()).To(BeFalse())
		})
	})
})
