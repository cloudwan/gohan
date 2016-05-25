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

var _ = Describe("Policies", func() {
	Describe("Policy validation", func() {
		var (
			abstractSchemaPath = "../tests/test_abstract_schema.yaml"
			schemaPath         = "../tests/test_schema.yaml"
			adminTenantID      = "12345678aaaaaaaaaaaa123456789012"
			demoTenantID       = "12345678bbbbbbbbbbbb123456789012"
			adminAuth          Authorization
			memberAuth         Authorization
		)

		BeforeEach(func() {
			manager := GetManager()
			Expect(manager.LoadSchemaFromFile(abstractSchemaPath)).To(Succeed())
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())

			adminAuth = NewAuthorization(adminTenantID, "admin", "fake_token", []string{"admin"}, nil)
			memberAuth = NewAuthorization(demoTenantID, "demo", "fake_token", []string{"Member"}, nil)
		})

		AfterEach(func() {
			ClearManager()
		})

		It("creates network as admin", func() {
			adminPolicy, role := manager.PolicyValidate("create", "/v2.0/networks", adminAuth)
			Expect(adminPolicy).NotTo(BeNil())
			Expect(role.Match("admin")).To(BeTrue())
			Expect(adminPolicy.RequireOwner()).To(BeFalse(), "Admin should not require ownership")
		})

		It("creates network as member", func() {
			memberPolicy, role := manager.PolicyValidate("create", "/v2.0/networks", memberAuth)
			Expect(memberPolicy).NotTo(BeNil())
			Expect(role.Match("Member")).To(BeTrue())
		})

		It("creates network as member - long url", func() {
			memberPolicy, role := manager.PolicyValidate("create", "/v2.0/networks/red", memberAuth)
			Expect(memberPolicy).NotTo(BeNil())
			Expect(role.Match("Member")).To(BeTrue())
			Expect(memberPolicy.RequireOwner()).To(BeTrue(), "Member should require ownership")
		})

		It("creates subnet as member", func() {
			memberPolicy, role := manager.PolicyValidate("create", "/v2.0/network/test1/subnets", memberAuth)
			Expect(memberPolicy).To(BeNil(), "Member should not be allowed to touch subnet %v", memberPolicy)
			Expect(role).To(BeNil())
		})
	})

	Describe("Creation", func() {
		var testPolicy map[string]interface{}

		BeforeEach(func() {
			testPolicy = map[string]interface{}{
				"action":    '*',
				"effect":    "allow",
				"id":        "policy1",
				"principal": "admin",
				"resource": map[string]interface{}{
					"path": ".*",
				},
			}
		})

		It("should show error - invalid condition", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				"invalid_condition",
			}
			_, err := NewPolicy(testPolicy)
			Expect(err).To(MatchError(ContainSubstring("Unknown condition 'invalid_condition'")))
		})

		It("should show error - unknown condition type", func() {
			testPolicy["condition"] = []interface{}{
				map[string]interface{}{
					"type": "unknown",
				},
			}
			_, err := NewPolicy(testPolicy)
			Expect(err).To(MatchError(ContainSubstring("Unknown condition type")))
		})

		It("should show error - invalid condition format", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				5,
			}
			_, err := NewPolicy(testPolicy)
			Expect(err).To(MatchError(ContainSubstring("Invalid condition format")))
		})

		It("tests multiple conditions", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				map[string]interface{}{
					"action":    "read",
					"tenant_id": "acf5662bbff44060b93ac3db3c25a590",
					"type":      "belongs_to",
				},
				map[string]interface{}{
					"action":    "update",
					"tenant_id": "acf5662bbff44060b93ac3db3c25a590",
					"type":      "belongs_to",
				},
			}
			policy, err := NewPolicy(testPolicy)
			Expect(err).NotTo(HaveOccurred())
			Expect(policy.RequireOwner()).To(BeTrue())
			Expect(policy.GetTenantIDFilter("create", "xyz")).To(ConsistOf("xyz"))
			Expect(policy.GetTenantIDFilter("read", "xyz")).To(ConsistOf("xyz", "acf5662bbff44060b93ac3db3c25a590"))
			Expect(policy.GetTenantIDFilter("update", "xyz")).To(ConsistOf("xyz", "acf5662bbff44060b93ac3db3c25a590"))
			Expect(policy.GetTenantIDFilter("delete", "xyz")).To(ConsistOf("xyz"))
		})

		It("tests glob action", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				map[string]interface{}{
					"action":    "*",
					"tenant_id": "acf5662bbff44060b93ac3db3c25a590",
					"type":      "belongs_to",
				},
			}
			policy, err := NewPolicy(testPolicy)
			Expect(err).NotTo(HaveOccurred())
			Expect(policy.RequireOwner()).To(BeTrue())
			Expect(policy.GetTenantIDFilter("create", "xyz")).To(ConsistOf("xyz", "acf5662bbff44060b93ac3db3c25a590"))
			Expect(policy.GetTenantIDFilter("read", "xyz")).To(ConsistOf("xyz", "acf5662bbff44060b93ac3db3c25a590"))
			Expect(policy.GetTenantIDFilter("update", "xyz")).To(ConsistOf("xyz", "acf5662bbff44060b93ac3db3c25a590"))
			Expect(policy.GetTenantIDFilter("delete", "xyz")).To(ConsistOf("xyz", "acf5662bbff44060b93ac3db3c25a590"))
		})
	})

	Describe("Tenants", func() {
		Describe("Creation", func() {
			It("should create tenant successfully", func() {
				tenant := newTenant("tenantID", "tenantName")
				Expect(tenant.ID.String()).To(Equal("tenantID"))
				Expect(tenant.Name.String()).To(Equal("tenantName"))
			})

			It("should create tenant with empty id successfully", func() {
				tenant := newTenant("", "tenantName")
				Expect(tenant.ID.String()).To(Equal(".*"))
				Expect(tenant.Name.String()).To(Equal("tenantName"))
			})

			It("should create tenant with empty name successfully", func() {
				tenant := newTenant("tenantID", "")
				Expect(tenant.ID.String()).To(Equal("tenantID"))
				Expect(tenant.Name.String()).To(Equal(".*"))
			})
		})

		Describe("Comparing", func() {
			It("should compare same tenants successfully", func() {
				tenant := newTenant("tenantID", "tenantName")
				Expect(tenant.equal(tenant)).To(BeTrue())
				Expect(tenant.notEqual(tenant)).To(BeFalse())
			})

			It("should compare different tenants successfully", func() {
				tenant1 := newTenant("tenantID1", "tenantName1")
				tenant2 := newTenant("tenantID2", "tenantName2")
				Expect(tenant1.equal(tenant2)).To(BeFalse())
				Expect(tenant1.notEqual(tenant2)).To(BeTrue())
				Expect(tenant2.equal(tenant1)).To(BeFalse())
				Expect(tenant2.notEqual(tenant1)).To(BeTrue())
			})

			It("should compare same tenants with id only successfully", func() {
				tenant := newTenant("tenantID", "")
				Expect(tenant.equal(tenant)).To(BeTrue())
				Expect(tenant.notEqual(tenant)).To(BeFalse())
			})

			It("should compare different tenants with id only successfully", func() {
				tenant1 := newTenant("tenantID1", "")
				tenant2 := newTenant("tenantID2", "")
				Expect(tenant1.equal(tenant2)).To(BeFalse())
				Expect(tenant1.notEqual(tenant2)).To(BeTrue())
				Expect(tenant2.equal(tenant1)).To(BeFalse())
				Expect(tenant2.notEqual(tenant1)).To(BeTrue())
			})

			It("should compare same tenants with name only successfully", func() {
				tenant := newTenant("", "tenantName")
				Expect(tenant.equal(tenant)).To(BeTrue())
				Expect(tenant.notEqual(tenant)).To(BeFalse())
			})

			It("should compare different tenants with name only successfully", func() {
				tenant1 := newTenant("", "tenantName1")
				tenant2 := newTenant("", "tenantName2")
				Expect(tenant1.equal(tenant2)).To(BeFalse())
				Expect(tenant1.notEqual(tenant2)).To(BeTrue())
				Expect(tenant2.equal(tenant1)).To(BeFalse())
				Expect(tenant2.notEqual(tenant1)).To(BeTrue())
			})

			It("should compare tenant with both values to id only", func() {
				tenant1 := newTenant("tenantID", "tenantName")
				tenant2 := newTenant("tenantID", "")
				Expect(tenant1.equal(tenant2)).To(BeTrue())
				Expect(tenant1.notEqual(tenant2)).To(BeFalse())
				Expect(tenant2.equal(tenant1)).To(BeTrue())
				Expect(tenant2.notEqual(tenant1)).To(BeFalse())
			})

			It("should compare tenant with both values to name only", func() {
				tenant1 := newTenant("tenantID", "tenantName")
				tenant2 := newTenant("", "tenantName")
				Expect(tenant1.equal(tenant2)).To(BeTrue())
				Expect(tenant1.notEqual(tenant2)).To(BeFalse())
				Expect(tenant2.equal(tenant1)).To(BeTrue())
				Expect(tenant2.notEqual(tenant1)).To(BeFalse())
			})
		})
	})

	Describe("Policy check", func() {
		var testPolicy map[string]interface{}
		var policy *Policy
		var authorization BaseAuthorization
		var data map[string]interface{}

		BeforeEach(func() {
			testPolicy = map[string]interface{}{
				"action":    '*',
				"effect":    "allow",
				"id":        "policy1",
				"principal": "admin",
				"resource": map[string]interface{}{
					"path": ".*",
				},
			}
			authorization = BaseAuthorization{
				tenantID:   "userID",
				tenantName: "userName",
				authToken:  "token",
				roles:      []*Role{},
				catalog:    []*Catalog{},
			}
		})

		Describe("Actions on own resources", func() {
			BeforeEach(func() {
				testPolicy["condition"] = []interface{}{"is_owner"}
				policy, _ = NewPolicy(testPolicy)
				data = map[string]interface{}{
					"tenant_id":   "userID",
					"tenant_name": "userName",
				}
			})

			It("should pass check", func() {
				err := policy.Check("create", &authorization, data)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not pass check - not an owner", func() {
				authorization.tenantID = "notOwnerID"
				authorization.tenantName = "notOwnerName"
				err := policy.Check("create", &authorization, data)
				Expect(err).To(MatchError(getProhibitedError("notOwnerName (notOwnerID)", "userName (userID)")))
			})
		})

		Describe("Actions on shared resources", func() {
			BeforeEach(func() {
				data = map[string]interface{}{
					"tenant_id":   "ownerID",
					"tenant_name": "ownerName",
				}
			})

			It("should pass check - tenant_id", func() {
				testPolicy["condition"] = []interface{}{
					"is_owner",
					map[string]interface{}{
						"type":      "belongs_to",
						"tenant_id": "ownerID",
					},
				}
				policy, _ = NewPolicy(testPolicy)
				err := policy.Check("create", &authorization, data)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should pass check - tenant_name", func() {
				testPolicy["condition"] = []interface{}{
					"is_owner",
					map[string]interface{}{
						"type":        "belongs_to",
						"tenant_name": "ownerName",
					},
				}
				policy, _ = NewPolicy(testPolicy)
				err := policy.Check("create", &authorization, data)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("Property based condition", func() {
			It("should work with string condition based on property", func() {
				testPolicy["condition"] = []interface{}{
					map[string]interface{}{
						"type":   "property",
						"action": "read",
						"match": map[string]interface{}{
							"status": "ACTIVE",
						},
					},
				}

				policy, _ = NewPolicy(testPolicy)
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ACTIVE",
				}, nil)).To(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ERROR",
				}, nil)).NotTo(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{}, nil)).NotTo(Succeed())
			})

			It("should work with string array condition based on propery", func() {
				testPolicy["condition"] = []interface{}{
					map[string]interface{}{
						"type":   "property",
						"action": "read",
						"match": map[string]interface{}{
							"status": []interface{}{
								"ACTIVE", "CREATING"},
						},
					},
					map[string]interface{}{
						"type":   "property",
						"action": "create",
						"match": map[string]interface{}{
							"status": []interface{}{
								"ACTIVE"},
						},
					},
					map[string]interface{}{
						"type":   "property",
						"action": "update",
						"match": map[string]interface{}{
							"status": map[string]interface{}{
								"ACTIVE": []interface{}{"UPDATING", "ERROR"},
							},
						},
					},
				}

				policy, _ = NewPolicy(testPolicy)
				Expect(policy.ApplyPropertyConditionFilter("create", map[string]interface{}{
					"status": "ACTIVE",
				}, nil)).To(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ACTIVE",
				}, nil)).To(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "CREATING",
				}, nil)).To(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ERROR",
				}, nil)).NotTo(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("update", map[string]interface{}{
					"status": "ACTIVE",
				}, map[string]interface{}{
					"status": "UPDATING",
				})).To(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("update", map[string]interface{}{
					"status": "ACTIVE",
				}, map[string]interface{}{
					"status": "ERROR",
				})).To(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("update", map[string]interface{}{
					"status": "ACTIVE",
				}, map[string]interface{}{
					"status": "FATAL_ERROR",
				})).NotTo(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ERROR",
				}, nil)).NotTo(Succeed())
				Expect(policy.ApplyPropertyConditionFilter("read", map[string]interface{}{}, nil)).NotTo(Succeed())
			})
		})
	})
})

func getProhibitedError(caller, owner string) string {
	return fmt.Sprintf("Tenant '%s' is prohibited from operating on resources of tenant '%s'", caller, owner)
}
