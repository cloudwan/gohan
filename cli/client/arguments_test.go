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
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"strings"
)

const (
	parsingError = "Error parsing parameter '%v': %v"
)

func getBoolAction() schema.Action {
	return schema.NewAction(
		"BoolAction",
		"GET",
		"/bool/",
		"action with input type bool",
		map[string]interface{}{
			"type": "boolean",
		},
		nil,
		nil,
	)
}

func getObjectAction() schema.Action {
	return schema.NewAction(
		"ObjectAction",
		"GET",
		"/object/",
		"action with input type object",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{
					"type": "number",
				},
				"b": map[string]interface{}{
					"type": "boolean",
				},
			},
		},
		nil,
		nil,
	)
}

var _ = Describe("Arguments", func() {
	Describe("Parsing raw value", func() {
		It("Should parse null", func() {
			value, err := getValueFromRaw("", "<null>", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(BeNil())
		})

		It("Should show error for invalid integer", func() {
			key := "Key"
			_, err := getValueFromRaw(key, "a", "number")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(parsingError, key, "strconv.ParseInt: parsing \"a\": invalid syntax")))
		})

		It("Should parse valid integer", func() {
			value, err := getValueFromRaw("", "1", "number")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(int64(1)))
		})

		It("Should show error for invalid boolean", func() {
			key := "Key"
			_, err := getValueFromRaw(key, "a", "boolean")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(parsingError, key, "strconv.ParseBool: parsing \"a\": invalid syntax")))
		})

		It("Shoud parse valid boolean", func() {
			value, err := getValueFromRaw("", "true", "boolean")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(BeTrue())
		})

		It("Should show error for invalid json", func() {
			key := "Key"
			_, err := getValueFromRaw(key, "a", "object")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf(parsingError, key, "invalid character 'a' looking for beginning of value")))
		})

		It("Should parse valid json", func() {
			var expected = map[string]interface{}{
				"a": 1.0,
				"b": true,
			}
			data, err := getValueFromRaw("", "{\"a\": 1, \"b\": true}", "object")
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal(expected))
		})
	})

	Describe("Tests with gohan client", func() {
		var gohanClientCLI GohanClientCLI

		BeforeEach(func() {
			gohanClientCLI = GohanClientCLI{opts: &GohanClientCLIOpts{}}
		})

		Describe("Handling common arguments", func() {
			const (
				outputFormatKey = "output-format"
				logLevelKey     = "verbosity"
				fieldsKey       = "fields"
			)

			var (
				outputFormats = []string{"table", "json"}
				logLevels     = []l.Level{
					l.WARNING,
					l.NOTICE,
					l.INFO,
					l.DEBUG,
				}
			)

			It("Should show output format error", func() {
				args := map[string]interface{}{outputFormatKey: "number"}
				err := gohanClientCLI.handleCommonArguments(args)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf(incorrectOutputFormat, outputFormats)))
			})

			It("Should show log level error", func() {
				args := map[string]interface{}{logLevelKey: "a"}
				err := gohanClientCLI.handleCommonArguments(args)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf(incorrectVerbosityLevel, 0, len(logLevels)-1)))
			})

			It("Should show handle common args", func() {
				fields := "a,b,c"
				args := map[string]interface{}{
					outputFormatKey: "table",
					logLevelKey:     "0",
					fieldsKey:       fields,
				}
				err := gohanClientCLI.handleCommonArguments(args)
				Expect(err).ToNot(HaveOccurred())
				Expect(gohanClientCLI.opts.outputFormat).To(Equal("table"))
				Expect(args[outputFormatKey]).To(BeNil())
				Expect(gohanClientCLI.opts.logLevel).To(Equal(l.WARNING))
				Expect(args[logLevelKey]).To(BeNil())
				Expect(gohanClientCLI.opts.fields).To(Equal(strings.Split(fields, ",")))
				Expect(args[fieldsKey]).To(BeNil())
			})

			Describe("Getting arguments", func() {
				const invalidFormat = "Parameters should be in [--param-name value]... format"

				var schema schema.Schema = schema.Schema{Properties: []schema.Property{
					{ID: "a", Type: "boolean"},
					{ID: "b", Type: "number"},
				}}

				It("Shoud show error for invalid parameters format", func() {
					_, err := getArgsAsMap([]string{"a", "b", "c"}, &schema)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(invalidFormat))
				})

				It("Shoud show parsing error", func() {
					_, err := getArgsAsMap([]string{"--a", "a"}, &schema)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf(parsingError, "a", "strconv.ParseBool: parsing \"a\": invalid syntax")))
				})

				It("Should get correct arguments map", func() {
					var text = "[\"a\", \"b\"]"
					data, err := getArgsAsMap([]string{"--a", "false", "--b", "2", "--c", "<null>", "--d", text}, &schema)
					Expect(err).ToNot(HaveOccurred())
					Expect(data["a"]).To(BeFalse())
					Expect(data["b"]).To(Equal(int64(2)))
					Expect(data["c"]).To(BeNil())
					Expect(data["d"]).To(Equal(text))
				})
			})

			Describe("Getting custom arguments", func() {

				It("Should show error for incorrect input type", func() {
					_, err := gohanClientCLI.getCustomArgsAsMap(nil, []string{"a"}, getBoolAction())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf(parsingError, "input", "strconv.ParseBool: parsing \"a\": invalid syntax")))
				})

				It("Should get arguments map with correct input", func() {
					action := getBoolAction()
					data, err := gohanClientCLI.getCustomArgsAsMap(nil, []string{"false"}, action)
					Expect(err).ToNot(HaveOccurred())
					Expect(data[action.ID]).To(BeFalse())
				})

				It("Shoud show error for many input arguments when type is not object", func() {
					_, err := gohanClientCLI.getCustomArgsAsMap(nil, []string{"true", "false"}, getBoolAction())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("Input schema type must be an object to pass input with parameters"))
				})

				It("Should show error for non existing parameter", func() {
					_, err := gohanClientCLI.getCustomArgsAsMap(nil, []string{"--c", "false"}, getObjectAction())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("Property with ID c not found"))
				})

				It("Should show error for invalid parameter value", func() {
					_, err := gohanClientCLI.getCustomArgsAsMap(nil, []string{"--a", "b"}, getObjectAction())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf(parsingError, "a", "strconv.ParseInt: parsing \"b\": invalid syntax")))
				})

				It("Should show error for invalid common parameter", func() {
					_, err := gohanClientCLI.getCustomArgsAsMap([]string{"--a", "b"}, []string{"false"}, getBoolAction())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("Error parsing parameter a"))
				})

				Describe("Correct input", func() {
					var (
						expected = map[string]interface{}{
							"a": int64(1),
							"b": true,
						}
						action = getObjectAction()
					)

					It("Should get argumets map with correct input - many parameters", func() {
						data, err := gohanClientCLI.getCustomArgsAsMap(
							[]string{"--output-format", "table"},
							[]string{"--a", "1", "--b", "true"},
							action)
						Expect(err).ToNot(HaveOccurred())
						Expect(data[action.ID]).To(Equal(expected))
						Expect(data["output-format"]).To(BeNil())
						Expect(gohanClientCLI.opts.outputFormat).To(Equal("table"))
					})

					It("Should get arguments map with correct input - one parameter", func() {
						data, err := gohanClientCLI.getCustomArgsAsMap(
							[]string{"--output-format", "table"},
							[]string{"{\"a\": 1, \"b\": true}"},
							action)
						Expect(err).ToNot(HaveOccurred())
						expected["a"] = 1.0
						Expect(data[action.ID]).To(Equal(expected))
						Expect(gohanClientCLI.opts.outputFormat).To(Equal("table"))
					})
				})
			})
		})
	})
})
