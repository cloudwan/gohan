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
		schemas  []*schema.Schema
		policies []*schema.Policy
	)

	BeforeEach(func() {
		manager := schema.GetManager()
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
		Context("With policy set to admin", func() {
			It("should return only admin's schemas", func() {

				filteredSchemas := filterSchemasForPolicy("admin", policies, schemas)

				Expect(filteredSchemas).To(HaveLen(3))
				Expect(filteredSchemas[0].URL).To(Equal("/v2.0/nets"))
				Expect(filteredSchemas[1].URL).To(Equal("/v2.0/networks"))
				Expect(filteredSchemas[2].URL).To(Equal("/v2.0/network/:network/subnets"))
			})
		})

		Context("With policy set to member", func() {
			It("should return only member's schemas", func() {

				filteredSchemas := filterSchemasForPolicy("Member", policies, schemas)

				Expect(filteredSchemas).To(HaveLen(1))
				Expect(filteredSchemas[0].URL).To(Equal("/v2.0/networks"))
			})
		})

		Context("With policy set to nobody", func() {
			It("should return only nobody's schemas", func() {

				filteredSchemas := filterSchemasForPolicy("Nobody", policies, schemas)

				Expect(filteredSchemas).To(BeEmpty())
			})
		})
	})
})
