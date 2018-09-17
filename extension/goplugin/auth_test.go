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

package goplugin_test

import (
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	abstractSchemaPath = "../../tests/test_abstract_schema.yaml"
	schemaPath         = "../../tests/test_schema.yaml"
	adminTenantID      = "12345678aaaaaaaaaaaa123456789012"
	demoTenantID       = "12345678bbbbbbbbbbbb123456789012"
)

var _ = Describe("Auth", func() {

	var (
		manager               *schema.Manager
		adminAuth             schema.Authorization
		adminOnDemoAuth       schema.Authorization
		memberAuth            schema.Authorization
		env                   goext.IEnvironment
	)

	BeforeEach(func() {
		manager = schema.GetManager()

		Expect(manager.LoadSchemaFromFile(abstractSchemaPath)).To(Succeed())
		Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())

		env = goplugin.NewEnvironment("test", nil, nil)
	})

	AfterEach(func() {
		schema.ClearManager()
	})

	setup := func(auth schema.Authorization) goext.Context {
		policy, role := manager.PolicyValidate("create", "/v2.0/networks", auth)
		Expect(policy).NotTo(BeNil())

		context := goext.MakeContext()
		context["policy"] = policy
		context["role"] = role
		context["tenant_id"] = auth.TenantID()
		context["auth"] = auth

		return context
	}

	Context("Keystone V2", func() {
		BeforeEach(func() {
			adminAuth = schema.NewAuthorizationBuilder().
				WithKeystoneV2Compatibility().
				WithTenant(schema.Tenant{ID: adminTenantID, Name: "admin"}).
				WithRoleIDs("admin").
				BuildScopedToTenant()
			adminOnDemoAuth = schema.NewAuthorizationBuilder().
				WithKeystoneV2Compatibility().
				WithTenant(schema.Tenant{ID: adminTenantID, Name: "demo"}).
				WithRoleIDs("admin").
				BuildScopedToTenant()
			memberAuth = schema.NewAuthorizationBuilder().
				WithKeystoneV2Compatibility().
				WithTenant(schema.Tenant{ID: demoTenantID, Name: "demo"}).
				WithRoleIDs("Member").
				BuildScopedToTenant()
		})

		Context("IsAdmin", func() {
			It("Returns true for admin context", func() {
				context := setup(adminAuth)
				Expect(env.Auth().IsAdmin(context)).To(BeTrue())
			})

			It("Returns true for admin user logged in as other tenant", func() {
				context := setup(adminOnDemoAuth)
				Expect(env.Auth().IsAdmin(context)).To(BeTrue())
			})
		})

		Context("GetTenantName", func() {
			It("Returns admin for admin context", func() {
				context := setup(adminAuth)
				Expect(env.Auth().GetTenantName(context)).To(Equal("admin"))
			})

			It("Returns demo for admin user logged in as demo", func() {
				context := setup(adminOnDemoAuth)
				Expect(env.Auth().GetTenantName(context)).To(Equal("demo"))
			})
		})

		Context("HasRole", func() {
			It("Returns true for admin role in admin context", func() {
				context := setup(adminAuth)
				Expect(env.Auth().HasRole(context, "admin")).To(BeTrue())
			})

			It("Returns true for admin role when admin logged in as demo", func() {
				context := setup(adminOnDemoAuth)
				Expect(env.Auth().HasRole(context, "admin")).To(BeTrue())
			})
		})
	})

	Context("Keystone V3", func() {
		BeforeEach(func() {
			adminAuth = schema.NewAuthorizationBuilder().
				WithTenant(schema.Tenant{ID: adminTenantID, Name: "admin"}).
				WithRoleIDs("admin").
				BuildAdmin()
			adminOnDemoAuth = schema.NewAuthorizationBuilder().
				WithTenant(schema.Tenant{ID: adminTenantID, Name: "demo"}).
				WithRoleIDs("admin").
				BuildScopedToTenant()
			memberAuth = schema.NewAuthorizationBuilder().
				WithTenant(schema.Tenant{ID: demoTenantID, Name: "demo"}).
				WithRoleIDs("Member").
				BuildScopedToTenant()
		})

		Context("IsAdmin", func() {
			It("Returns true for admin context", func() {
				context := setup(adminAuth)
				Expect(env.Auth().IsAdmin(context)).To(BeTrue())
			})

			It("Returns false for member context", func() {
				context := setup(memberAuth)
				Expect(env.Auth().IsAdmin(context)).To(BeFalse())
			})

			It("Returns false for admin user logged in as other tenant", func() {
				context := setup(adminOnDemoAuth)
				Expect(env.Auth().IsAdmin(context)).To(BeFalse())
			})
		})

		Context("GetTenantName", func() {
			It("Returns admin for admin context", func() {
				context := setup(adminAuth)
				Expect(env.Auth().GetTenantName(context)).To(Equal("admin"))
			})

			It("Returns demo for demo context", func() {
				context := setup(memberAuth)
				Expect(env.Auth().GetTenantName(context)).To(Equal("demo"))
			})

			It("Returns demo for admin user logged in as demo", func() {
				context := setup(adminOnDemoAuth)
				Expect(env.Auth().GetTenantName(context)).To(Equal("demo"))
			})
		})

		Context("HasRole", func() {
			It("Returns false for admin role in demo context", func() {
				context := setup(memberAuth)
				Expect(env.Auth().HasRole(context, "admin")).To(BeFalse())
			})
		})
	})
})
