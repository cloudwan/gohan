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

package cli

import (
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Templates", func() {
	var (
		manager  *schema.Manager
		schemas  []*schema.Schema
		policies []*schema.Policy
	)

	BeforeEach(func() {
		manager = schema.GetManager()
		schemaPath := "../tests/test_schema.json"
		Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
		schemas = manager.OrderedSchemas()
		Expect(schemas).To(HaveLen(3))
		policies = manager.Policies()
	})

	AfterEach(func() {
		schema.ClearManager()
	})

	Describe("Filtering schemas for specific policy", func() {
		allPermissions := []string{"create", "read", "update", "delete"}

		shouldReturnAdminSchemas := func(filteredSchemas []*SchemaWithPolicy) {
			Expect(filteredSchemas).To(HaveLen(3))
			Expect(filteredSchemas[0].Schema.URL).To(Equal("/v2.0/nets"))
			Expect(filteredSchemas[0].Policies).To(Equal(allPermissions))
			Expect(filteredSchemas[1].Schema.URL).To(Equal("/v2.0/networks"))
			Expect(filteredSchemas[1].Policies).To(Equal(allPermissions))
			Expect(filteredSchemas[2].Schema.URL).To(Equal("/v2.0/network/:network/subnets"))
			Expect(filteredSchemas[2].Policies).To(Equal(allPermissions))
		}

		shouldReturnMemberSchemas := func(filteredSchemas []*SchemaWithPolicy) {
			Expect(filteredSchemas).To(HaveLen(2))
			Expect(filteredSchemas[0].Schema.URL).To(Equal("/v2.0/nets"))
			Expect(filteredSchemas[0].Policies).To(Equal([]string{"create", "read", "delete"}))
			Expect(filteredSchemas[1].Schema.URL).To(Equal("/v2.0/networks"))
			Expect(filteredSchemas[1].Policies).To(Equal([]string{"create", "update"}))
		}

		shouldReturnDomainAdminSchemas := func(filteredSchemas []*SchemaWithPolicy) {
			Expect(filteredSchemas).To(HaveLen(1))
			Expect(filteredSchemas[0].Schema.URL).To(Equal("/v2.0/network/:network/subnets"))
			Expect(filteredSchemas[0].Policies).To(Equal([]string{"read"}))
		}

		Context("With policy set to admin", func() {
			It("should return only admin's schemas", func() {
				filteredSchemas := filterSchemasForPolicy("admin", schema.AllTokenTypes, policies, schemas)
				shouldReturnAdminSchemas(filteredSchemas)
			})
		})

		Context("With empty policy", func() {
			It("should return all schemas", func() {
				filteredSchemas := filterSchemasForPolicy("", schema.AllTokenTypes, policies, schemas)
				shouldReturnAdminSchemas(filteredSchemas)
			})
		})

		Context("With policy set to member", func() {
			It("should return only member's schemas", func() {
				filteredSchemas := filterSchemasForPolicy("Member", schema.AllTokenTypes, policies, schemas)
				shouldReturnMemberSchemas(filteredSchemas)
			})
		})

		Context("With policy set to nobody", func() {
			It("should return only nobody's schemas", func() {
				filteredSchemas := filterSchemasForPolicy("Nobody", schema.AllTokenTypes, policies, schemas)
				Expect(filteredSchemas).To(BeEmpty())
			})
		})

		Context("With scope set to [domain]", func() {
			It("should return only domain admin's schemas", func() {
				domainAdminScope := []schema.Scope{schema.DomainScope}
				filteredSchemas := filterSchemasForPolicy("admin", domainAdminScope, policies, schemas)
				shouldReturnDomainAdminSchemas(filteredSchemas)
			})
		})
	})

	Describe("Filtering schemas for specific resource", func() {
		var schemaWithPolicy []*SchemaWithPolicy
		BeforeEach(func() {
			schemaWithPolicy = filterSchemasForPolicy("", schema.AllTokenTypes, policies, schemas)
		})
		Context("With resource set to a", func() {
			It("should return only a schemas", func() {

				filteredSchemas := filerSchemasByResourceGroup("a", schemaWithPolicy)

				Expect(filteredSchemas).To(HaveLen(2))
				Expect(filteredSchemas[0].Schema.URL).To(Equal("/v2.0/nets"))
				Expect(filteredSchemas[1].Schema.URL).To(Equal("/v2.0/networks"))
			})
		})

		Context("With resource set to b", func() {
			It("should return only b schemas", func() {

				filteredSchemas := filerSchemasByResourceGroup("b", schemaWithPolicy)

				Expect(filteredSchemas).To(HaveLen(1))
				Expect(filteredSchemas[0].Schema.URL).To(Equal("/v2.0/network/:network/subnets"))
			})
		})

		Context("With resource set to c", func() {
			It("should not return any schemas", func() {

				filteredSchemas := filerSchemasByResourceGroup("c", schemaWithPolicy)

				Expect(filteredSchemas).To(BeEmpty())
			})
		})

		Context("With schema containing 2 resources", func() {
			It("should recognize correctly all of them", func() {

				resources := getAllResourceGroupsFromSchemas(schemaWithPolicy)

				Expect(resources).To(HaveLen(2))
				Expect(resources).To(ContainElement("a"))
				Expect(resources).To(ContainElement("b"))
			})
		})
	})

	Describe("Filtering properties for specific resource", func() {
		It("should return properties written in the policy", func() {
			role := "Member"
			filteredSchemas := filterSchemasForPolicy(role, schema.AllTokenTypes, policies, schemas)
			calculateAllowedProperties(filteredSchemas, role, manager)

			networkSchema := filteredSchemas[1]
			Expect(networkSchema.JSONSchemaOnCreate).ToNot(BeNil())
			Expect(networkSchema.JSONSchemaOnCreate).To(HaveKey("properties"))

			JSONProperties := networkSchema.JSONSchemaOnCreate["properties"].(map[string]interface{})
			Expect(JSONProperties).To(HaveLen(3))
			Expect(JSONProperties).To(HaveKey("id"))
			Expect(JSONProperties).To(HaveKey("description"))
			Expect(JSONProperties).To(HaveKey("name"))
		})

		It("should not return properties blacklisted in the policy", func() {
			role := "Member"
			filteredSchemas := filterSchemasForPolicy(role, schema.AllTokenTypes, policies, schemas)
			calculateAllowedProperties(filteredSchemas, role, manager)

			networkSchema := filteredSchemas[1]
			Expect(networkSchema.JSONSchemaOnUpdate).ToNot(BeNil())
			Expect(networkSchema.JSONSchemaOnUpdate).To(HaveKey("properties"))

			JSONProperties := networkSchema.JSONSchemaOnUpdate["properties"].(map[string]interface{})
			Expect(JSONProperties).To(HaveLen(3))
			Expect(JSONProperties).ToNot(HaveKey("id"))
			Expect(JSONProperties).ToNot(HaveKey("tenant_id"))
			Expect(JSONProperties).ToNot(HaveKey("providor_networks"))
			Expect(JSONProperties).ToNot(HaveKey("route_targets"))
		})
	})
})
