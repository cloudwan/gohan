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
	"encoding/json"
	"fmt"

	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Schema", func() {
	Describe("Schema plural", func() {
		var (
			manager *Manager
			s       *Schema
			ok      bool
		)

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemasFromFiles(
				"../tests/test_schema_plural.yaml")).To(Succeed())

			s, ok = manager.Schema("box")
			Expect(ok).To(BeTrue())
		})

		It("should use schema.ID + \"s\" as table name when legacy is true", func() {
			Expect(s.GetDbTableName()).To(Equal(s.ID + "s"))
		})

		It("should use schema.Plural as table name when legacy is false", func() {
			config := util.GetConfig()
			config.ReadConfig("../tests/test_legacy_config.yaml")
			Expect(config.GetBool("database/legacy", true)).To(BeFalse())
			Expect(s.GetDbTableName()).To(Equal(s.Plural))
		})

		AfterEach(func() {
			ClearManager()
		})
	})

	Describe("Action", func() {
		var manager *Manager

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemasFromFiles("../tests/test_schema_action.yaml")).To(Succeed())
		})

		It("Should get nil for non existing action", func() {
			s, ok := manager.schema("action")
			Expect(ok).To(BeTrue())
			Expect(s.GetActionFromCommand("b")).To(BeNil())
		})

		It("Should get correct action", func() {
			s, ok := manager.schema("action")
			Expect(ok).To(BeTrue())
			Expect(s.GetActionFromCommand("a")).To(Equal(&s.Actions[0]))
		})

		AfterEach(func() {
			ClearManager()
		})
	})

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

	Describe("Properties access", func() {
		var manager *Manager
		var schema *Schema

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemasFromFiles(
				"../tests/test_schema_property_order.yaml")).To(Succeed())

			s, ok := manager.Schema("school")
			Expect(ok).To(BeTrue())
			schema = s
		})

		It("should access properties", func() {
			property, err := schema.GetPropertyByID("best_in_town")

			Expect(err).NotTo(HaveOccurred())
			Expect(property.ID).To(Equal("best_in_town"))
		})

		It("should err on access of undefined properties", func() {
			_, err := schema.GetPropertyByID("unknown")
			Expect(err).To(HaveOccurred())
		})

		It("should check property existence", func() {
			Expect(schema.HasPropertyID("patron")).To(BeTrue())
			Expect(schema.HasPropertyID("best_in_town")).To(BeTrue())

			Expect(schema.HasPropertyID("unknown")).To(BeFalse())
			Expect(schema.HasPropertyID("some_property_not_in_schema")).To(BeFalse())
		})
	})

	Describe("Nested properties", func() {
		var manager *Manager
		var schema *Schema

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemasFromFiles(
				"../tests/test_abstract_schema.yaml",
				"../tests/test_schema.yaml",
			)).To(Succeed())

			s, ok := manager.Schema("nested_attacher")
			Expect(ok).To(BeTrue())
			schema = s
		})

		It("should get all properties in the schema via GetAllPropertiesFullyQualifiedMap", func() {
			var propertyNames []string
			for name := range schema.GetAllPropertiesFullyQualifiedMap() {
				propertyNames = append(propertyNames, name)
			}

			Expect(propertyNames).To(ConsistOf(
				"id",
				"container_object",
				"container_object.attach_object_id",
				"container_array",
				"container_array.[].attach_array_id",
			))
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
			cityIDIdx := index(s.Properties, "city_id")
			patronIdx := index(s.Properties, "patron")
			Expect(cityIDIdx).ToNot(Equal(-1))
			Expect(patronIdx).ToNot(Equal(-1))
			Expect(cityIDIdx).Should(BeNumerically("<", patronIdx))
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

		AfterEach(func() {
			ClearManager()
		})
	})

	Describe("Order properties before", func() {
		var manager *Manager
		type JSONSchema struct {
			PropertiesOrder []string `json:"propertiesOrder"`
		}
		var jsonSchema *JSONSchema

		index := func(properties []string, id string) int {
			for i, property := range properties {
				if property == id {
					return i
				}
			}
			return -1
		}

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemaFromFile(
				"../tests/test_schema_order_properties_before.yaml")).To(Succeed())
			jsonSchema = &JSONSchema{}
			s, ok := manager.Schema("school")
			Expect(ok).To(BeTrue())
			js, _ := json.Marshal(s.JSONSchema)
			json.Unmarshal(js, jsonSchema)
		})

		It("should list all extends properties in PropertiesOrder", func() {
			idIdx := index(jsonSchema.PropertiesOrder, "id")
			nameIdx := index(jsonSchema.PropertiesOrder, "name")
			fundingIdx := index(jsonSchema.PropertiesOrder, "funding")
			fmt.Println(jsonSchema.PropertiesOrder, idIdx, nameIdx, fundingIdx)
			Expect(idIdx).ToNot(Equal(-1))
			Expect(nameIdx).ToNot(Equal(-1))
			Expect(fundingIdx).ToNot(Equal(-1))
		})

		It("should order properties before order_properties_before in PropertiesOrder", func() {
			nameIdx := index(jsonSchema.PropertiesOrder, "name")
			bestInTownIdx := index(jsonSchema.PropertiesOrder, "best_in_town")
			fundingIdx := index(jsonSchema.PropertiesOrder, "funding")
			Expect(nameIdx).ToNot(Equal(-1))
			Expect(bestInTownIdx).ToNot(Equal(-1))
			Expect(fundingIdx).ToNot(Equal(-1))
			Expect(nameIdx).Should(BeNumerically("<", bestInTownIdx))
			Expect(bestInTownIdx).Should(BeNumerically("<", fundingIdx))
		})

		AfterEach(func() {
			ClearManager()
		})
	})

	Describe("Indexes", func() {
		var todosSchema *Schema
		BeforeEach(func() {
			var exists bool
			manager := GetManager()
			schemaPath := "../tests/test_schema_indexes.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			todosSchema, exists = manager.Schema("todo")
			Expect(exists).To(BeTrue())
		})

		It("Parse indexes", func() {
			for _, index := range todosSchema.Indexes {
				if index.Type == Unique {
					Expect(index.Columns).To(Equal([]string{"m1", "m2"}))
					Expect(index.Name).To(Equal("unique_m1_m2"))
				}
				if index.Type == Spatial {
					Expect(index.Columns).To(Equal([]string{"m2", "m3"}))
					Expect(index.Name).To(Equal("spatial_m2_m3"))
				}
				if index.Type == FullText {
					Expect(index.Columns).To(Equal([]string{"m1", "m3"}))
					Expect(index.Name).To(Equal("fulltext_m1_m3"))
				}
				if index.Type == None {
					Expect(index.Columns).To(Equal([]string{"m1", "m2", "m3"}))
					Expect(index.Name).To(Equal("emptyType_m1_m2_m3"))
				}
			}
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

	Describe("IsolationLevel", func() {
		var netSchema *Schema

		BeforeEach(func() {
			var exists bool
			manager := GetManager()
			basePath := "../tests/test_abstract_schema.yaml"
			Expect(manager.LoadSchemaFromFile(basePath)).To(Succeed())

			schemaPath := "../tests/test_schema.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			netSchema, exists = manager.Schema("network")
			Expect(exists).To(BeTrue())
		})

		It("Direct Setting", func() {
			Expect(netSchema.IsolationLevel["read"]).To(Equal("REPEATABLE READ"))
		})

		It("Inherit Base", func() {
			Expect(netSchema.IsolationLevel["delete"]).To(Equal("READ COMMITTED"))
		})

		It("Override Base", func() {
			Expect(netSchema.IsolationLevel["update"]).To(Equal("SERIALIZABLE"))
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

		It("Version", func() {
			netMap := map[string]interface{}{"version": "1.2,3"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(MatchError(getErrorMessage("version", "version")))

			netMap = map[string]interface{}{"version": "1.2.3"}
			Expect(netSchema.ValidateOnCreate(netMap)).To(Succeed())
		})
	})

	It("should ignore empty schema file", func() {
		manager := GetManager()
		Expect(manager.LoadSchemasFromFiles("")).To(Succeed())
	})
})

func getErrorMessage(fieldName string, formatterName string) string {
	return fmt.Sprintf("Json validation error:\n\t%s: Does not match format '%s',", fieldName, formatterName)
}
