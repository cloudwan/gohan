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
	Describe("Schema manager", func() {
		It("should reorder schemas if it is DAG", func() {
			manager := GetManager()
			schemaPath := "../tests/test_schema_dag_dependency.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			Expect(manager.schemaOrder).To(Equal(
				[]string{
					"base_resource",
					"common_resource",
					"red_resource",
					"blue_resource",
					"green_resource",
					"orange_resource"}))
		})

		It("should read schemas even if it isn't DAG", func() {
			manager := GetManager()
			schemaPath := "../tests/test_schema_non_dag_dependency.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
		})

		AfterEach(func() {
			ClearManager()
		})
	})

	Describe("Properties order", func() {
		var manager *Manager

		index := func(properties []Property, id string) int {
			for i, property := range properties {
				if property.ID == id {
					return i
				}
			}
			return -1
		}

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemasFromFiles(
				"../tests/test_schema_property_order.yaml")).To(Succeed())

		})

		It("PropertiesOrder first", func() {
			s, ok := manager.Schema("school")
			Expect(ok).To(BeTrue())
			idIdx := index(s.Properties, "id")
			nameIdx := index(s.Properties, "name")
			patronIdx := index(s.Properties, "patron")
			Expect(idIdx).ToNot(Equal(-1))
			Expect(nameIdx).ToNot(Equal(-1))
			Expect(patronIdx).ToNot(Equal(-1))
			Expect(idIdx).Should(BeNumerically("<", nameIdx))
			Expect(nameIdx).Should(BeNumerically("<", patronIdx))
		})

		It("Relations after propertiesOrder", func() {
			s, ok := manager.Schema("school")
			Expect(ok).To(BeTrue())
			cityIdIdx := index(s.Properties, "city_id")
			patronIdx := index(s.Properties, "patron")
			Expect(cityIdIdx).ToNot(Equal(-1))
			Expect(patronIdx).ToNot(Equal(-1))
			Expect(cityIdIdx).Should(BeNumerically("<", patronIdx))
		})

		It("Alphabetical order", func() {
			s, ok := manager.Schema("school")
			Expect(ok).To(BeTrue())
			bestInTownIdx := index(s.Properties, "best_in_town")
			patronIdx := index(s.Properties, "patron")
			Expect(bestInTownIdx).ToNot(Equal(-1))
			Expect(patronIdx).ToNot(Equal(-1))
			Expect(bestInTownIdx).Should(BeNumerically("<", patronIdx))
		})
	})

	Describe("Metadata", func() {
		var metadataSchema *Schema
		var metadataFailedSchema *Schema
		var metadataIDSchema *Schema
		var metadataPolicySchema *Schema

		BeforeEach(func() {
			var exists bool
			var failedExists bool
			var idExists bool
			var policyExists bool
			manager := GetManager()
			schemaPath := "../tests/test_schema_metadata.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			metadataSchema, exists = manager.Schema("metadata")
			metadataFailedSchema, failedExists = manager.Schema("metadata_failed")
			metadataIDSchema, idExists = manager.Schema("metadata_id")
			metadataPolicySchema, policyExists = manager.Schema("metadata_policy")
			Expect(exists).To(BeTrue())
			Expect(failedExists).To(BeTrue())
			Expect(idExists).To(BeTrue())
			Expect(policyExists).To(BeTrue())
		})

		It("SyncKeyTemplate", func() {
			syncKeyTemplate, ok := metadataSchema.SyncKeyTemplate()
			Expect(ok).To(BeTrue())
			Expect(syncKeyTemplate).To(Equal("/v1.0/metadata/{{m1}}/{{m2}}/{{m3}}"))
		})

		It("GenerateCustomPath", func() {
			data := map[string]interface{}{
				"m1": "mm1",
				"m2": "true",
				"m3": "3",
			}
			path, err := metadataSchema.GenerateCustomPath(data)
			Expect(err).To(Succeed())
			Expect(path).To(Equal("/v1.0/metadata/mm1/true/3"))
		})

		It("GenerateCustomPathDoesntFail", func() {
			data := map[string]interface{}{
				"m1": "mm1",
				"m2": "true",
				"m3": "3",
			}
			path, err := metadataFailedSchema.GenerateCustomPath(data)
			Expect(err).To(Succeed())
			Expect(path).To(Equal("/v1.0/metadata-failed/mm1/"))
		})

		It("MetadataId", func() {
			data := map[string]interface{}{
				"id": "1234",
			}
			path, err := metadataIDSchema.GenerateCustomPath(data)
			Expect(err).To(Succeed())
			Expect(path).To(Equal("/v1.0/metadata-id/1234/"))
			str := metadataIDSchema.GetResourceIDFromPath(path)
			Expect(str).To(Equal("1234"))
		})

		It("Should use non locking policy when unspecified", func() {
			Expect(metadataIDSchema.GetLockingPolicy("update")).To(Equal(NoLocking))
		})

		It("Should return locking policies", func() {
			Expect(metadataPolicySchema.GetLockingPolicy("update")).To(Equal(LockRelatedResources))
			Expect(metadataPolicySchema.GetLockingPolicy("delete")).To(Equal(SkipRelatedResources))
		})

		AfterEach(func() {
			ClearManager()
		})
	})

	Describe("SchemaPaths", func() {
		var metadataIDSchema *Schema

		BeforeEach(func() {
			var idExists bool
			manager := GetManager()
			schemaPath := "../tests/test_schema_metadata.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			metadataIDSchema, idExists = manager.Schema("metadata_id")
			Expect(idExists).To(BeTrue())
		})

		It("GetSchemaByTemplatePath", func() {
			data := map[string]interface{}{
				"id": "1234",
			}
			path, err := metadataIDSchema.GenerateCustomPath(data)
			Expect(err).To(Succeed())
			Expect(GetSchemaByPath(path)).To(Equal(metadataIDSchema))
		})

		It("GetSchemaByUrl", func() {
			Expect(metadataIDSchema).To(Equal(GetSchemaByURLPath("/metadatas_id")))
		})

		AfterEach(func() {
			ClearManager()
		})
	})

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
