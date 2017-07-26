// Copyright (C) 2015 NTT Innovation Institute, Inc.
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
	"sort"
)

func getData(data map[string]interface{}) ([]string, []interface{}) {
	keys := make([]string, len(data))
	values := make([]interface{}, len(data))
	i := 0
	for k, v := range data {
		keys[i] = k
		values[i] = v
		i++
	}
	return keys, values
}

func emptyAction() Action {
	return NewAction(
		"empty",
		"GET",
		"/empty/",
		"empty action",
		nil,
		nil,
		nil,
	)
}

func noInputTypeAction() Action {
	return NewAction(
		"noInputType",
		"GET",
		"/noInputType/:id/",
		"action with no input type",
		map[string]interface{}{
			"properties": map[string]interface{}{
				"a": map[string]interface{}{
					"type": "number",
				},
				"b": "no map",
			},
		},
		nil,
		nil,
	)
}

func invalidInputTypeAction() Action {
	return NewAction(
		"invalidInputType",
		"GET",
		"/invalidType",
		"action with invalid input type",
		map[string]interface{}{
			"type": struct{}{},
		},
		nil,
		nil,
	)
}

func invalidPropertiesType() Action {
	return NewAction(
		"invalidPropertiesType",
		"GET",
		"/invalidPropertiesType/",
		"action with invalid input properties type",
		map[string]interface{}{
			"type":       "object",
			"properties": "abc",
		},
		nil,
		nil,
	)
}

func invalidProperties() Action {
	return NewAction(
		"invalidProperties",
		"GET",
		"/invalidProperties/",
		"action with invalid input properties",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": "null",
				"b": map[string]interface{}{
					"test": "null",
				},
				"c": map[string]interface{}{
					"type": struct{}{},
				},
			},
		},
		nil,
		nil,
	)
}

func validAction() Action {
	return NewAction(
		"validAction",
		"GET",
		"/valid/:id/action/",
		"valid action",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{
					"type": "number",
				},
				"b": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"1": map[string]interface{}{
							"type": "string",
						},
						"2": map[string]interface{}{
							"type": "boolean",
						},
					},
				},
			},
		},
		nil,
		nil,
	)
}

var _ = Describe("Action", func() {
	const (
		noInputError         = "Action does not take input"
		noPropertiesError    = "Input schema does not have properties"
		noInputTypeError     = "Input schema does not have a type"
		notFoundError        = "Property with ID %s not found"
		noParameterTypeError = "Parameter with ID %s does not have a type"
	)

	var (
		empty                 Action
		noInputType           Action
		invalidInputType      Action
		invalidParametersType Action
		invalidParameters     Action
		valid                 Action
	)

	BeforeEach(func() {
		empty = emptyAction()
		noInputType = noInputTypeAction()
		invalidInputType = invalidInputTypeAction()
		invalidParametersType = invalidPropertiesType()
		invalidParameters = invalidProperties()
		valid = validAction()
	})

	Describe("id in Path", func() {
		It("Should check if action takes an id a as parameter", func() {
			Expect(empty.TakesID()).To(BeFalse())
			Expect(noInputType.TakesID()).To(BeTrue())
			Expect(invalidInputType.TakesID()).To(BeFalse())
			Expect(invalidParametersType.TakesID()).To(BeFalse())
			Expect(invalidParameters.TakesID()).To(BeFalse())
			Expect(valid.TakesID()).To(BeTrue())
		})
	})

	Describe("input type", func() {
		It("Should show error for action with no input", func() {
			_, err := empty.GetInputType()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noInputError))
		})

		It("Should show error for action with no input type", func() {
			_, err := noInputType.GetInputType()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noInputTypeError))
		})

		It("Should show error for action with invalid input type", func() {
			_, err := invalidInputType.GetInputType()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noInputTypeError))
		})

		It("Should show correct type for action with valid type", func() {
			result, err := invalidParameters.GetInputType()
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(invalidParameters.InputSchema["type"]))
		})
	})

	Describe("input parameter names", func() {
		It("Should show error for action with no input", func() {
			_, err := empty.GetInputParameterNames()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noInputError))
		})

		It("Should show error for action with no properties", func() {
			_, err := invalidInputType.GetInputParameterNames()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noPropertiesError))
		})

		It("Should show error for action with invalid properties type", func() {
			_, err := invalidParametersType.GetInputParameterNames()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noPropertiesError))
		})

		It("Should get input parameter names for valid action", func() {
			result, err := valid.GetInputParameterNames()
			Expect(err).ToNot(HaveOccurred())
			keys, _ := getData(valid.InputSchema["properties"].(map[string]interface{}))
			sort.Strings(result)
			sort.Strings(keys)
			Expect(result).To(Equal(keys))
		})
	})

	Describe("input parameter types", func() {
		It("Should show error for action with no input", func() {
			_, err := empty.GetInputParameterType("a")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noInputError))
		})

		It("Should show error for action with no properties", func() {
			_, err := invalidInputType.GetInputParameterType("a")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noPropertiesError))
		})

		It("Should show error for non existing parameter", func() {
			_, err := valid.GetInputParameterType("c")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(notFoundError, "c")))
		})

		It("Should show error for invalid parameter", func() {
			_, err := invalidParameters.GetInputParameterType("a")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(notFoundError, "a")))
		})

		It("Should show error for parameter with no type", func() {
			_, err := invalidParameters.GetInputParameterType("b")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(noParameterTypeError, "b")))
		})

		It("Should show error for parameter with invalid type", func() {
			_, err := invalidParameters.GetInputParameterType("c")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(noParameterTypeError, "c")))
		})

		It("Should get correct parameter type", func() {
			properties := valid.InputSchema["properties"].(map[string]interface{})
			result, err := valid.GetInputParameterType("a")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(properties["a"].(map[string]interface{})["type"]))
			result, err = valid.GetInputParameterType("b")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(properties["b"].(map[string]interface{})["type"]))
		})
	})
})
