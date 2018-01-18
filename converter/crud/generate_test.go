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

package crud

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("generate tests", func() {
	Describe("Fetch tests", func() {
		It("Should generate a correct fetch function", func() {
			expected := `func FetchA(` +
				`schema goext.ISchema, ` +
				`id string, ` +
				`context goext.Context` +
				`) (esi.IA, error) {
	result, err := schema.Fetch(id, context)
	if err != nil {
		return nil, err
	}
	return result.(esi.IA), nil
}
`
			result := GenerateFetch(
				"goext",
				"A",
				"esi.IA",
				Params{Raw: false, Lock: false, Filter: false},
			)

			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct lock fetch raw function", func() {
			expected := `func LockFetchRawB(` +
				`schema goext.ISchema, ` +
				`id string, ` +
				`context goext.Context, ` +
				`policy goext.LockPolicy` +
				`) (*resources.B, error) {
	result, err := schema.LockFetchRaw(id, context, policy)
	if err != nil {
		return nil, err
	}
	return result.(*resources.B), nil
}
`
			result := GenerateFetch(
				"goext",
				"B",
				"*resources.B",
				Params{Raw: true, Lock: true, Filter: false},
			)

			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct lock fetch filter raw function", func() {
			expected := `func LockFetchFilterRawB(` +
				`schema goext.ISchema, ` +
				`filter goext.Filter, ` +
				`context goext.Context, ` +
				`policy goext.LockPolicy` +
				`) (*resources.B, error) {
	result, err := schema.LockFetchFilterRaw(filter, context, policy)
	if err != nil {
		return nil, err
	}
	return result.(*resources.B), nil
}
`
			result := GenerateFetch(
				"goext",
				"B",
				"*resources.B",
				Params{Raw: true, Lock: true, Filter: true},
			)

			Expect(result).To(Equal(expected))
		})
	})

	Describe("List tests", func() {
		It("Should generate a correct list function", func() {
			expected := `func ListA(` +
				`schema goext.ISchema, ` +
				`filter goext.Filter, ` +
				`paginator *goext.Paginator, ` +
				`context goext.Context` +
				`) ([]esi.IA, error) {
	list, err := schema.List(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	result := make([]esi.IA, len(list))
	for i, object := range list {
		result[i] = object.(esi.IA)
	}
	return result, nil
}
`
			result := GenerateList(
				"goext",
				"A",
				"esi.IA",
				Params{Raw: false, Lock: false},
			)
			Expect(result).To(Equal(expected))
		})

		It("Should generate a correct lock list raw function", func() {
			expected := `func LockListRawB(` +
				`schema goext.ISchema, ` +
				`filter goext.Filter, ` +
				`paginator *goext.Paginator, ` +
				`context goext.Context, ` +
				`policy goext.LockPolicy` +
				`) ([]*resources.B, error) {
	list, err := schema.LockListRaw(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	result := make([]*resources.B, len(list))
	for i, object := range list {
		result[i] = object.(*resources.B)
	}
	return result, nil
}
`
			result := GenerateList(
				"goext",
				"B",
				"*resources.B",
				Params{Raw: true, Lock: true},
			)
			Expect(result).To(Equal(expected))
		})
	})
})
