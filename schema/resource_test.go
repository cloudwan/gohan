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
	var (
		manager *Manager
	)

	BeforeEach(func() {
		manager = GetManager()
		gohanSchemaPath := "../etc/schema/core.json"
		schemaPath := "../tests/test_schema.json"
		Expect(manager.ValidateSchema(gohanSchemaPath, schemaPath)).To(Succeed())
		Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
	})

	AfterEach(func() {
		ClearManager()
	})

	It("default value array", func() {
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
		_, ok := networkRed.Data()["route_targets"]
		Expect(ok).To(BeFalse(), "networkRed should not contain route_targets")

		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedRT := []interface{}{}
		actualRT, ok := networkRed.Data()["route_targets"]
		Expect(ok).To(BeTrue(), "networkRed should contain route_targets")
		Expect(actualRT).To(Equal(expectedRT))
	})

	It("default values inside nested object", func() {
		networkSchema, exists := manager.Schema("network")
		Expect(exists).To(BeTrue())
		networkRedObj := map[string]interface{}{
			"id":                "networkRed",
			"name":              "NetworkRed",
			"tenant_id":         "red",
			"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"},
			"route_targets":     []interface{}{},
			"config": map[string]interface{}{
				"default_vlan": map[string]interface{}{
					"name": "my_vlan",
				},
			},
		}
		networkRed, err := NewResource(networkSchema, networkRedObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedConfig := map[string]interface{}{
			"default_vlan": map[string]interface{}{
				"name":    "my_vlan",
				"vlan_id": float64(1),
			},
			"empty_vlan": map[string]interface{}{},
			"vpn_vlan": map[string]interface{}{
				"name": "vpn_vlan",
			},
		}
		actualConfig, ok := networkRed.Data()["config"]
		Expect(ok).To(BeTrue(), "networkRed should contain config")
		Expect(actualConfig).To(Equal(expectedConfig))
	})

	It("fills all default values", func() {
		networkSchema, exists := manager.Schema("network")
		Expect(exists).To(BeTrue())
		networkRedObj := map[string]interface{}{
			"id":                "networkRed",
			"name":              "NetworkRed",
			"tenant_id":         "red",
			"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"},
			"route_targets":     []interface{}{},
			"config": map[string]interface{}{
				"default_vlan": map[string]interface{}{},
			},
		}
		networkRed, err := NewResource(networkSchema, networkRedObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedConfig := map[string]interface{}{
			"default_vlan": map[string]interface{}{
				"name":    "default_vlan",
				"vlan_id": float64(1),
			},
			"vpn_vlan": map[string]interface{}{
				"name": "vpn_vlan",
			},
			"empty_vlan": map[string]interface{}{},
		}
		actualConfig, ok := networkRed.Data()["config"]
		Expect(ok).To(BeTrue(), "networkRed should contain config")
		Expect(actualConfig).To(Equal(expectedConfig))
	})

	It("empty object as default value", func() {
		networkSchema, exists := manager.Schema("network")
		Expect(exists).To(BeTrue())
		networkRedObj := map[string]interface{}{
			"id":                "networkRed",
			"name":              "NetworkRed",
			"tenant_id":         "red",
			"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"},
			"route_targets":     []interface{}{},
			"config":            map[string]interface{}{},
		}
		networkRed, err := NewResource(networkSchema, networkRedObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedConfig := map[string]interface{}{
			"empty_vlan": map[string]interface{}{},
			"vpn_vlan": map[string]interface{}{
				"name": "vpn_vlan",
			},
		}
		actualConfig, ok := networkRed.Data()["config"]
		Expect(ok).To(BeTrue(), "networkRed should contain config")
		Expect(actualConfig).To(Equal(expectedConfig))
	})

	It("allows nil property when type is object", func() {
		networkSchema, exists := manager.Schema("network")
		Expect(exists).To(BeTrue())
		networkRedObj := map[string]interface{}{
			"id":                "networkRed",
			"name":              "NetworkRed",
			"tenant_id":         "red",
			"providor_networks": nil,
			"route_targets":     []interface{}{},
			"config":            map[string]interface{}{},
		}
		networkRed, err := NewResource(networkSchema, networkRedObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedProvidorNetworks := map[string]interface{}{}
		actualProvidorNetworks, ok := networkRed.Data()["providor_networks"]
		Expect(ok).To(BeTrue(), "networkRed should contain config")
		Expect(actualProvidorNetworks).To(Equal(expectedProvidorNetworks))
	})

	It("override property which has default object", func() {
		networkSchema, exists := manager.Schema("network")
		Expect(exists).To(BeTrue())
		networkRedObj := map[string]interface{}{
			"id":                "networkRed",
			"name":              "NetworkRed",
			"tenant_id":         "red",
			"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"},
			"route_targets":     []interface{}{},
			"config": map[string]interface{}{
				"vpn_vlan": map[string]interface{}{
					"name": "my_vpn_vlan",
				},
			},
		}
		networkRed, err := NewResource(networkSchema, networkRedObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkRed.PopulateDefaults()).To(Succeed())
		expectedConfig := map[string]interface{}{
			"empty_vlan": map[string]interface{}{},
			"vpn_vlan": map[string]interface{}{
				"name": "my_vpn_vlan",
			},
		}
		actualConfig, ok := networkRed.Data()["config"]
		Expect(ok).To(BeTrue(), "networkRed should contain config")
		Expect(actualConfig).To(Equal(expectedConfig))
	})
})
