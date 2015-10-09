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

package util

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util functions", func() {

	Describe("SaveFile and LoadFile", func() {
		It("should handle json file properly", func() {
			testFileUtil("./util_test.json")
		})

		It("should handle yaml file properly", func() {
			testFileUtil("./util_test.yaml")
		})
	})

	Describe("TempFile", func() {
		It("should create temporary file properly", func() {
			file, err := TempFile("./", "util_", "_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(file).ToNot(BeNil())
			_, err = os.Stat(file.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(os.Remove(file.Name())).To(Succeed())
		})
	})

	Describe("GetContents", func() {
		It("should get contents form http", func() {
			_, err := GetContent("http://www.google.com")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should get contents form file", func() {
			_, err := GetContent("file://../tests/test_schema.yaml")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("GetSortedKeys", func() {
		It("should return sorted map[string]interface{} keys", func() {
			input := map[string]interface{}{
				"Lorem": "v1",
				"ipsum": "v2",
				"dolor": "v3",
				"sit":   "v4",
			}
			result := GetSortedKeys(input)
			Expect(result).To(Equal([]string{"dolor", "ipsum", "Lorem", "sit"}))
		})
	})
})

func testFileUtil(file string) {
	os.Remove(file)
	data := map[string]interface{}{
		"test": "data",
	}
	Expect(SaveFile(file, data)).To(Succeed())
	defer os.Remove(file)
	loadedData, err := LoadFile(file)
	Expect(err).ToNot(HaveOccurred())
	Expect(loadedData["test"]).To(Equal(data["test"]))
}
