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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resources", func() {
	BeforeEach(func() {
		manager := GetManager()
		gohanSchemaPath := "../etc/schema/core.json"
		schemaPath := "../tests/test_schema.json"
		Expect(manager.ValidateSchema(gohanSchemaPath, schemaPath)).To(Succeed())
		Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
	})

	AfterEach(func() {
		ClearManager()
	})

	It("tests defaults", func() {
		networkSchema, exists := manager.Schema("network")
		Expect(exists).To(BeTrue())
		networkRedObj := map[string]interface{}{
			"id":                "networkRed",
			"name":              "NetworkRed",
			"tenant_id":         "red",
			"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"},
		}
		networkRed, err := NewResource(networkSchema, networkRedObj)
		Expect(err).ToNot(HaveOccurred())
		actualRT, ok := networkRed.Data()["route_targets"]
		Expect(ok).To(BeFalse(), "networkRed should not contain route_targets")

		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedRT := []interface{}{}
		actualRT, ok = networkRed.Data()["route_targets"]
		Expect(ok).To(BeTrue(), "networkRed should contain route_targets")
		Expect(actualRT).To(Equal(expectedRT))
	})
})
