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

package reader

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("reader tests", func() {
	const path = "../tests/"

	Describe("get Schemas tests", func() {
		It("Should return error for invalid schema", func() {
			filename := "test"
			data := []interface{}{1, 2}

			_, err := getSchemas(filename, data)

			expected := fmt.Errorf(
				"error in file %s: schema should have type map[interface{}]interface{}",
				filename,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return correct schemas", func() {
			first := map[interface{}]interface{}{
				"a": "a",
				"b": "b",
			}
			second := map[interface{}]interface{}{
				"c": "c",
				"d": "d",
			}
			data := []interface{}{first, second}

			result, err := getSchemas("", data)

			expected := []map[interface{}]interface{}{first, second}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Describe("read single tests", func() {
		It("Should return error when failed to read a file", func() {
			filename := path + "NonExistingFile.yaml"

			_, err := ReadSingle(filename)

			expected := fmt.Errorf(
				"failed to open file %s",
				filename,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for invalid yaml", func() {
			filename := path + "invalid.yaml"

			_, err := ReadSingle(filename)

			expected := fmt.Errorf(
				"cannot parse given schema from file %s",
				filename,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error when file contains no schemas", func() {
			filename := path + "no_schemas.yaml"

			_, err := ReadSingle(filename)

			expected := fmt.Errorf(
				"no schemas found in file %s",
				filename,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return correct schemas", func() {
			filename := path + "only_names.yaml"
			first := map[interface{}]interface{}{
				"a": "a",
				"b": "b",
			}
			second := map[interface{}]interface{}{
				"c": "c",
				"d": "d",
			}

			result, err := ReadSingle(filename)

			expected := []map[interface{}]interface{}{first, second}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Describe("read all tests", func() {
		It("Should return error when file contains no schemas", func() {
			filename := path + "no_schemas.yaml"

			_, err := ReadAll(filename, "")

			expected := fmt.Errorf(
				"no schemas found in file %s",
				filename,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for file with types other than string", func() {
			filename := path + "no_string_names.yaml"

			_, err := ReadAll(filename, "")

			expected := fmt.Errorf(
				"in config file %s schemas should be filenames",
				filename,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should ignore files with no schemas", func() {
			filename := path + "invalid_file_config.yaml"

			result, err := ReadAll(filename, "")

			expected := []map[interface{}]interface{}{}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("Should return correct schemas", func() {
			filename := path + "only_names_config.yaml"
			first := map[interface{}]interface{}{
				"a": "a",
				"b": "b",
			}
			second := map[interface{}]interface{}{
				"c": "c",
				"d": "d",
			}
			third := map[interface{}]interface{}{
				"e": "e",
				"f": "f",
			}
			fourth := map[interface{}]interface{}{
				"g": "g",
				"h": "h",
			}

			result, err := ReadAll(filename, "")

			expected := []map[interface{}]interface{}{first, second, third, fourth}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})

		It("Should ignore restricted file", func() {
			filename := path + "only_names_config.yaml"
			third := map[interface{}]interface{}{
				"e": "e",
				"f": "f",
			}
			fourth := map[interface{}]interface{}{
				"g": "g",
				"h": "h",
			}

			result, err := ReadAll(filename, path+"only_names.yaml")

			expected := []map[interface{}]interface{}{third, fourth}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})
})
