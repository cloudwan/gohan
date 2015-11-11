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
)

var _ = Describe("Schema", func() {
	Describe("Formatters", func() {
		var netSchema *Schema

		BeforeEach(func() {
			var exists bool
			manager := GetManager()
			schemaPath := "../tests/test_schema.json"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			netSchema, exists = manager.Schema("net")
			Expect(exists).To(BeTrue())
		})

		AfterEach(func() {
			ClearManager()
		})

		It("CIDR", func() {
			netMap := map[string]interface{}{"cidr": "asdf"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("cidr", "cidr")))

			netMap = map[string]interface{}{"cidr": "10.10.10.10/24"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())

			netMap = map[string]interface{}{"cidr": "127.0.0.1"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("cidr", "cidr")))
		})

		It("MAC", func() {
			netMap := map[string]interface{}{"mac": "aa:bb:cc:dd:ee"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("mac", "mac")))

			netMap = map[string]interface{}{"mac": "aa-aa-aa-aa-aa-aa"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("mac", "mac")))

			netMap = map[string]interface{}{"mac": "aa:bb:cc:dd:ee:ff"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())

			netMap = map[string]interface{}{"mac": "11:22:33:DD:1e:FF"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())
		})

		It("UUID", func() {
			netMap := map[string]interface{}{"id": "wrong-id"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("id", "uuid")))

			netMap = map[string]interface{}{"id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("id", "uuid")))

			netMap = map[string]interface{}{"id": "de305d54-75b4-431b-adb2-eb6b9e546014"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())

			netMap = map[string]interface{}{"id": "de305d5475b4431badb2eb6b9e546014"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())
		})

		It("Port", func() {
			netMap := map[string]interface{}{"port": "wrong-port"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("port", "port")))

			netMap = map[string]interface{}{"port": "-1"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("port", "port")))

			netMap = map[string]interface{}{"port": "0"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("port", "port")))

			netMap = map[string]interface{}{"port": "65536"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("port", "port")))

			netMap = map[string]interface{}{"port": "42"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())
		})

		It("Regex", func() {
			netMap := map[string]interface{}{"regex": "[[[{{{"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("regex", "regex")))

			netMap = map[string]interface{}{"regex": "[a-z0-7]{3}.*[,.;']{1,2}"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())
		})
	})
})

func getErrorMessage(fieldName string, formatterName string) string {
	return fmt.Sprintf("Json validation error:\n\t%s: Does not match format '%s',", fieldName, formatterName)
}
