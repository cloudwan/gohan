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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	tenantProhibitedError = "Operating on resources from other tenant is prohibited"
	domainProhibitedError = "Operating on resources from other domain is prohibited"
)

var _ = Describe("Policies", func() {
	Describe("Policy validation", func() {
		var (
			manager            *Manager
			abstractSchemaPath = "../tests/test_abstract_schema.yaml"
			schemaPath         = "../tests/test_schema.yaml"
			adminTenantID      = "12345678aaaaaaaaaaaa123456789012"
			demoTenantID       = "12345678bbbbbbbbbbbb123456789012"
			adminAuth          Authorization
			legacyAdminAuth    Authorization
			memberAuth         Authorization
		)

		BeforeEach(func() {
			manager = GetManager()
			Expect(manager.LoadSchemaFromFile(abstractSchemaPath)).To(Succeed())
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())

			adminAuth = NewAuthorizationBuilder().
				WithTenant(Tenant{ID: adminTenantID, Name: "admin"}).
				WithRoleIDs("admin").
				BuildAdmin()
			legacyAdminAuth = NewAuthorizationBuilder().
				WithKeystoneV2Compatibility().
				WithTenant(Tenant{ID: adminTenantID, Name: "admin"}).
				WithRoleIDs("admin").
				BuildScopedToTenant()
			memberAuth = NewAuthorizationBuilder().
				WithTenant(Tenant{ID: demoTenantID, Name: "demo"}).
				WithRoleIDs("Member").
				BuildScopedToTenant()
		})

		AfterEach(func() {
			ClearManager()
		})

		DescribeTable("creates network as admin",
			func(auth *Authorization) {
				adminPolicy, role := manager.PolicyValidate("create", "/v2.0/networks", *auth)
				Expect(adminPolicy).NotTo(BeNil())
				Expect(role.Match("admin")).To(BeTrue())
				currCond := adminPolicy.GetCurrentResourceCondition()
				Expect(currCond.RequireOwner()).To(BeFalse(), "Admin should not require ownership")
			},
			Entry("Keystone V2 admin", &adminAuth),
			Entry("Keystone V3 admin", &legacyAdminAuth),
		)

		It("creates network as member", func() {
			memberPolicy, role := manager.PolicyValidate("create", "/v2.0/networks", memberAuth)
			Expect(memberPolicy).NotTo(BeNil())
			Expect(role.Match("Member")).To(BeTrue())
		})

		It("creates network as member - long url", func() {
			memberPolicy, role := manager.PolicyValidate("create", "/v2.0/networks/red", memberAuth)
			Expect(memberPolicy).NotTo(BeNil())
			Expect(role.Match("Member")).To(BeTrue())
			currCond := memberPolicy.GetCurrentResourceCondition()
			Expect(currCond.RequireOwner()).To(BeTrue(), "Member should require ownership")
		})

		It("creates subnet as member", func() {
			memberPolicy, role := manager.PolicyValidate("create", "/v2.0/network/test1/subnets", memberAuth)
			Expect(memberPolicy).To(BeNil(), "Member should not be allowed to touch subnet %v", memberPolicy)
			Expect(role).To(BeNil())
		})
	})

	Describe("Creation", func() {
		var (
			manager      *Manager
			testPolicy   map[string]interface{}
			someTenantID = "acf5662bbff44060b93ac3db3c25a590"
			xyzAuth      Authorization
		)

		getTenantIDFilter := func(rc *ResourceCondition, action string, auth Authorization) []string {
			tenantIDs, _ := rc.GetTenantAndDomainFilters(action, auth)
			return tenantIDs
		}

		BeforeEach(func() {
			manager = GetManager()
			testPolicy = map[string]interface{}{
				"action":    '*',
				"effect":    "allow",
				"id":        "policy1",
				"principal": "admin",
				"resource": map[string]interface{}{
					"path": ".*",
				},
			}

			tenant := Tenant{ID: "xyz", Name: "xyz"}
			xyzAuth = NewAuthorizationBuilder().
				WithTenant(tenant).
				WithRoleIDs("Member").
				BuildScopedToTenant()
		})

		It("should return error on both types of properties", func() {
			testPolicy["resource"].(map[string]interface{})["properties"] = []string{"a"}
			testPolicy["resource"].(map[string]interface{})["blacklistProperties"] = []string{"b"}

			_, err := NewPolicy(testPolicy)
			Expect(err).To(HaveOccurred())
		})

		It("should show panic on invalid condition", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				"invalid_condition",
			}
			Expect(func() { NewPolicy(testPolicy) }).To(Panic())
		})

		It("should show panic on unknown condition type", func() {
			testPolicy["condition"] = []interface{}{
				map[string]interface{}{
					"type": "unknown",
				},
			}
			Expect(func() { NewPolicy(testPolicy) }).To(Panic())
		})

		It("should panic on invalid condition format", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				5,
			}
			Expect(func() { NewPolicy(testPolicy) }).To(Panic())
		})

		It("tests multiple conditions", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				map[string]interface{}{
					"action":    "read",
					"tenant_id": someTenantID,
					"type":      "belongs_to",
				},
				map[string]interface{}{
					"action":    "update",
					"tenant_id": someTenantID,
					"type":      "belongs_to",
				},
			}
			policy, err := NewPolicy(testPolicy)
			Expect(err).NotTo(HaveOccurred())
			currCond := policy.GetCurrentResourceCondition()
			Expect(currCond.RequireOwner()).To(BeTrue())
			Expect(getTenantIDFilter(currCond, "create", xyzAuth)).To(ConsistOf("xyz"))
			Expect(getTenantIDFilter(currCond, "read", xyzAuth)).To(ConsistOf("xyz", someTenantID))
			Expect(getTenantIDFilter(currCond, "update", xyzAuth)).To(ConsistOf("xyz", someTenantID))
			Expect(getTenantIDFilter(currCond, "delete", xyzAuth)).To(ConsistOf("xyz"))
		})

		It("tests glob action", func() {
			testPolicy["condition"] = []interface{}{
				"is_owner",
				map[string]interface{}{
					"action":    "*",
					"tenant_id": someTenantID,
					"type":      "belongs_to",
				},
			}
			policy, err := NewPolicy(testPolicy)
			Expect(err).NotTo(HaveOccurred())
			currCond := policy.GetCurrentResourceCondition()
			Expect(currCond.RequireOwner()).To(BeTrue())
			Expect(getTenantIDFilter(currCond, "create", xyzAuth)).To(ConsistOf("xyz", someTenantID))
			Expect(getTenantIDFilter(currCond, "read", xyzAuth)).To(ConsistOf("xyz", someTenantID))
			Expect(getTenantIDFilter(currCond, "update", xyzAuth)).To(ConsistOf("xyz", someTenantID))
			Expect(getTenantIDFilter(currCond, "delete", xyzAuth)).To(ConsistOf("xyz", someTenantID))
		})

		Describe("'__attach__' policy", func() {
			var (
				abstractSchemaPath = "../tests/test_abstract_schema.yaml"
				schemaPath         = "../tests/test_schema.yaml"
				testPolicy         map[string]interface{}
			)

			BeforeEach(func() {
				Expect(manager.LoadSchemaFromFile(abstractSchemaPath)).To(Succeed())
				Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
				testPolicy = map[string]interface{}{
					"action":    "__attach__",
					"id":        "attach_test",
					"effect":    "allow",
					"principal": "admin",
					"resource": map[string]interface{}{
						"path": ".*",
					},
					"relation_property": "attach_if_accessible_id",
					"target_condition":  []interface{}{"is_owner"},
				}
			})

			type policyModifier func(p map[string]interface{})

			It("should create a valid attach policy successfully", func() {
				policy, err := NewPolicy(testPolicy)
				Expect(err).ToNot(HaveOccurred())
				Expect(policy.Action).To(Equal(ActionAttach))
				Expect(policy.GetCurrentResourceCondition()).ToNot(BeNil())
				Expect(policy.GetRelationPropertyName()).To(Equal("attach_if_accessible_id"))
			})

			DescribeTable("Attach policy creation failure tests",
				func(modifier policyModifier, expectedMessage string) {
					modifier(testPolicy)
					_, err := NewPolicy(testPolicy)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(expectedMessage))
				},
				Entry("missing relation_property",
					func(p map[string]interface{}) { delete(p, "relation_property") },
					"\"relation_property\" is required in an attach policy",
				),
				Entry("missing target_condition",
					func(p map[string]interface{}) { delete(p, "target_condition") },
					"\"target_condition\" is required in an attach policy",
				),
			)
		})
	})

	Describe("Tenants", func() {
		Describe("Creation", func() {
			It("should create tenant successfully", func() {
				tenant := newTenantMatcher("tenantID", "tenantName")
				Expect(tenant.ID.String()).To(Equal("tenantID"))
				Expect(tenant.Name.String()).To(Equal("tenantName"))
			})

			It("should create tenant with empty id successfully", func() {
				tenant := newTenantMatcher("", "tenantName")
				Expect(tenant.ID.String()).To(Equal(".*"))
				Expect(tenant.Name.String()).To(Equal("tenantName"))
			})

			It("should create tenant with empty name successfully", func() {
				tenant := newTenantMatcher("tenantID", "")
				Expect(tenant.ID.String()).To(Equal("tenantID"))
				Expect(tenant.Name.String()).To(Equal(".*"))
			})
		})

		Describe("Comparing", func() {
			It("should compare same tenants successfully", func() {
				tenant := newTenantMatcher("tenantID", "tenantName")
				Expect(tenant.equal(tenant)).To(BeTrue())
				Expect(tenant.notEqual(tenant)).To(BeFalse())
			})

			It("should compare different tenants successfully", func() {
				tenant1 := newTenantMatcher("tenantID1", "tenantName1")
				tenant2 := newTenantMatcher("tenantID2", "tenantName2")
				Expect(tenant1.equal(tenant2)).To(BeFalse())
				Expect(tenant1.notEqual(tenant2)).To(BeTrue())
				Expect(tenant2.equal(tenant1)).To(BeFalse())
				Expect(tenant2.notEqual(tenant1)).To(BeTrue())
			})

			It("should compare same tenants with id only successfully", func() {
				tenant := newTenantMatcher("tenantID", "")
				Expect(tenant.equal(tenant)).To(BeTrue())
				Expect(tenant.notEqual(tenant)).To(BeFalse())
			})

			It("should compare different tenants with id only successfully", func() {
				tenant1 := newTenantMatcher("tenantID1", "")
				tenant2 := newTenantMatcher("tenantID2", "")
				Expect(tenant1.equal(tenant2)).To(BeFalse())
				Expect(tenant1.notEqual(tenant2)).To(BeTrue())
				Expect(tenant2.equal(tenant1)).To(BeFalse())
				Expect(tenant2.notEqual(tenant1)).To(BeTrue())
			})

			It("should compare same tenants with name only successfully", func() {
				tenant := newTenantMatcher("", "tenantName")
				Expect(tenant.equal(tenant)).To(BeTrue())
				Expect(tenant.notEqual(tenant)).To(BeFalse())
			})

			It("should compare different tenants with name only successfully", func() {
				tenant1 := newTenantMatcher("", "tenantName1")
				tenant2 := newTenantMatcher("", "tenantName2")
				Expect(tenant1.equal(tenant2)).To(BeFalse())
				Expect(tenant1.notEqual(tenant2)).To(BeTrue())
				Expect(tenant2.equal(tenant1)).To(BeFalse())
				Expect(tenant2.notEqual(tenant1)).To(BeTrue())
			})

			It("should compare tenant with both values to id only", func() {
				tenant1 := newTenantMatcher("tenantID", "tenantName")
				tenant2 := newTenantMatcher("tenantID", "")
				Expect(tenant1.equal(tenant2)).To(BeTrue())
				Expect(tenant1.notEqual(tenant2)).To(BeFalse())
				Expect(tenant2.equal(tenant1)).To(BeTrue())
				Expect(tenant2.notEqual(tenant1)).To(BeFalse())
			})

			It("should compare tenant with both values to name only", func() {
				tenant1 := newTenantMatcher("tenantID", "tenantName")
				tenant2 := newTenantMatcher("", "tenantName")
				Expect(tenant1.equal(tenant2)).To(BeTrue())
				Expect(tenant1.notEqual(tenant2)).To(BeFalse())
				Expect(tenant2.equal(tenant1)).To(BeTrue())
				Expect(tenant2.notEqual(tenant1)).To(BeFalse())
			})
		})
	})

	Describe("Policy check", func() {
		var manager *Manager
		var testPolicy map[string]interface{}
		var policy *Policy
		var authorizationBuilder *AuthorizationBuilder
		var authorization Authorization
		var data map[string]interface{}

		BeforeEach(func() {
			manager = GetManager()
			testPolicy = map[string]interface{}{
				"action":    '*',
				"effect":    "allow",
				"id":        "policy1",
				"principal": "admin",
				"resource": map[string]interface{}{
					"path": ".*",
				},
			}
			authorizationBuilder = NewAuthorizationBuilder().
				WithTenant(Tenant{
					ID:   "userID",
					Name: "userName",
				}).
				WithDomain(Domain{
					ID:   "domainID",
					Name: "domainName",
				})
			authorization = authorizationBuilder.BuildScopedToTenant()
		})

		Describe("Actions on own resources", func() {
			BeforeEach(func() {
				testPolicy["condition"] = []interface{}{"is_owner"}
				policy, _ = NewPolicy(testPolicy)
				data = map[string]interface{}{
					"tenant_id":   "userID",
					"tenant_name": "userName",
					"domain_id":   "domainID",
					"domain_name": "domainName",
				}
			})

			It("should pass check", func() {
				err := policy.Check("create", authorization, data)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not pass check - not an owner", func() {
				authorization = authorizationBuilder.
					WithTenant(Tenant{
						ID:   "notOwnerID",
						Name: "notOwnerName",
					}).
					BuildScopedToTenant()
				err := policy.Check("create", authorization, data)
				Expect(err).To(MatchError(tenantProhibitedError))
			})

			It("should not pass check - different domain", func() {
				authorization = authorizationBuilder.
					WithDomain(Domain{
						ID:   "otherDomainID",
						Name: "otherDomainName",
					}).
					BuildScopedToDomain()
				err := policy.Check("create", authorization, data)
				Expect(err).To(MatchError(domainProhibitedError))
			})

			Describe("Effect property", func() {
				BeforeEach(func() {
					policy.Action = "*"
					authorization = authorizationBuilder.WithRoleIDs("admin").BuildAdmin()
				})
				It("should allow access be default", func() {
					policy.Effect = ""
					receivedPolicy, role := PolicyValidate("create", "/abc", authorization, []*Policy{policy})
					Expect(receivedPolicy.ID).To(Equal("admin_statement"))
					Expect(role).To(Equal(&Role{"admin"}))
				})

				It("should deny access", func() {
					policy.Effect = "deny"
					policy, role := PolicyValidate("create", "/abc", authorization, []*Policy{policy})
					Expect(policy).To(BeNil())
					Expect(role).To(BeNil())
				})
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
				err := policy.Check("create", authorization, data)
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
				err := policy.Check("create", authorization, data)
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
				currCond := policy.GetCurrentResourceCondition()
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ACTIVE",
				}, nil)).To(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ERROR",
				}, nil)).NotTo(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{}, nil)).NotTo(Succeed())
			})

			It("should work with string array condition based on property", func() {
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
				currCond := policy.GetCurrentResourceCondition()
				Expect(currCond.ApplyPropertyConditionFilter("create", map[string]interface{}{
					"status": "ACTIVE",
				}, nil)).To(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ACTIVE",
				}, nil)).To(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "CREATING",
				}, nil)).To(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ERROR",
				}, nil)).NotTo(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("update", map[string]interface{}{
					"status": "ACTIVE",
				}, map[string]interface{}{
					"status": "UPDATING",
				})).To(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("update", map[string]interface{}{
					"status": "ACTIVE",
				}, map[string]interface{}{
					"status": "ERROR",
				})).To(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("update", map[string]interface{}{
					"status": "ACTIVE",
				}, map[string]interface{}{
					"status": "FATAL_ERROR",
				})).NotTo(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{
					"status": "ERROR",
				}, nil)).NotTo(Succeed())
				Expect(currCond.ApplyPropertyConditionFilter("read", map[string]interface{}{}, nil)).NotTo(Succeed())
			})
		})

		Describe("Custom filter", func() {
			var testAuth Authorization

			getSchema := func(name string) *Schema {
				schema, ok := manager.Schema(name)
				Expect(ok).To(BeTrue())
				return schema
			}

			BeforeEach(func() {
				tenant := Tenant{ID: "test", Name: "test"}
				testAuth = NewAuthorizationBuilder().
					WithTenant(tenant).
					WithRoleIDs("Member").
					BuildScopedToTenant()
			})

			It("should work with string condition based on conjunction property", func() {
				schema := getSchema("test")
				testPolicy["condition"] = []interface{}{
					map[string]interface{}{
						"and": []interface{}{
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "status",
									"type":     "eq",
									"value":    "ACTIVE",
								},
							},
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "state",
									"type":     "eq",
									"value":    "UP",
								},
							},
						},
					},
				}

				var err error
				policy, err = NewPolicy(testPolicy)
				Expect(err).ToNot(HaveOccurred())
				filter := map[string]interface{}{}
				currCond := policy.GetCurrentResourceCondition()
				currCond.AddCustomFilters(schema, filter, testAuth)
				expected := map[string]interface{}{
					"__and__": []map[string]interface{}{
						{
							"property": "status",
							"type":     "eq",
							"value":    "ACTIVE",
						},
						{
							"property": "state",
							"type":     "eq",
							"value":    "UP",
						},
					},
				}
				Expect(filter).To(Equal(expected))
			})
			It("should work with string condition based on disjunction property", func() {
				schema := getSchema("test")
				testPolicy["condition"] = []interface{}{
					map[string]interface{}{
						"or": []interface{}{
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "status",
									"type":     "eq",
									"value":    "ACTIVE",
								},
							},
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "state",
									"type":     "eq",
									"value":    "UP",
								},
							},
						},
					},
				}

				var err error
				policy, err = NewPolicy(testPolicy)
				Expect(err).ToNot(HaveOccurred())
				filter := map[string]interface{}{}
				currCond := policy.GetCurrentResourceCondition()
				currCond.AddCustomFilters(schema, filter, testAuth)
				expected := map[string]interface{}{
					"__or__": []map[string]interface{}{
						{
							"property": "status",
							"type":     "eq",
							"value":    "ACTIVE",
						},
						{
							"property": "state",
							"type":     "eq",
							"value":    "UP",
						},
					},
				}
				Expect(filter).To(Equal(expected))
			})
			It("should work with string condition based on is_owner, con/disjunction property", func() {
				schema := getSchema("test")
				testPolicy["condition"] = []interface{}{
					map[string]interface{}{
						"or": []interface{}{
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "status",
									"type":     "eq",
									"value":    "ACTIVE",
								},
							},
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "state",
									"type":     "eq",
									"value":    "UP",
								},
							},
							map[string]interface{}{
								"and": []interface{}{
									"is_owner",
									map[string]interface{}{
										"match": map[string]interface{}{
											"property": "state",
											"type":     "eq",
											"value":    "DOWN",
										},
									},
								},
							},
						},
					},
				}

				var err error
				policy, err = NewPolicy(testPolicy)
				Expect(err).ToNot(HaveOccurred())
				filter := map[string]interface{}{}
				currCond := policy.GetCurrentResourceCondition()
				currCond.AddCustomFilters(schema, filter, testAuth)
				expected := map[string]interface{}{
					"__or__": []map[string]interface{}{
						{
							"property": "status",
							"type":     "eq",
							"value":    "ACTIVE",
						},
						{
							"property": "state",
							"type":     "eq",
							"value":    "UP",
						},
						{
							"__and__": []map[string]interface{}{
								{
									"__and__": []map[string]interface{}{
										{
											"property": "tenant_id",
											"type":     "eq",
											"value":    testAuth.TenantID(),
										},
										{
											"property": "domain_id",
											"type":     "eq",
											"value":    testAuth.DomainID(),
										},
									},
								},
								{
									"property": "state",
									"type":     "eq",
									"value":    "DOWN",
								},
							},
						},
					},
				}
				Expect(filter).To(Equal(expected))
			})
			It("Should, in case of is_owner, filter by domain_id only when the field is defined in the schema", func() {
				schema := getSchema("network")
				testPolicy["condition"] = []interface{}{
					map[string]interface{}{
						"or": []interface{}{
							"is_owner",
							map[string]interface{}{
								"match": map[string]interface{}{
									"property": "status",
									"type":     "eq",
									"value":    "ACTIVE",
								},
							},
						},
					},
				}

				var err error
				policy, err = NewPolicy(testPolicy)
				Expect(err).ToNot(HaveOccurred())
				filter := map[string]interface{}{}
				currCond := policy.GetCurrentResourceCondition()
				currCond.AddCustomFilters(schema, filter, testAuth)
				expected := map[string]interface{}{
					"__or__": []map[string]interface{}{
						{
							"property": "tenant_id",
							"type":     "eq",
							"value":    testAuth.TenantID(),
						},
						{
							"property": "status",
							"type":     "eq",
							"value":    "ACTIVE",
						},
					},
				}
				Expect(filter).To(Equal(expected))
			})
		})
	})
})
