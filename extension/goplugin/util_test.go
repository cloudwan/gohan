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

package goplugin_test

import (
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util tests", func() {
	var (
		env *goplugin.Environment
	)

	BeforeEach(func() {
		env = goplugin.NewEnvironment("test", nil, nil)
	})

	AfterEach(func() {
		env.Stop()
	})

	Describe("Marshalling", func() {

		Context("Custom Maybe structure", func() {
			type TestResource struct {
				goext.Maybe
				MaybeInt goext.MaybeInt `json:"maybe_int"`
			}

			type TestResourceWithNestedMaybeStructure struct {
				String          string `json:"string"`
				CustomStructure struct {
					goext.Maybe
					NestedString string `json:"nested_string"`
				} `json:"custom_structure"`
			}

			It("Empty map", func() {
				input := map[string]interface{}{}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.HasValue()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("Fills maybe state", func() {
				input := map[string]interface{}{
					"maybe_int": 1,
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.HasValue()).To(BeTrue())
				Expect(resource.MaybeInt.HasValue()).To(BeTrue())
				Expect(resource.MaybeInt.Value()).To(Equal(1))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("Undefined nested structures", func() {
				input := map[string]interface{}{
					"string": "hello",
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResourceWithNestedMaybeStructure{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResourceWithNestedMaybeStructure)
				Expect(resource.CustomStructure.IsUndefined()).To(BeTrue())
				Expect(resource.String).To(Equal("hello"))
				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("null nested structures", func() {
				input := map[string]interface{}{
					"string":           "hello",
					"custom_structure": nil,
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResourceWithNestedMaybeStructure{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResourceWithNestedMaybeStructure)
				Expect(resource.CustomStructure.IsNull()).To(BeTrue())
				Expect(resource.String).To(Equal("hello"))
				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("defined nested structures", func() {
				input := map[string]interface{}{
					"string": "hello",
					"custom_structure": map[string]interface{}{
						"nested_string": "world",
					},
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResourceWithNestedMaybeStructure{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResourceWithNestedMaybeStructure)
				Expect(resource.String).To(Equal("hello"))
				Expect(resource.CustomStructure.HasValue()).To(BeTrue())
				Expect(resource.CustomStructure.NestedString).To(Equal("world"))
				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})
		})

		Context("Undefined null values", func() {
			type TestResource struct {
				MaybeInt goext.MaybeInt `json:"maybe_int"`
			}

			It("value defined", func() {
				input := map[string]interface{}{
					"maybe_int": 1,
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.MaybeInt.HasValue()).To(BeTrue())
				Expect(resource.MaybeInt.Value()).To(Equal(1))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("value undefined", func() {
				input := map[string]interface{}{}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.MaybeInt.IsUndefined()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("value null", func() {
				input := map[string]interface{}{
					"maybe_int": nil,
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.MaybeInt.IsNull()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})
		})

		Context("Null values", func() {
			type TestResource struct {
				NullInteger goext.MaybeInt    `json:"maybe_integer,omitempty"`
				NullBool    goext.MaybeBool   `json:"maybe_bool,omitempty"`
				NullString  goext.MaybeString `json:"maybe_string,omitempty"`
				NullFloat   goext.MaybeFloat  `json:"maybe_float,omitempty"`
			}

			It("integer", func() {
				input := map[string]interface{}{
					"maybe_integer": 123,
					"maybe_bool":    nil,
					"maybe_string":  nil,
					"maybe_float":   nil,
				}
				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.NullInteger.HasValue()).To(BeTrue())
				Expect(resource.NullInteger.Value()).To(Equal(123))
				Expect(resource.NullFloat.IsNull()).To(BeTrue())
				Expect(resource.NullBool.IsNull()).To(BeTrue())
				Expect(resource.NullString.IsNull()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("float", func() {
				input := map[string]interface{}{
					"maybe_float":   123.0,
					"maybe_integer": nil,
					"maybe_bool":    nil,
					"maybe_string":  nil,
				}
				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.NullFloat.HasValue()).To(BeTrue())
				Expect(resource.NullFloat.Value()).To(Equal(123.0))
				Expect(resource.NullInteger.IsNull()).To(BeTrue())
				Expect(resource.NullBool.IsNull()).To(BeTrue())
				Expect(resource.NullString.IsNull()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("string", func() {
				input := map[string]interface{}{
					"maybe_string":  "hello",
					"maybe_integer": nil,
					"maybe_bool":    nil,
					"maybe_float":   nil,
				}
				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.NullString.HasValue()).To(BeTrue())
				Expect(resource.NullString.Value()).To(Equal("hello"))
				Expect(resource.NullFloat.IsNull()).To(BeTrue())
				Expect(resource.NullBool.IsNull()).To(BeTrue())
				Expect(resource.NullInteger.IsNull()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("bool", func() {
				input := map[string]interface{}{
					"maybe_bool":    true,
					"maybe_integer": nil,
					"maybe_string":  nil,
					"maybe_float":   nil,
				}
				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.NullBool.HasValue()).To(BeTrue())
				Expect(resource.NullBool.Value()).To(BeTrue())
				Expect(resource.NullFloat.IsNull()).To(BeTrue())
				Expect(resource.NullString.IsNull()).To(BeTrue())
				Expect(resource.NullString.IsNull()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})
		})

		Context("Object", func() {
			type TestResource struct {
				Strings []string `json:"strings"`
			}

			It("with undefined slice", func() {
				input := map[string]interface{}{}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.Strings).To(BeNil())
				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("With slice of primitives", func() {
				input := map[string]interface{}{
					"strings": []string{
						"abc", "def",
					},
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.Strings).To(HaveLen(2))
				Expect(resource.Strings[0]).To(Equal("abc"))
				Expect(resource.Strings[1]).To(Equal("def"))
			})
		})

		Context("Nested Objects", func() {
			type TestResource struct {
				Obj struct {
					NestedObj struct {
						NullString goext.MaybeString `json:"maybe_string"`
					} `json:"nested_obj"`
				} `json:"obj"`
			}

			type TestResourceWithSlice struct {
				Obj struct {
					NestedSliceOfObj []struct {
						String string `json:"string"`
					} `json:"nested_slice_of_obj"`
				} `json:"obj"`
			}

			It("nested with empty slice of objects", func() {
				input := map[string]interface{}{
					"obj": map[string]interface{}{
						"nested_slice_of_obj": []map[string]interface{}{},
					},
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResourceWithSlice{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResourceWithSlice)
				Expect(resource.Obj.NestedSliceOfObj).To(HaveLen(0))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("nested with filled slice of objects", func() {
				input := map[string]interface{}{
					"obj": map[string]interface{}{
						"nested_slice_of_obj": []map[string]interface{}{
							{
								"string": "hello",
							},
							{
								"string": "hello",
							},
						},
					},
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResourceWithSlice{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResourceWithSlice)
				Expect(resource.Obj.NestedSliceOfObj).To(HaveLen(2))
				Expect(resource.Obj.NestedSliceOfObj[0].String).To(Equal("hello"))
				Expect(resource.Obj.NestedSliceOfObj[1].String).To(Equal("hello"))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("nested objects with null field", func() {
				input := map[string]interface{}{
					"obj": map[string]interface{}{
						"nested_obj": map[string]interface{}{
							"maybe_string": nil,
						},
					},
				}
				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.Obj.NestedObj.NullString.IsNull()).To(BeTrue())

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			It("nested object with filled field", func() {
				input := map[string]interface{}{
					"obj": map[string]interface{}{
						"nested_obj": map[string]interface{}{
							"maybe_string": "hello",
						},
					},
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.Obj.NestedObj.NullString.HasValue()).To(BeTrue())
				Expect(resource.Obj.NestedObj.NullString.Value()).To(Equal("hello"))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})
		})

		Context("object with interface", func() {
			type TestResourceWithInterface struct {
				IInt    interface{} `json:"iint"`
				Ifloat  interface{} `json:"ifloat"`
				Ibool   interface{} `json:"ibool"`
				Istring interface{} `json:"istring"`
			}

			It("object with primitive string", func() {
				input := map[string]interface{}{
					"iint":    42,
					"ifloat":  69.0,
					"ibool":   true,
					"istring": "abc",
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResourceWithInterface{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResourceWithInterface)
				Expect(resource.IInt).To(Equal(42))
				Expect(resource.Ifloat).To(Equal(69.0))
				Expect(resource.Ibool).To(BeTrue())
				Expect(resource.Istring).To(Equal("abc"))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})

			type TestResource struct {
				Field1 *TestResourceWithInterface `json:"field1"`
				Field2 *TestResourceWithInterface `json:"field2"`
			}

			It("pointers to object with interfaces", func() {
				input := map[string]interface{}{
					"field1": map[string]interface{}{
						"iint":    42,
						"ifloat":  69.0,
						"ibool":   true,
						"istring": "abc",
					},
					"field2": map[string]interface{}{
						"iint":    69,
						"ifloat":  42.0,
						"ibool":   false,
						"istring": "cba",
					},
				}

				rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
				Expect(err).To(BeNil())
				resource := rawResource.(*TestResource)
				Expect(resource.Field1.IInt).To(Equal(42))
				Expect(resource.Field1.Ifloat).To(Equal(69.0))
				Expect(resource.Field1.Ibool).To(BeTrue())
				Expect(resource.Field1.Istring).To(Equal("abc"))
				Expect(resource.Field2.IInt).To(Equal(69))
				Expect(resource.Field2.Ifloat).To(Equal(42.0))
				Expect(resource.Field2.Ibool).To(BeFalse())
				Expect(resource.Field2.Istring).To(Equal("cba"))

				mapRepresentation := env.Util().ResourceToMap(resource)
				Expect(mapRepresentation).To(Equal(input))
			})
		})

		Context("Slice", func() {
			Context("of objects", func() {
				type TestResource struct {
					ArrayOfPtrsToObj []*struct {
						NullInteger goext.MaybeInt    `json:"maybe_integer,omitempty"`
						String      string            `json:"string"`
						Ptr         goext.MaybeString `json:"ptr,omitempty"`
						Integer     int               `json:"integer"`
					} `json:"array_of_ptrs_to_obj"`
				}

				It("input as slice of interfaces but maps inside", func() {
					structAsMap := map[string]interface{}{
						"maybe_integer": 123,
						"string":        "hello",
					}
					input := map[string]interface{}{
						"array_of_ptrs_to_obj": []interface{}{
							structAsMap,
						},
					}
					rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
					Expect(err).To(BeNil())
					resource := rawResource.(*TestResource)
					Expect(resource.ArrayOfPtrsToObj).To(HaveLen(1))
					Expect(resource.ArrayOfPtrsToObj[0].String).To(Equal("hello"))
					Expect(resource.ArrayOfPtrsToObj[0].NullInteger.HasValue()).To(BeTrue())
					Expect(resource.ArrayOfPtrsToObj[0].NullInteger.Value()).To(Equal(123))
					Expect(resource.ArrayOfPtrsToObj[0].Ptr.IsUndefined()).To(BeTrue())
					Expect(resource.ArrayOfPtrsToObj[0].Integer).To(Equal(0))

					mapRepresentation := env.Util().ResourceToMap(resource)
					Expect(mapRepresentation["array_of_ptrs_to_obj"].([]map[string]interface{})).To(HaveLen(1))
					Expect(mapRepresentation["array_of_ptrs_to_obj"].([]map[string]interface{})[0]).To(HaveKeyWithValue("integer", 0))
				})

				It("map with single null", func() {
					input := map[string]interface{}{
						"array_of_ptrs_to_obj": []map[string]interface{}{
							nil,
						},
					}
					rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
					Expect(err).To(BeNil())
					resource := rawResource.(*TestResource)
					Expect(resource.ArrayOfPtrsToObj).To(HaveLen(1))
					Expect(resource.ArrayOfPtrsToObj[0]).To(BeNil())

					mapRepresentation := env.Util().ResourceToMap(resource)
					Expect(mapRepresentation).To(Equal(input))
				})

				It("map not containing required values results in zero value", func() {
					input := map[string]interface{}{
						"array_of_ptrs_to_obj": []map[string]interface{}{
							{
								"maybe_integer": 123,
								"string":        "hello",
							},
						},
					}
					rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
					Expect(err).To(BeNil())
					resource := rawResource.(*TestResource)
					Expect(resource.ArrayOfPtrsToObj).To(HaveLen(1))
					Expect(resource.ArrayOfPtrsToObj[0].String).To(Equal("hello"))
					Expect(resource.ArrayOfPtrsToObj[0].NullInteger.HasValue()).To(BeTrue())
					Expect(resource.ArrayOfPtrsToObj[0].NullInteger.Value()).To(Equal(123))
					Expect(resource.ArrayOfPtrsToObj[0].Ptr.IsUndefined()).To(BeTrue())
					Expect(resource.ArrayOfPtrsToObj[0].Integer).To(Equal(0))

					mapRepresentation := env.Util().ResourceToMap(resource)
					Expect(mapRepresentation["array_of_ptrs_to_obj"].([]map[string]interface{})).To(HaveLen(1))
					Expect(mapRepresentation["array_of_ptrs_to_obj"].([]map[string]interface{})[0]).To(HaveKeyWithValue("integer", 0))
				})
			})

			Context("of objects containing slice of primitives", func() {
				type TestResource struct {
					ArrayOfPtrsToObj *struct {
						Strings []string `json:"strings"`
					} `json:"array_of_ptrs_to_obj"`
				}

				It("map not containing required values results in zero value", func() {
					sliceOfStrings := []interface{}{"hello", "world"}
					input := map[string]interface{}{
						"array_of_ptrs_to_obj": map[string]interface{}{
							"strings": sliceOfStrings,
						},
					}
					rawResource, err := env.Util().ResourceFromMapForType(input, TestResource{})
					Expect(err).To(BeNil())
					resource := rawResource.(*TestResource)
					Expect(resource.ArrayOfPtrsToObj.Strings).To(HaveLen(2))
					Expect(resource.ArrayOfPtrsToObj.Strings[0]).To(Equal(sliceOfStrings[0]))
					Expect(resource.ArrayOfPtrsToObj.Strings[1]).To(Equal(sliceOfStrings[1]))

					mapRepresentation := env.Util().ResourceToMap(resource)
					Expect(mapRepresentation).To(Equal(input))
				})
			})
		})

	})
})
