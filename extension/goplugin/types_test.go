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
	"encoding/json"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Types tests", func() {
	var (
		env *goplugin.Environment
	)

	BeforeEach(func() {
		env = goplugin.NewEnvironment("test", nil, nil)
	})

	AfterEach(func() {
		env.Stop()
	})

	Describe("Equity", func() {
		DescribeTable("String",
			func(lhv goext.MaybeString, rhv goext.MaybeString, expected bool) {
				Expect(lhv.Equals(rhv)).To(Equal(expected))
			},
			Entry("undefined vs undefined", goext.MakeUndefinedString(), goext.MakeUndefinedString(), true),
			Entry("undefined vs defined", goext.MakeUndefinedString(), goext.MakeString("hello"), false),
			Entry("null vs undefined", goext.MakeNullString(), goext.MakeUndefinedString(), true),
			Entry("null vs defined", goext.MakeNullString(), goext.MakeString("hello"), false),
			Entry("defined vs defined", goext.MakeString("hello"), goext.MakeString("hello"), true),
			Entry("defined vs defined different value", goext.MakeString("hello"), goext.MakeString("world"), false),
		)

		DescribeTable("Float",
			func(lhv goext.MaybeFloat, rhv goext.MaybeFloat, expected bool) {
				Expect(lhv.Equals(rhv)).To(Equal(expected))
			},
			Entry("undefined vs undefined", goext.MakeUndefinedFloat(), goext.MakeUndefinedFloat(), true),
			Entry("undefined vs defined", goext.MakeUndefinedFloat(), goext.MakeFloat(1.23), false),
			Entry("null vs undefined", goext.MakeNullFloat(), goext.MakeUndefinedFloat(), true),
			Entry("null vs defined", goext.MakeNullFloat(), goext.MakeFloat(1.23), false),
			Entry("defined vs defined", goext.MakeFloat(1.23), goext.MakeFloat(1.23), true),
			Entry("defined vs defined different value", goext.MakeFloat(1.23), goext.MakeFloat(3.21), false),
		)

		DescribeTable("Int",
			func(lhv goext.MaybeInt, rhv goext.MaybeInt, expected bool) {
				Expect(lhv.Equals(rhv)).To(Equal(expected))
			},
			Entry("undefined vs undefined", goext.MakeUndefinedInt(), goext.MakeUndefinedInt(), true),
			Entry("undefined vs defined", goext.MakeUndefinedInt(), goext.MakeInt(123), false),
			Entry("null vs undefined", goext.MakeNullInt(), goext.MakeUndefinedInt(), true),
			Entry("null vs defined", goext.MakeNullInt(), goext.MakeInt(123), false),
			Entry("defined vs defined", goext.MakeInt(123), goext.MakeInt(123), true),
			Entry("defined vs defined different value", goext.MakeInt(123), goext.MakeInt(321), false),
		)

		DescribeTable("Bool",
			func(lhv goext.MaybeBool, rhv goext.MaybeBool, expected bool) {
				Expect(lhv.Equals(rhv)).To(Equal(expected))
			},
			Entry("undefined vs undefined", goext.MakeUndefinedBool(), goext.MakeUndefinedBool(), true),
			Entry("undefined vs defined", goext.MakeUndefinedBool(), goext.MakeBool(true), false),
			Entry("null vs undefined", goext.MakeNullBool(), goext.MakeUndefinedBool(), true),
			Entry("null vs defined", goext.MakeNullBool(), goext.MakeBool(true), false),
			Entry("defined vs defined", goext.MakeBool(true), goext.MakeBool(true), true),
			Entry("defined vs defined different value", goext.MakeBool(true), goext.MakeBool(false), false),
		)
	})

	Describe("JSON Marshalling", func() {
		Context("String", func() {
			type TestResource struct {
				Value goext.MaybeString `json:"value"`
			}
			It("value defined", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeString("hello")})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":"hello"}`))
			})

			It("value undefined", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullString()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})
		})

		Context("Float", func() {
			type TestResource struct {
				Value goext.MaybeFloat `json:"value"`
			}
			It("defined value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeFloat(1.23)})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":1.23}`))
			})

			It("undefined value", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullFloat()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})
		})

		Context("Bool", func() {
			type TestResource struct {
				Value goext.MaybeBool `json:"value"`
			}
			It("defined value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeBool(true)})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":true}`))
			})

			It("undefined value", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullBool()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})
		})

		Context("Int", func() {
			type TestResource struct {
				Value goext.MaybeInt `json:"value"`
			}
			It("defined value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeInt(123)})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":123}`))
			})

			It("undefined value", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullInt()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(`{"value":null}`))
			})
		})

	})
})
