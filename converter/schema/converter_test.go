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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("converter tests", func() {
	Describe("error tests", func() {
		var (
			validSchema = []map[interface{}]interface{}{
				{
					"id":     "my_id",
					"schema": map[interface{}]interface{}{},
				},
			}
			invalidSchema = []map[interface{}]interface{}{
				{
					"invalid schema": "test",
				},
			}
		)

		Describe("parse all errors", func() {
			var expected = fmt.Errorf("schema does not have an id")

			It("Should return error for invalid other schema", func() {
				_, err := Convert(validSchema, invalidSchema, "", "", "")

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expected))
			})

			It("Should return error for invalid other schema", func() {
				_, err := Convert(invalidSchema, validSchema, "", "", "")

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expected))
			})
		})

		Describe("collect errors", func() {
			It("Should return error for multiple schemas with the same name", func() {
				_, err := Convert(validSchema, validSchema, "", "", "")

				expected := fmt.Errorf("multiple schemas with the same name")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expected))
			})

			It("Should return error for multiple objects with the same name", func() {
				name := "a"
				schemas := []map[interface{}]interface{}{
					{
						"id": name,
						"schema": map[interface{}]interface{}{
							"type": "object",
							"properties": map[interface{}]interface{}{
								"__": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"_": map[interface{}]interface{}{
											"type":       "object",
											"properties": map[interface{}]interface{}{},
										},
									},
								},
								"_": map[interface{}]interface{}{
									"type": "object",
									"properties": map[interface{}]interface{}{
										"__": map[interface{}]interface{}{
											"type":       "object",
											"properties": map[interface{}]interface{}{},
										},
									},
								},
							},
						},
					},
				}

				_, err := Convert(nil, schemas, "", "", "")

				expected := fmt.Errorf(
					"invalid schema %s: multiple objects with the same type at object %s",
					name,
					name,
				)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expected))
			})
		})
	})
	Describe("valid data tests", func() {
		It("Should convert valid schemas", func() {
			other := []map[interface{}]interface{}{
				{
					"id": "base",
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"id": map[interface{}]interface{}{
								"type": "string",
							},
							"ip": map[interface{}]interface{}{
								"type": "number",
							},
							"object": map[interface{}]interface{}{
								"type": "object",
								"properties": map[interface{}]interface{}{
									"x": map[interface{}]interface{}{
										"type":    "string",
										"default": "abc",
									},
									"y": map[interface{}]interface{}{
										"type": "string",
									},
								},
								"required": []interface{}{
									"y",
								},
							},
						},
					},
				},
				{
					"id":     "middle",
					"parent": "parent",
					"extends": []interface{}{
						"base",
					},
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"null": map[interface{}]interface{}{
								"type": []interface{}{
									"boolean",
									"null",
								},
							},
							"array": map[interface{}]interface{}{
								"type": "array",
								"items": map[interface{}]interface{}{
									"type": "array",
									"items": map[interface{}]interface{}{
										"type": "number",
									},
								},
							},
							"nested": map[interface{}]interface{}{
								"type": "object",
								"properties": map[interface{}]interface{}{
									"first": map[interface{}]interface{}{
										"type": "object",
										"properties": map[interface{}]interface{}{
											"second": map[interface{}]interface{}{
												"type":       "object",
												"properties": map[interface{}]interface{}{},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			toConvert := []map[interface{}]interface{}{
				{
					"id": "general",
					"extends": []interface{}{
						"middle",
						"base",
					},
					"schema": map[interface{}]interface{}{
						"type": "object",
						"properties": map[interface{}]interface{}{
							"complex": map[interface{}]interface{}{
								"type": "array",
								"items": map[interface{}]interface{}{
									"type": "array",
									"items": map[interface{}]interface{}{
										"type": "object",
										"properties": map[interface{}]interface{}{
											"for": map[interface{}]interface{}{
												"type": "number",
											},
											"int": map[interface{}]interface{}{
												"type": "boolean",
											},
										},
									},
								},
							},
							"tree": map[interface{}]interface{}{
								"type": "object",
								"properties": map[interface{}]interface{}{
									"left": map[interface{}]interface{}{
										"type": "object",
										"properties": map[interface{}]interface{}{
											"leaf_first": map[interface{}]interface{}{
												"type": "string",
											},
											"leaf_second": map[interface{}]interface{}{
												"type": "object",
												"properties": map[interface{}]interface{}{
													"value": map[interface{}]interface{}{
														"type": "number",
													},
												},
											},
										},
									},
									"right": map[interface{}]interface{}{
										"type": "object",
										"properties": map[interface{}]interface{}{
											"leaf_third": map[interface{}]interface{}{
												"type": "array",
												"items": map[interface{}]interface{}{
													"type": "boolean",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					"id": "only_derive",
					"extends": []interface{}{
						"base",
					},
					"schema": map[interface{}]interface{}{},
				},
				{
					"id":     "empty",
					"schema": map[interface{}]interface{}{},
				},
			}

			generated, err := Convert(other, toConvert, "", "Gen", "goext")
			Expect(err).ToNot(HaveOccurred())

			generalGenerated := `type IGeneralGen interface {
	GetArray() [][]float64
	SetArray([][]float64)
	GetComplex() [][]IGeneralComplexGen
	SetComplex([][]IGeneralComplexGen)
	GetID() string
	SetID(string)
	GetIP() goext.NullFloat
	SetIP(goext.NullFloat)
	GetNested() IMiddleNestedGen
	SetNested(IMiddleNestedGen)
	GetNull() goext.NullBool
	SetNull(goext.NullBool)
	GetObject() IBaseObjectGen
	SetObject(IBaseObjectGen)
	GetParentID() string
	SetParentID(string)
	GetTree() IGeneralTreeGen
	SetTree(IGeneralTreeGen)
}
`
			onlyDeriveGenerated := `type IOnlyDeriveGen interface {
	GetID() string
	SetID(string)
	GetIP() goext.NullFloat
	SetIP(goext.NullFloat)
	GetObject() IBaseObjectGen
	SetObject(IBaseObjectGen)
}
`
			generalTreeLeftLeafSecondGenerated := `type IGeneralTreeLeftLeafSecondGen interface {
	GetValue() float64
	SetValue(float64)
}
`
			middleNestedFirstGenerated := `type IMiddleNestedFirstGen interface {
	GetSecond() interface{}
	SetSecond(interface{})
}
`
			middleNestedGenerated := `type IMiddleNestedGen interface {
	GetFirst() IMiddleNestedFirstGen
	SetFirst(IMiddleNestedFirstGen)
}
`
			generalComplexGenerated := `type IGeneralComplexGen interface {
	GetFor() float64
	SetFor(float64)
	GetInt() bool
	SetInt(bool)
}
`
			generalTreeLeftGenerated := `type IGeneralTreeLeftGen interface {
	GetLeafFirst() string
	SetLeafFirst(string)
	GetLeafSecond() IGeneralTreeLeftLeafSecondGen
	SetLeafSecond(IGeneralTreeLeftLeafSecondGen)
}
`
			generalTreeGenerated := `type IGeneralTreeGen interface {
	GetLeft() IGeneralTreeLeftGen
	SetLeft(IGeneralTreeLeftGen)
	GetRight() IGeneralTreeRightGen
	SetRight(IGeneralTreeRightGen)
}
`
			baseObjectGenerated := `type IBaseObjectGen interface {
	GetX() string
	SetX(string)
	GetY() string
	SetY(string)
}
`
			generalTreeRightGenerated := `type IGeneralTreeRightGen interface {
	GetLeafThird() []bool
	SetLeafThird([]bool)
}
`
			generalInterface := `type IGeneral interface {
	IGeneralGen
}
`
			onlyDeriveInterface := `type IOnlyDerive interface {
	IOnlyDeriveGen
}
`
			generalTreeLeftInterface := `type IGeneralTreeLeft interface {
	IGeneralTreeLeftGen
}
`
			middleNestedFirstInterface := `type IMiddleNestedFirst interface {
	IMiddleNestedFirstGen
}
`
			middleNestedInterface := `type IMiddleNested interface {
	IMiddleNestedGen
}
`
			generalComplexInterface := `type IGeneralComplex interface {
	IGeneralComplexGen
}
`
			generalTreeLeftLeafSecondInterface := `type IGeneralTreeLeftLeafSecond interface {
	IGeneralTreeLeftLeafSecondGen
}
`
			generalTreeInterface := `type IGeneralTree interface {
	IGeneralTreeGen
}
`
			generalTreeRightInterface := `type IGeneralTreeRight interface {
	IGeneralTreeRightGen
}
`
			baseObjectInterface := `type IBaseObject interface {
	IBaseObjectGen
}
`
			generalStruct := `type General struct {
	Array [][]float64 ` + "`" + `db:"array"` + ` json:"array"` + "`" + `
	Complex [][]*GeneralComplex ` + "`" + `db:"complex"` + ` json:"complex"` + "`" + `
	ID string ` + "`" + `db:"id"` + ` json:"id"` + "`" + `
	IP goext.NullFloat ` + "`" + `db:"ip"` + ` json:"ip,omitempty"` + "`" + `
	Nested *MiddleNested ` + "`" + `db:"nested"` + ` json:"nested"` + "`" + `
	Null goext.NullBool ` + "`" + `db:"null"` + ` json:"null,omitempty"` + "`" + `
	Object *BaseObject ` + "`" + `db:"object"` + ` json:"object"` + "`" + `
	ParentID string ` + "`" + `db:"parent_id"` + ` json:"parent_id"` + "`" + `
	Tree *GeneralTree ` + "`" + `db:"tree"` + ` json:"tree"` + "`" + `
}
`
			onlyDeriveStruct := `type OnlyDerive struct {
	ID string ` + "`" + `db:"id"` + ` json:"id"` + "`" + `
	IP goext.NullFloat ` + "`" + `db:"ip"` + ` json:"ip,omitempty"` + "`" + `
	Object *BaseObject ` + "`" + `db:"object"` + ` json:"object"` + "`" + `
}
`
			generalTreeLeftLeafSecondStruct := `type GeneralTreeLeftLeafSecond struct {
	Value float64 ` + "`" + `json:"value,omitempty"` + "`" + `
}
`
			middleNestedFirstStruct := `type MiddleNestedFirst struct {
	Second interface{} ` + "`" + `json:"second"` + "`" + `
}
`
			middleNestedStruct := `type MiddleNested struct {
	First *MiddleNestedFirst ` + "`" + `json:"first"` + "`" + `
}
`
			generalComplexStruct := `type GeneralComplex struct {
	For float64 ` + "`" + `json:"for,omitempty"` + "`" + `
	Int bool ` + "`" + `json:"int,omitempty"` + "`" + `
}
`
			generalTreeLeftStruct := `type GeneralTreeLeft struct {
	LeafFirst string ` + "`" + `json:"leaf_first,omitempty"` + "`" + `
	LeafSecond *GeneralTreeLeftLeafSecond ` + "`" + `json:"leaf_second"` + "`" + `
}
`
			generalTreeStruct := `type GeneralTree struct {
	Left *GeneralTreeLeft ` + "`" + `json:"left"` + "`" + `
	Right *GeneralTreeRight ` + "`" + `json:"right"` + "`" + `
}
`
			baseObjectStruct := `type BaseObject struct {
	X string ` + "`" + `json:"x"` + "`" + `
	Y string ` + "`" + `json:"y"` + "`" + `
}
`
			generalTreeRightStruct := `type GeneralTreeRight struct {
	LeafThird []bool ` + "`" + `json:"leaf_third"` + "`" + `
}
`
			generalImplementation := `func (general *General) GetArray() [][]float64 {
	return general.Array
}

func (general *General) SetArray(array [][]float64) {
	general.Array = array
}

func (general *General) GetComplex() [][]IGeneralComplexGen {
	result := make([][]IGeneralComplexGen, len(general.Complex))
	for i := range general.Complex {
		result[i] = make([]IGeneralComplexGen, len(general.Complex[i]))
		for j := range general.Complex[i] {
			result[i][j] = general.Complex[i][j]
		}
	}
	return result
}

func (general *General) SetComplex(complex [][]IGeneralComplexGen) {
	general.Complex = make([][]*GeneralComplex, len(complex))
	for i := range complex {
		general.Complex[i] = make([]*GeneralComplex, len(complex[i]))
		for j := range complex[i] {
			general.Complex[i][j], _ = complex[i][j].(*GeneralComplex)
		}
	}
}

func (general *General) GetID() string {
	return general.ID
}

func (general *General) SetID(id string) {
	general.ID = id
}

func (general *General) GetIP() goext.NullFloat {
	return general.IP
}

func (general *General) SetIP(ip goext.NullFloat) {
	general.IP = ip
}

func (general *General) GetNested() IMiddleNestedGen {
	return general.Nested
}

func (general *General) SetNested(nested IMiddleNestedGen) {
	general.Nested, _ = nested.(*MiddleNested)
}

func (general *General) GetNull() goext.NullBool {
	return general.Null
}

func (general *General) SetNull(null goext.NullBool) {
	general.Null = null
}

func (general *General) GetObject() IBaseObjectGen {
	return general.Object
}

func (general *General) SetObject(object IBaseObjectGen) {
	general.Object, _ = object.(*BaseObject)
}

func (general *General) GetParentID() string {
	return general.ParentID
}

func (general *General) SetParentID(parentID string) {
	general.ParentID = parentID
}

func (general *General) GetTree() IGeneralTreeGen {
	return general.Tree
}

func (general *General) SetTree(tree IGeneralTreeGen) {
	general.Tree, _ = tree.(*GeneralTree)
}
`
			onlyDeriveImplementation := `func (onlyDerive *OnlyDerive) GetID() string {
	return onlyDerive.ID
}

func (onlyDerive *OnlyDerive) SetID(id string) {
	onlyDerive.ID = id
}

func (onlyDerive *OnlyDerive) GetIP() goext.NullFloat {
	return onlyDerive.IP
}

func (onlyDerive *OnlyDerive) SetIP(ip goext.NullFloat) {
	onlyDerive.IP = ip
}

func (onlyDerive *OnlyDerive) GetObject() IBaseObjectGen {
	return onlyDerive.Object
}

func (onlyDerive *OnlyDerive) SetObject(object IBaseObjectGen) {
	onlyDerive.Object, _ = object.(*BaseObject)
}
`
			generalTreeLeftLeafSecondImplementation := `func (generalTreeLeftLeafSecond *GeneralTreeLeftLeafSecond) GetValue() float64 {
	return generalTreeLeftLeafSecond.Value
}

func (generalTreeLeftLeafSecond *GeneralTreeLeftLeafSecond) SetValue(value float64) {
	generalTreeLeftLeafSecond.Value = value
}
`
			middleNestedFirstImplementation := `func (middleNestedFirst *MiddleNestedFirst) GetSecond() interface{} {
	return middleNestedFirst.Second
}

func (middleNestedFirst *MiddleNestedFirst) SetSecond(second interface{}) {
	middleNestedFirst.Second = second
}
`
			middleNestedImplementation := `func (middleNested *MiddleNested) GetFirst() IMiddleNestedFirstGen {
	return middleNested.First
}

func (middleNested *MiddleNested) SetFirst(first IMiddleNestedFirstGen) {
	middleNested.First, _ = first.(*MiddleNestedFirst)
}
`
			generalComplexImplementation := `func (generalComplex *GeneralComplex) GetFor() float64 {
	return generalComplex.For
}

func (generalComplex *GeneralComplex) SetFor(forObject float64) {
	generalComplex.For = forObject
}

func (generalComplex *GeneralComplex) GetInt() bool {
	return generalComplex.Int
}

func (generalComplex *GeneralComplex) SetInt(int bool) {
	generalComplex.Int = int
}
`
			generalTreeLeftImplementation := `func (generalTreeLeft *GeneralTreeLeft) GetLeafFirst() string {
	return generalTreeLeft.LeafFirst
}

func (generalTreeLeft *GeneralTreeLeft) SetLeafFirst(leafFirst string) {
	generalTreeLeft.LeafFirst = leafFirst
}

func (generalTreeLeft *GeneralTreeLeft) GetLeafSecond() IGeneralTreeLeftLeafSecondGen {
	return generalTreeLeft.LeafSecond
}

func (generalTreeLeft *GeneralTreeLeft) SetLeafSecond(leafSecond IGeneralTreeLeftLeafSecondGen) {
	generalTreeLeft.LeafSecond, _ = leafSecond.(*GeneralTreeLeftLeafSecond)
}
`
			generalTreeImplementation := `func (generalTree *GeneralTree) GetLeft() IGeneralTreeLeftGen {
	return generalTree.Left
}

func (generalTree *GeneralTree) SetLeft(left IGeneralTreeLeftGen) {
	generalTree.Left, _ = left.(*GeneralTreeLeft)
}

func (generalTree *GeneralTree) GetRight() IGeneralTreeRightGen {
	return generalTree.Right
}

func (generalTree *GeneralTree) SetRight(right IGeneralTreeRightGen) {
	generalTree.Right, _ = right.(*GeneralTreeRight)
}
`
			baseObjectImplementation := `func (baseObject *BaseObject) GetX() string {
	return baseObject.X
}

func (baseObject *BaseObject) SetX(x string) {
	baseObject.X = x
}

func (baseObject *BaseObject) GetY() string {
	return baseObject.Y
}

func (baseObject *BaseObject) SetY(y string) {
	baseObject.Y = y
}
`
			generalTreeRightImplementation := `func (generalTreeRight *GeneralTreeRight) GetLeafThird() []bool {
	return generalTreeRight.LeafThird
}

func (generalTreeRight *GeneralTreeRight) SetLeafThird(leafThird []bool) {
	generalTreeRight.LeafThird = leafThird
}
`
			expectedGenerated := []string{
				baseObjectGenerated,
				generalGenerated,
				generalComplexGenerated,
				generalTreeGenerated,
				generalTreeLeftGenerated,
				generalTreeLeftLeafSecondGenerated,
				generalTreeRightGenerated,
				middleNestedGenerated,
				middleNestedFirstGenerated,
				onlyDeriveGenerated,
			}
			expectedInterfaces := []string{
				baseObjectInterface,
				generalInterface,
				generalComplexInterface,
				generalTreeInterface,
				generalTreeLeftInterface,
				generalTreeLeftLeafSecondInterface,
				generalTreeRightInterface,
				middleNestedInterface,
				middleNestedFirstInterface,
				onlyDeriveInterface,
			}
			expectedStructs := []string{
				baseObjectStruct,
				generalStruct,
				generalComplexStruct,
				generalTreeStruct,
				generalTreeLeftStruct,
				generalTreeLeftLeafSecondStruct,
				generalTreeRightStruct,
				middleNestedStruct,
				middleNestedFirstStruct,
				onlyDeriveStruct,
			}
			expectedImplementations := []string{
				baseObjectImplementation,
				generalImplementation,
				generalComplexImplementation,
				generalTreeImplementation,
				generalTreeLeftImplementation,
				generalTreeLeftLeafSecondImplementation,
				generalTreeRightImplementation,
				middleNestedImplementation,
				middleNestedFirstImplementation,
				onlyDeriveImplementation,
			}

			Expect(generated.RawInterfaces).To(Equal(expectedGenerated))
			Expect(generated.Interfaces).To(Equal(expectedInterfaces))
			Expect(generated.Structs).To(Equal(expectedStructs))
			Expect(generated.Implementations).To(Equal(expectedImplementations))
		})
	})
})
