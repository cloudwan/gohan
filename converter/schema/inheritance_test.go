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

	"github.com/cloudwan/gohan/converter/set"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("inheritance tests", func() {
	var (
		getObject = func(base, name, itemType string) map[interface{}]interface{} {
			return map[interface{}]interface{}{
				"id":     name,
				"parent": "p" + name,
				"schema": map[interface{}]interface{}{
					"type": "object",
					"properties": map[interface{}]interface{}{
						base: map[interface{}]interface{}{
							"type": itemType,
						},
						name: map[interface{}]interface{}{
							"type": "string",
						},
					},
				},
			}
		}

		createFromObject = func(object map[interface{}]interface{}) *Schema {
			objectSchema := &Schema{}

			err := objectSchema.parse(object)

			Expect(err).ToNot(HaveOccurred())
			return objectSchema
		}
	)

	Describe("collect tests", func() {
		It("Should return error for multiple schemas with the same name", func() {
			objectSchema := createFromObject(getObject("a", "b", "string"))
			other := createFromObject(getObject("a", "b", "number"))

			Expect(objectSchema).ToNot(Equal(other))

			toConvert := set.New()
			toConvert.Insert(objectSchema)
			otherSet := set.New()
			otherSet.Insert(other)

			err := collectSchemas(toConvert, otherSet)

			expected := fmt.Errorf("multiple schemas with the same name")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return error for non existing base schema", func() {
			id := "b"
			object := getObject("a", "a", "string")
			object["extends"] = []interface{}{id}
			toConvert := set.New()
			toConvert.Insert(createFromObject(object))

			err := collectSchemas(toConvert, set.New())

			expected := fmt.Errorf(
				"schema with id %s does not exist",
				id,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return cyclic dependency error", func() {
			objects := make([]map[interface{}]interface{}, 3)

			objects[0] = getObject("a", "a", "string")
			objects[0]["extends"] = []interface{}{"b"}

			objects[1] = getObject("b", "b", "string")
			objects[1]["extends"] = []interface{}{"c"}

			objects[2] = getObject("c", "c", "string")
			objects[2]["extends"] = []interface{}{"a"}

			toConvert := set.New()
			for _, object := range objects {
				toConvert.Insert(createFromObject(object))
			}

			err := collectSchemas(toConvert, set.New())

			expected := fmt.Errorf("cyclic dependencies detected")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should return join error", func() {
			id := "a"
			base := "base"
			objects := make([]map[interface{}]interface{}, 3)

			objects[0] = getObject(id, id, "string")
			objects[0]["extends"] = []interface{}{"b", "c"}

			objects[1] = getObject(base, "b", "string")

			objects[2] = getObject(base, "c", "string")

			toConvert := set.New()
			for _, object := range objects {
				toConvert.Insert(createFromObject(object))
			}

			err := collectSchemas(toConvert, set.New())

			expected := fmt.Errorf(
				"multiple properties with the same name in bases of schema %s",
				id,
			)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		})

		It("Should correctly update schemas", func() {
			objects := make([]map[interface{}]interface{}, 3)

			objects[0] = getObject("a", "a", "string")
			objects[0]["extends"] = []interface{}{"b", "c"}

			objects[1] = getObject("b", "b", "string")
			objects[1]["extends"] = []interface{}{"c"}

			objects[2] = getObject("c", "c", "string")

			toConvert := set.New()
			for _, object := range objects {
				toConvert.Insert(createFromObject(object))
			}

			err := collectSchemas(toConvert, set.New())

			Expect(err).ToNot(HaveOccurred())
			array := toConvert.ToArray()
			for i, arraySchema := range array {
				newSet, err := arraySchema.(*Schema).collectProperties(-1, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(newSet.Size()).To(Equal(2 * (3 - i)))
			}
		})
	})
})
