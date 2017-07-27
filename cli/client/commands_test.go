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

package client

import (
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func idAction() schema.Action {
	return schema.NewAction(
		"IdAction",
		"GET",
		"/id/:id/",
		"action with id",
		nil,
		nil,
		nil,
	)
}

func inputAction() schema.Action {
	return schema.NewAction(
		"InputAction",
		"GET",
		"/input/",
		"action with input",
		map[string]interface{}{
			"type": "boolean",
		},
		nil,
		nil,
	)
}

func idInputAction() schema.Action {
	return schema.NewAction(
		"IdInputAction",
		"GET",
		"/input/:id/",
		"action with input and id",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{
					"type": "boolean",
				},
				"b": map[string]interface{}{
					"type": "number",
				},
			},
		},
		nil,
		nil,
	)
}

var _ = Describe("Commands", func() {
	Describe("Split arguments", func() {
		const (
			argumentsError    = "Wrong number of arguments"
			noPropertiesError = "Input schema does not have properties"
		)

		var (
			idAction      = idAction()
			inputAction   = inputAction()
			idInputAction = idInputAction()
		)

		It("Should show error for wrong number of arguments - id", func() {
			_, _, _, err := splitArgs([]string{}, &idAction)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(argumentsError))
		})

		It("Should show error for wrong number of arguments - input", func() {
			_, _, _, err := splitArgs([]string{}, &inputAction)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(argumentsError))
		})

		It("Should show error for wrong number of arguments - id and input", func() {
			_, _, _, err := splitArgs([]string{}, &idInputAction)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(argumentsError))
		})

		It("Should show error for input with no properties", func() {
			_, _, _, err := splitArgs([]string{"a", "b"}, &inputAction)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(noPropertiesError))
		})

		It("Should get correct arguments - multiple arguments", func() {
			args, input, id, err := splitArgs([]string{"1", "2", "3", "4", "5", "6", "7"}, &idInputAction)
			Expect(err).ToNot(HaveOccurred())
			Expect(args).To(Equal([]string{"1", "2"}))
			Expect(input).To(Equal([]string{"3", "4", "5", "6"}))
			Expect(id).To(Equal("7"))
		})

		It("Should get correct arguments - one argument", func() {
			args, input, id, err := splitArgs([]string{"1", "2"}, &idInputAction)
			Expect(err).ToNot(HaveOccurred())
			Expect(args).To(Equal([]string{}))
			Expect(input).To(Equal([]string{"1"}))
			Expect(id).To(Equal("2"))
		})
	})
})
