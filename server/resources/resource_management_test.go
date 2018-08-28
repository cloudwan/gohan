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

package resources_test

import (
	context_pkg "context"
	"fmt"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/server/resources"
	"github.com/cloudwan/gohan/util"
	"github.com/mohae/deepcopy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/twinj/uuid"
)

var _ = Describe("Resource manager", func() {
	const (
		adminResourceID       = "6660fbf8-ca60-4cb0-a42e-9a913beafbaf"
		memberResourceID      = "6660fbf8-ca60-4cb0-a42e-9a913beafbae"
		otherDomainResourceID = "6660fbf8-ca60-4cb0-a42e-9a913beafbad"
	)

	var (
		manager                 *schema.Manager
		adminAuth               schema.Authorization
		memberAuth              schema.Authorization
		domainScopedAuth        schema.Authorization
		auth                    schema.Authorization
		context                 middleware.Context
		schemaID                string
		path                    string
		action                  string
		currentSchema           *schema.Schema
		extensions              []*schema.Extension
		env                     extension.Environment
		events                  map[string]string
		timeLimit               time.Duration
		timeLimits              []*schema.PathEventTimeLimit
		ctx                     context_pkg.Context
		adminResourceData       map[string]interface{}
		memberResourceData      map[string]interface{}
		otherDomainResourceData map[string]interface{}
	)

	BeforeEach(func() {
		manager = schema.GetManager()
		ctx = context_pkg.Background()

		domainA := schema.Domain{
			ID:   domainAID,
			Name: "domainA",
		}

		adminAuth = schema.NewAuthorizationBuilder().
			WithTenant(schema.Tenant{ID: adminTenantID, Name: "admin"}).
			WithRoleIDs("admin").
			BuildAdmin()
		memberAuth = schema.NewAuthorizationBuilder().
			WithTenant(schema.Tenant{ID: memberTenantID, Name: "demo"}).
			WithDomain(domainA).
			WithRoleIDs("Member").
			BuildScopedToTenant()
		domainScopedAuth = schema.NewAuthorizationBuilder().
			WithDomain(domainA).
			WithRoleIDs("Member").
			BuildScopedToDomain()

		auth = adminAuth
		context = middleware.Context{
			"context": ctx,
		}
		events = map[string]string{}
		timeLimit = time.Duration(10) * time.Second
		timeLimits = []*schema.PathEventTimeLimit{}
	})

	environmentManager := extension.GetManager()

	setupAuthContext := func(context middleware.Context, auth schema.Authorization, path, action string) {
		policy, role := manager.PolicyValidate(action, path, auth)
		Expect(policy).ToNot(BeNil())
		context["policy"] = policy
		context["role"] = role
		context["tenant_id"] = auth.TenantID()
		context["auth"] = auth
	}

	setupTestResources := func() {
		adminResourceData = map[string]interface{}{
			"id":           adminResourceID,
			"tenant_id":    adminTenantID,
			"domain_id":    domainAID,
			"test_string":  "Steloj estas en ordo.",
			"test_number":  0.5,
			"test_integer": 1,
			"test_bool":    false,
		}
		memberResourceData = map[string]interface{}{
			"id":           memberResourceID,
			"tenant_id":    memberTenantID,
			"domain_id":    domainAID,
			"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
			"test_number":  0.5,
			"test_integer": 1,
			"test_bool":    false,
		}
		otherDomainResourceData = map[string]interface{}{
			"id":           otherDomainResourceID,
			"tenant_id":    otherDomainTenantID,
			"domain_id":    domainBID,
			"test_string":  "Mi estas tekruĉo.",
			"test_number":  0.5,
			"test_integer": 1,
			"test_bool":    false,
		}
	}

	createTestResources := func() {
		adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
		Expect(err).NotTo(HaveOccurred())
		memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
		Expect(err).NotTo(HaveOccurred())
		otherDomainResource, err := manager.LoadResource(currentSchema.ID, otherDomainResourceData)
		Expect(err).NotTo(HaveOccurred())
		transaction, err := testDB.BeginTx()
		Expect(err).NotTo(HaveOccurred())
		defer transaction.Close()
		Expect(transaction.Create(ctx, adminResource)).To(Succeed())
		Expect(transaction.Create(ctx, memberResource)).To(Succeed())
		Expect(transaction.Create(ctx, otherDomainResource)).To(Succeed())
		Expect(transaction.Commit()).To(Succeed())
	}

	setupAndCreateTestResources := func() {
		setupTestResources()
		createTestResources()
	}

	JustBeforeEach(func() {
		var ok bool
		currentSchema, ok = manager.Schema(schemaID)
		Expect(ok).To(BeTrue())

		path = currentSchema.GetPluralURL()

		setupAuthContext(context, auth, path, action)

		env = otto.NewEnvironment("resource_management_test", testDB, &middleware.FakeIdentity{}, testSync)
		extensions = []*schema.Extension{}
		for event, javascript := range events {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id":   event + "_extension",
				"code": `gohan_register_handler("` + event + `", function(context) {` + javascript + `});`,
				"path": path,
			})
			Expect(err).ToNot(HaveOccurred())
			extensions = append(extensions, extension)
		}
		Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, path)).To(Succeed())
		environmentManager.RegisterEnvironment(schemaID, env)
		environmentManager.RegisterEnvironment("network", env)
		environmentManager.RegisterEnvironment("nil_test", env)
	})

	AfterEach(func() {
		Expect(db.WithinTx(testDB, func(tx transaction.Transaction) error {

			environmentManager.UnRegisterEnvironment(schemaID)
			environmentManager.UnRegisterEnvironment("network")
			environmentManager.UnRegisterEnvironment("nil_test")
			for _, schema := range schema.GetManager().Schemas() {
				if whitelist[schema.ID] {
					continue
				}
				Expect(dbutil.ClearTable(ctx, tx, schema)).ToNot(HaveOccurred(), "Failed to clear table.")
			}
			return nil
		})).ToNot(HaveOccurred(), "Failed to create or commit transaction.")
	})

	Describe("Getting a schema", func() {
		BeforeEach(func() {
			schemaID = "schema"
			action = "read"
		})

		Context("As an admin", func() {
			It("Should always return the full schema", func() {
				for _, currentSchema := range manager.OrderedSchemas() {
					if currentSchema.IsAbstract() {
						continue
					}
					trimmedSchema := server.TrimmedResource(currentSchema, auth)
					rawSchema := currentSchema.JSON()
					fullSchema := schema.NewResource(currentSchema, rawSchema)
					Expect(trimmedSchema).To(util.MatchAsJSON(fullSchema))
				}
			})
		})

		Context("As a member", func() {
			BeforeEach(func() {
				auth = memberAuth
			})

			It("Should return full schema when appropriate", func() {
				By("Fetching the schema")
				schemaSchema, ok := manager.Schema("schema")
				Expect(ok).To(BeTrue())
				By("The schema being equal to the full schema")
				trimmedSchema := server.TrimmedResource(currentSchema, auth)
				Expect(trimmedSchema).NotTo(BeNil())
				rawSchema := schemaSchema.JSON()
				fullSchema := schema.NewResource(currentSchema, rawSchema)
				Expect(trimmedSchema).To(util.MatchAsJSON(fullSchema))
			})

			It("Should return trimmed schema when appropriate", func() {
				By("Fetching the schema")
				networkSchema, ok := manager.Schema("network")
				Expect(ok).To(BeTrue())
				trimmedSchema := server.TrimmedResource(networkSchema, auth)
				Expect(trimmedSchema).NotTo(BeNil())
				theSchema, ok := trimmedSchema.Get("schema").(map[string]interface{})
				Expect(ok).To(BeTrue())
				properties, ok := theSchema["properties"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				By("Leaving allowed properties")
				_, ok = properties["id"]
				Expect(ok).To(BeTrue())
				_, ok = properties["name"]
				Expect(ok).To(BeTrue())
				_, ok = properties["description"]
				Expect(ok).To(BeTrue())
				_, ok = properties["tenant_id"]
				Expect(ok).To(BeTrue())

				By("Removing other properties")
				_, ok = properties["providor_networks"]
				Expect(ok).To(BeFalse())
				_, ok = properties["route_targets"]
				Expect(ok).To(BeFalse())
				_, ok = properties["shared"]
				Expect(ok).To(BeFalse())

				By("Adding schema permission")
				permission, ok := theSchema["permission"].([]string)
				Expect(ok).To(BeTrue())
				Expect(permission).To(Equal([]string{"create", "read", "update", "delete"}))
			})

			It("Should return no schema when appropriate", func() {
				By("Not fetching the schema")
				testSchema, ok := manager.Schema("admin_only")
				Expect(ok).To(BeTrue())
				trimmedSchema := server.TrimmedResource(testSchema, auth)
				Expect(trimmedSchema).To(BeNil())
			})
		})
	})

	Describe("Listing resources", func() {
		BeforeEach(func() {
			schemaID = "test"
			action = "read"
		})

		Describe("When there are no resources in the database", func() {
			It("Should return an empty list", func() {
				err := resources.GetMultipleResources(
					context, testDB, currentSchema, map[string][]string{})
				result, _ := context["response"].(map[string]interface{})
				number := context["total"].(uint64)
				Expect(err).NotTo(HaveOccurred())
				Expect(err).NotTo(HaveOccurred())
				Expect(number).To(Equal(uint64(0)))
				Expect(result).To(HaveKeyWithValue("tests", BeEmpty()))
			})

			It("Should fail if invalid filter is specified", func() {
				err := resources.GetMultipleResources(
					context, testDB, currentSchema, map[string][]string{"asd": []string{"asd"}})
				Expect(err).To(HaveOccurred())
			})

			Describe("With extensions", func() {
				Context("Only pre_list", func() {
					BeforeEach(func() {
						events["pre_list"] = `context["response"] = {"respondo": "bona"};`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						result, _ := context["response"].(map[string]interface{})
						number, _ := context["total"].(uint64)

						Expect(err).NotTo(HaveOccurred())
						Expect(number).To(Equal(uint64(0)))
						Expect(result).To(HaveKeyWithValue("respondo", "bona"))
					})
				})

				Context("Only pre_list_in_transaction", func() {
					BeforeEach(func() {
						events["pre_list_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_list_in_transaction", func() {
					BeforeEach(func() {
						events["post_list_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_list", func() {
					BeforeEach(func() {
						events["post_list"] = `context["response"] = {"respondo": "tre bona"};`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						result := context["response"].(map[string]interface{})
						number := context["total"].(uint64)
						Expect(err).NotTo(HaveOccurred())
						Expect(number).To(Equal(uint64(0)))
						Expect(result).To(HaveKeyWithValue("respondo", "tre bona"))
					})
				})

				Context("With pre_list throwing exception", func() {
					BeforeEach(func() {
						events["pre_list"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("With post_list throwing exception", func() {
					BeforeEach(func() {
						events["post_list"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("With pre_list returning invalid JSON", func() {
					BeforeEach(func() {
						events["pre_list"] = `context["response"] = "erara";`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(MatchError(HavePrefix("extension returned invalid JSON:")))
					})
				})

				Context("With post_list returning invalid JSON", func() {
					BeforeEach(func() {
						events["post_list"] = `context["response"] = "erara";`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(MatchError(HavePrefix("extension returned invalid JSON:")))
					})
				})

				Context("With post_list returning no response", func() {
					BeforeEach(func() {
						events["post_list"] = `delete context["response"];`
					})

					It("Should run the extension", func() {
						err := resources.GetMultipleResources(
							context, testDB, currentSchema, map[string][]string{})
						Expect(err).To(MatchError("No response"))
					})
				})
			})
		})

		Describe("When there are resources in the database", func() {
			JustBeforeEach(setupAndCreateTestResources)

			Context("As an admin", func() {
				It("Should return a filled list", func() {
					err := resources.GetMultipleResources(
						context, testDB, currentSchema, map[string][]string{})
					result := context["response"].(map[string]interface{})
					number := context["total"].(uint64)
					Expect(err).NotTo(HaveOccurred())
					Expect(number).To(Equal(uint64(3)))
					Expect(result).To(HaveKeyWithValue("tests", ConsistOf(adminResourceData, memberResourceData, otherDomainResourceData)))
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should return a only owned resources", func() {
					err := resources.GetMultipleResources(
						context, testDB, currentSchema, map[string][]string{})

					result := context["response"].(map[string]interface{})
					number := context["total"].(uint64)
					Expect(err).NotTo(HaveOccurred())
					Expect(number).To(Equal(uint64(1)))
					Expect(result).To(HaveKeyWithValue("tests", ConsistOf(memberResourceData)))
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				It("Should return resources only from the scoped domain", func() {
					err := resources.GetMultipleResources(
						context, testDB, currentSchema, map[string][]string{})

					result := context["response"].(map[string]interface{})
					number := context["total"].(uint64)
					Expect(err).NotTo(HaveOccurred())
					Expect(number).To(Equal(uint64(2)))
					Expect(result).To(HaveKeyWithValue("tests", ConsistOf(adminResourceData, memberResourceData)))
				})
			})
		})
	})

	Describe("Showing a resource", func() {
		BeforeEach(func() {
			schemaID = "test"
			action = "read"
		})

		Describe("When there are no resources in the database", func() {
			It("Should return an informative error", func() {
				err := resources.GetSingleResource(
					context, testDB, currentSchema, adminResourceID)
				Expect(err).To(HaveOccurred())
				_, ok := err.(resources.ResourceError)
				Expect(ok).To(BeTrue())
			})

			Describe("With extensions", func() {
				Context("Only pre_show", func() {
					BeforeEach(func() {
						events["pre_show"] = `context["response"] = {"respondo": "bona"};`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						result := context["response"].(map[string]interface{})
						Expect(err).NotTo(HaveOccurred())
						Expect(result).To(HaveKeyWithValue("respondo", "bona"))
					})
				})

				Context("Only pre_show_in_transaction", func() {
					BeforeEach(func() {
						events["pre_show_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_show_in_transaction", func() {
					BeforeEach(func() {
						events["post_show_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})

				Context("Only post_show", func() {
					BeforeEach(func() {
						events["post_show"] = `context["response"] = {"respondo": "tre bona"};`
					})

					It("Should not run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})

				Context("With pre_show throwing exception", func() {
					BeforeEach(func() {
						events["pre_show"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

			})
		})

		Describe("When there are resources in the database", func() {
			JustBeforeEach(setupAndCreateTestResources)

			Context("As an admin", func() {
				It("Should return owned resource", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, adminResourceID)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(adminResourceData)))
				})

				It("Should return not owned resource from the same domain", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, memberResourceID)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(memberResourceData)))
				})

				It("Should return resource from other domain", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, otherDomainResourceID)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(otherDomainResourceData)))
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should return owned resource", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, memberResourceID)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(memberResourceData)))
				})

				It("Should not return not owned resource from the same domain", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, adminResourceID)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})

				It("Should not return resource from other domain", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, otherDomainResourceID)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				DescribeTable("Should return resource from the same domain",
					func(resourceID string, data *map[string]interface{}) {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, resourceID)
						Expect(err).ToNot(HaveOccurred())
						result := context["response"].(map[string]interface{})
						Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(*data)))
					},
					Entry("Resource 1", adminResourceID, &adminResourceData),
					Entry("Resource 2", memberResourceID, &memberResourceData),
				)

				It("Should not return resource from other domain", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, otherDomainResourceID)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Describe("With extensions", func() {
				Context("Only pre_show", func() {
					BeforeEach(func() {
						events["pre_show"] = `context["response"] = {"respondo": "bona"};`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						result := context["response"].(map[string]interface{})
						Expect(err).NotTo(HaveOccurred())
						Expect(result).To(HaveKeyWithValue("respondo", "bona"))
					})
				})

				Context("Only pre_show_in_transaction", func() {
					BeforeEach(func() {
						events["pre_show_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_show_in_transaction", func() {
					BeforeEach(func() {
						events["post_show_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_show", func() {
					BeforeEach(func() {
						events["post_show"] = `context["response"] = {"respondo": "tre bona"};`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						result := context["response"].(map[string]interface{})
						Expect(err).NotTo(HaveOccurred())
						Expect(result).To(HaveKeyWithValue("respondo", "tre bona"))
					})
				})

				Context("With pre_show throwing exception", func() {
					BeforeEach(func() {
						events["pre_show"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("With post_show throwing exception", func() {
					BeforeEach(func() {
						events["post_show"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.GetSingleResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

			})
		})
	})

	Describe("Deleting a resource", func() {
		BeforeEach(func() {
			schemaID = "test"
			action = "delete"
		})

		Describe("When there are no resources in the database", func() {
			It("Should return an informative error", func() {
				err := resources.DeleteResource(
					context, testDB, currentSchema, adminResourceID)
				Expect(err).To(HaveOccurred())
				_, ok := err.(resources.ResourceError)
				Expect(ok).To(BeTrue())
			})

			Describe("With extensions", func() {
				Context("Only pre_delete", func() {
					BeforeEach(func() {
						events["pre_delete"] = `throw new CustomException("bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only pre_delete_in_transaction", func() {
					BeforeEach(func() {
						events["pre_delete_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})

				Context("Only post_delete_in_transaction", func() {
					BeforeEach(func() {
						events["post_delete_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})

				Context("Only post_delete", func() {
					BeforeEach(func() {
						events["post_delete"] = `throw new CustomException("tre bona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})
			})
		})

		Describe("When there are resources in the database", func() {
			JustBeforeEach(setupAndCreateTestResources)

			Context("As an admin", func() {
				It("Should delete owned resource", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, adminResourceID)).To(Succeed())
				})

				It("Should delete not owned resource", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, memberResourceID)).To(Succeed())
				})

				It("Should delete resource from other domain", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, otherDomainResourceID)).To(Succeed())
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should delete owned resource", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, memberResourceID)).To(Succeed())
				})

				It("Should not delete not owned resource", func() {
					err := resources.DeleteResource(
						context, testDB, currentSchema, adminResourceID)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})

				It("Should not delete resource from other domain", func() {
					err := resources.DeleteResource(
						context, testDB, currentSchema, otherDomainResourceID)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				It("Should delete resource from the same domain", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, adminResourceID)).To(Succeed())
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, memberResourceID)).To(Succeed())
				})

				It("Should not delete resource from other domain", func() {
					err := resources.DeleteResource(
						context, testDB, currentSchema, otherDomainResourceID)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Describe("With extensions", func() {
				Context("Only pre_delete", func() {
					BeforeEach(func() {
						events["pre_delete"] = `throw new CustomException("bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only pre_delete_in_transaction", func() {
					BeforeEach(func() {
						events["pre_delete_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_delete_in_transaction", func() {
					BeforeEach(func() {
						events["post_delete_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_delete", func() {
					BeforeEach(func() {
						events["post_delete"] = `throw new CustomException("tre bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.DeleteResource(
							context, testDB, currentSchema, adminResourceID)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})
			})
		})
	})

	Describe("Creating a resource", func() {
		var (
			fakeIdentity middleware.IdentityService
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
			setupTestResources()
			fakeIdentity = &middleware.FakeIdentity{}
		})

		Describe("When there are no resources in the database", func() {
			Context("As an admin", func() {
				It("Should create own resource", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, adminResourceData)

					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(adminResourceData)))
				})

				It("Should create not own resource", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, memberResourceData)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(memberResourceData)))
				})

				It("Should fill in an id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"tenant_id": adminTenantID, "domain_id": domainAID})
					Expect(err).NotTo(HaveOccurred())
					result := context["response"].(map[string]interface{})
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("domain_id", domainAID))
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKey("id"))
				})

				It("Should fill in the tenant_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"id": adminResourceID, "domain_id": domainAID})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("domain_id", domainAID))
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKeyWithValue("id", adminResourceID))
				})

				It("Should fill in the domain_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"id": adminResourceID, "tenant_id": adminTenantID})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("domain_id", schema.DefaultDomain.ID))
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKeyWithValue("id", adminResourceID))
				})

				It("Should fill in id, tenant_id and domain_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{})
					Expect(err).NotTo(HaveOccurred())
					result := context["response"].(map[string]interface{})
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("domain_id", schema.DefaultDomain.ID))
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKey("id"))
				})

				It("Should replace empty id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{
							"id": "",
						})
					Expect(err).NotTo(HaveOccurred())
					result := context["response"].(map[string]interface{})
					theResource, ok := result[schemaID]
					resource := theResource.(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(resource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(resource).To(HaveKey("id"))
					_, err = uuid.Parse(resource["id"].(string))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should create own resource", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, memberResourceData)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(memberResourceData)))
				})

				It("Should not create not own resource", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, adminResourceData)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})

				It("Should not create resource in other domain", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, otherDomainResourceData)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				DescribeTable("Should create resource in current domain",
					func(resourceData *map[string]interface{}) {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, *resourceData)
						result := context["response"].(map[string]interface{})
						Expect(err).NotTo(HaveOccurred())
						Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(*resourceData)))
					},
					Entry("Resource 1", &adminResourceData),
					Entry("Resource 2", &memberResourceData),
				)

				It("Should not create resource in other domain", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, otherDomainResourceData)
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})

				It("Should fill in the domain_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"id": adminResourceID, "tenant_id": adminTenantID})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("domain_id", domainAID))
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKeyWithValue("id", adminResourceID))
				})

				It("Should not create resource without tenant_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"id": adminResourceID})
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Describe("With extensions", func() {
				Context("Only pre_create", func() {
					BeforeEach(func() {
						events["pre_create"] = `throw new CustomException("bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceData)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only pre_create_in_transaction", func() {
					BeforeEach(func() {
						events["pre_create_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceData)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_create_in_transaction", func() {
					BeforeEach(func() {
						events["post_create_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceData)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_create", func() {
					BeforeEach(func() {
						events["post_create"] = `throw new CustomException("tre bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceData)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

			})
		})

		Describe("When there are resources in the database", func() {
			JustBeforeEach(setupAndCreateTestResources)

			It("Should not create duplicate resource", func() {
				err := resources.CreateResource(
					context, testDB, fakeIdentity, currentSchema, adminResourceData)
				Expect(err).To(HaveOccurred())
				_, ok := err.(resources.ResourceError)
				Expect(ok).To(BeTrue())
			})
		})
	})

	Describe("Updating a resource", func() {
		var (
			fakeIdentity middleware.IdentityService
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"

			setupTestResources()

			fakeIdentity = &middleware.FakeIdentity{}
		})

		Describe("When there are no resources in the database", func() {
			It("Should return an informative error", func() {
				err := resources.UpdateResource(
					context, testDB, fakeIdentity, currentSchema, adminResourceID, adminResourceData)
				Expect(err).To(HaveOccurred())
				_, ok := err.(resources.ResourceError)
				Expect(ok).To(BeTrue())
			})

			Describe("With extensions", func() {
				Context("Only pre_update", func() {
					BeforeEach(func() {
						events["pre_update"] = `throw new CustomException("bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID, adminResourceData)
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only pre_update_in_transaction", func() {
					BeforeEach(func() {
						events["pre_update_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID, adminResourceData)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})

				Context("Only post_update_in_transaction", func() {
					BeforeEach(func() {
						events["post_update_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID, adminResourceData)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})

				Context("Only post_update", func() {
					BeforeEach(func() {
						events["post_update"] = `throw new CustomException("tre bona", 390);`
					})

					It("Should not run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID, adminResourceData)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})
			})
		})
		Describe("Whether id is not empty during update and update_in_transaction", func() {
			BeforeEach(func() {
				javascriptCode := `if (context.resource.id === undefined || context.resource.id === ""){
					throw new CustomException();
				}`

				events["pre_update"] = javascriptCode
				events["pre_update_in_transaction"] = javascriptCode
			})
			It("Should receive id, tenant_id and domain_id but should not update them", func() {
				err := resources.CreateResource(
					context, testDB, fakeIdentity, currentSchema, adminResourceData)
				Expect(err).To(Succeed())
				delete(adminResourceData, "id")
				delete(adminResourceData, "tenant_id")
				delete(adminResourceData, "domain_id")

				err = resources.UpdateResource(
					context, testDB, fakeIdentity, currentSchema, adminResourceID, adminResourceData)
				Expect(err).To(Succeed())

			})
		})

		Describe("When there are resources in the database", func() {
			JustBeforeEach(setupAndCreateTestResources)

			Context("As an admin", func() {
				var (
					adminNetworkData, adminNetworkUpdate map[string]interface{}
				)

				BeforeEach(func() {
					adminNetworkData = map[string]interface{}{
						"id":            memberResourceID,
						"route_targets": []interface{}{"routeTarget1"},
					}

					adminNetworkUpdate = map[string]interface{}{
						"config": map[string]interface{}{
							"default_vlan": map[string]interface{}{
								"vlan_id": 5,
							},
						},
					}
				})

				DescribeTable("Should update",
					func(resourceID string) {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, resourceID,
							map[string]interface{}{"test_string": "Ia, ia, HJPEV fhtang!"})
						result := context["response"].(map[string]interface{})
						Expect(err).NotTo(HaveOccurred())
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_string", "Ia, ia, HJPEV fhtang!"))
					},
					Entry("Own resource", adminResourceID),
					Entry("Not own resource, same domain", memberResourceID),
					Entry("Resource form other domain", otherDomainResourceID),
				)

				It("Should only modify updated fields in subobjects", func() {
					networkSchema, _ := manager.Schema("network")

					Expect(resources.CreateResource(context, testDB, fakeIdentity, networkSchema, adminNetworkData)).To(Succeed())

					result := context["response"].(map[string]interface{})
					network, found := result["network"].(map[string]interface{})
					Expect(found).To(BeTrue())
					config, found := network["config"].(map[string]interface{})
					Expect(found).To(BeTrue())
					defaultVlan, found := config["default_vlan"].(map[string]interface{})
					Expect(found).To(BeTrue())
					vlanName := defaultVlan["name"]
					Expect(vlanName).To(Equal("default_vlan"))
					vlanId := defaultVlan["vlan_id"]
					Expect(vlanId).To(Equal(1)) // default vlan_id value

					Expect(resources.UpdateResource(context, testDB, fakeIdentity, networkSchema, memberResourceID, adminNetworkUpdate)).To(Succeed())

					result = context["response"].(map[string]interface{})
					network, found = result["network"].(map[string]interface{})
					Expect(found).To(BeTrue())
					config, found = network["config"].(map[string]interface{})
					Expect(found).To(BeTrue())
					defaultVlan, found = config["default_vlan"].(map[string]interface{})
					Expect(found).To(BeTrue())
					vlanName = defaultVlan["name"]
					Expect(vlanName).To(Equal("default_vlan"))
					vlanId = defaultVlan["vlan_id"]
					Expect(vlanId).To(Equal(5))
				})

				It("Should properly update array values", func() {
					networkSchema, _ := manager.Schema("network")

					Expect(resources.CreateResource(context, testDB, fakeIdentity, networkSchema, adminNetworkData)).To(Succeed())

					result := context["response"].(map[string]interface{})
					network, found := result["network"].(map[string]interface{})
					Expect(found).To(BeTrue())
					routeTargets := network["route_targets"]
					Expect(routeTargets).To(Equal([]interface{}{"routeTarget1"}))

					Expect(resources.UpdateResource(context, testDB, fakeIdentity, networkSchema, memberResourceID, map[string]interface{}{
						"route_targets": []interface{}{"testTarget2", "testTarget3"},
					})).To(Succeed())

					result = context["response"].(map[string]interface{})
					network, found = result["network"].(map[string]interface{})
					Expect(found).To(BeTrue())
					Expect(network["route_targets"]).To(Equal([]interface{}{"testTarget2", "testTarget3"}))
				})

				It("Should properly update nil objects", func() {
					testSchema, _ := manager.Schema("nil_test")
					Expect(resources.CreateResource(context, testDB, fakeIdentity, testSchema, map[string]interface{}{"id": memberResourceID})).To(Succeed())
					result := context["response"].(map[string]interface{})
					mainObject, found := result["nil_test"].(map[string]interface{})
					Expect(found).To(BeTrue())
					subObject := mainObject["nested_obj"]
					Expect(subObject).To(BeNil())

					Expect(resources.UpdateResource(context, testDB, fakeIdentity, testSchema, memberResourceID, map[string]interface{}{
						"nested_obj": map[string]interface{}{
							"nested_string": "nestedString",
						}})).To(Succeed())
					result = context["response"].(map[string]interface{})
					mainObject, found = result["nil_test"].(map[string]interface{})
					Expect(found).To(BeTrue())
					subObject, found = mainObject["nested_obj"]

					Expect(found).To(BeTrue())
					Expect(subObject.(map[string]interface{})["nested_string"]).To(Equal("nestedString"))
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should update own resource", func() {
					err := resources.UpdateResource(
						context, testDB, fakeIdentity, currentSchema, memberResourceID,
						map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
					Expect(err).NotTo(HaveOccurred())
					result := context["response"].(map[string]interface{})
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
				})

				It("Should not update not own resource", func() {
					err := resources.UpdateResource(
						context, testDB, fakeIdentity, currentSchema, adminResourceID,
						map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"})
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
					resource, _ := context["response"].(map[string]interface{})
					Expect(resource).To(BeNil())
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				DescribeTable("Should update resource in current domain",
					func(resourceID string) {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, resourceID,
							map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
						Expect(err).NotTo(HaveOccurred())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
					},
					Entry("Resource 1", adminResourceID),
					Entry("Resource 2", memberResourceID),
				)

				It("Should not update resource from other domain", func() {
					err := resources.UpdateResource(
						context, testDB, fakeIdentity, currentSchema, otherDomainResourceID,
						map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"})
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
					resource, _ := context["response"].(map[string]interface{})
					Expect(resource).To(BeNil())
				})
			})

			Describe("With extensions", func() {
				Context("pre_create inserts zero-value", func() {
					var (
						requestMap map[string]interface{}
					)
					BeforeEach(func() {
						requestMap = map[string]interface{}{
							"test_string": "Steloj ne estas en ordo.",
							"test_bool":   true,
						}
						events["pre_create"] = `if (context.run_precreate) { context.resource.test_string = "test123"; }`
						context["go_validation"] = true // simulate Goext
						context["request_data"] = requestMap
					})
					It("should create non-empty resource", func() {
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.CreateResource(context, testDB, fakeIdentity, currentSchema, requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", true))
						Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
					})

					It("should create empty resource", func() {
						delete(requestMap, "test_bool")
						delete(requestMap, "test_string")
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.CreateResource(context, testDB, fakeIdentity, currentSchema, requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", false))
						Expect(theResource).To(HaveKeyWithValue("test_string", ""))
					})

					It("should set request data in precreate", func() {
						delete(requestMap, "test_bool")
						delete(requestMap, "test_string")
						context["run_precreate"] = true
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.CreateResource(context, testDB, fakeIdentity, currentSchema, requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", false))
						Expect(theResource).To(HaveKeyWithValue("test_string", "test123"))
					})

					It("should override request data in precreate", func() {
						context["run_precreate"] = true
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.CreateResource(context, testDB, fakeIdentity, currentSchema, requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", true))
						Expect(theResource).To(HaveKeyWithValue("test_string", "test123"))
					})
				})
				Context("pre_update inserts zero-value", func() {
					var (
						requestMap map[string]interface{}
					)
					BeforeEach(func() {
						requestMap = map[string]interface{}{
							"test_string": "Steloj ne estas en ordo.",
							"test_bool":   true,
						}
						events["pre_update"] = `if (context.run_preupdate) { context.resource.test_bool = false; context.resource.test_string = ""; }`
						context["go_validation"] = true // simulate Goext
						context["request_data"] = requestMap
					})
					JustBeforeEach(func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							requestMap)
						result := context["response"].(map[string]interface{})
						Expect(err).NotTo(HaveOccurred())
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
						Expect(theResource).To(HaveKeyWithValue("test_bool", true))
					})

					It("should not update data absent in request", func() {
						delete(requestMap, "test_bool")
						delete(requestMap, "test_string")
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", true))
						Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
					})

					It("should update zero-value data", func() {
						requestMap["test_bool"] = false
						requestMap["test_string"] = ""
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", false))
						Expect(theResource).To(HaveKeyWithValue("test_string", ""))
					})

					It("should update data in preupate", func() {
						delete(requestMap, "test_bool")
						delete(requestMap, "test_string")
						context["run_preupdate"] = true
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", false))
						Expect(theResource).To(HaveKeyWithValue("test_string", ""))
					})

					It("should override request data in preupate", func() {
						context["run_preupdate"] = true
						context["request_data"] = deepcopy.Copy(requestMap).(map[string]interface{})

						Expect(resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							requestMap)).To(Succeed())
						result := context["response"].(map[string]interface{})
						theResource, ok := result[schemaID]
						Expect(ok).To(BeTrue())
						Expect(theResource).To(HaveKeyWithValue("test_bool", false))
						Expect(theResource).To(HaveKeyWithValue("test_string", ""))
					})
				})
				Context("Validation", func() {
					var (
						requestMap map[string]interface{}
					)
					BeforeEach(func() {
						requestMap = map[string]interface{}{
							"test_string": "123",
						}
						context["go_validation"] = true // simulate Goext
						events["pre_update"] = `if (context.resource.test_string === undefined) { context.resource.test_string = "12345678901234567890123456789012345678901"; }`
						events["pre_create"] = `if (context.resource.test_string === undefined) { context.resource.test_string = "12345678901234567890123456789012345678901"; }`
					})
					It("should run validation after pre_create", func() {
						err := resources.CreateResource(context, testDB, fakeIdentity, currentSchema, map[string]interface{}{})
						Expect(err).To(MatchError("Json validation error:\n\ttest_string: String length must be less than or equal to 40,"))

						Expect(resources.CreateResource(context, testDB, fakeIdentity, currentSchema, requestMap)).To(Succeed())
					})
					It("should run validation after pre_update", func() {
						Expect(resources.CreateResource(context, testDB, fakeIdentity, currentSchema, deepcopy.Copy(requestMap).(map[string]interface{}))).To(Succeed())

						err := resources.UpdateResource(context, testDB, fakeIdentity, currentSchema, adminResourceID, map[string]interface{}{})
						Expect(err).To(MatchError("Json validation error:\n\ttest_string: String length must be less than or equal to 40,"))

						Expect(resources.UpdateResource(context, testDB, fakeIdentity, currentSchema, adminResourceID, requestMap)).To(Succeed())
					})
				})

				Context("Only pre_update", func() {
					BeforeEach(func() {
						events["pre_update"] = `throw new CustomException("bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only pre_update_in_transaction", func() {
					BeforeEach(func() {
						events["pre_update_in_transaction"] = `throw new CustomException("malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_update_in_transaction", func() {
					BeforeEach(func() {
						events["post_update_in_transaction"] = `throw new CustomException("tre malbona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre malbona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

				Context("Only post_update", func() {
					BeforeEach(func() {
						events["post_update"] = `throw new CustomException("tre bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, adminResourceID,
							map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
						Expect(err).To(HaveOccurred())
						extErr, ok := err.(extension.Error)
						Expect(ok).To(BeTrue())
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "tre bona"))
						Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
					})
				})

			})
		})
	})

	Describe("Running an action on a resource", func() {
		var (
			fakeIdentity           middleware.IdentityService
			fakeAction             schema.Action
			fakeActionWithoutInput schema.Action
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
			setupTestResources()
			fakeIdentity = &middleware.FakeIdentity{}
			inputSchema := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
					},
					"delay": map[string]interface{}{
						"type": "string",
					},
				},
			}
			fakeAction = schema.NewAction("fake_action", "GET", "/:id/whatever", "", inputSchema, nil, nil)
			fakeActionWithoutInput = schema.NewAction("fake_action", "GET", "/:id/whatever", "", nil, nil, nil)
		})

		// Actions do not care resource existence or tenant ownership
		Describe("With extension", func() {
			BeforeEach(func() {
				events["fake_action"] = `throw new CustomException("malbona", 390);`
			})

			It("Should run the extension", func() {
				err := resources.ActionResource(
					context, currentSchema, fakeAction, adminResourceID,
					map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
				Expect(err).To(HaveOccurred())
				extErr, ok := err.(extension.Error)
				Expect(ok).To(BeTrue())
				Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
				Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
				Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
			})

			Context("Without input shcema", func() {
				It("Should run the extension", func() {
					err := resources.ActionResource(
						context, currentSchema, fakeActionWithoutInput, adminResourceID,
						map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
					Expect(err).To(HaveOccurred())
					extErr, ok := err.(extension.Error)
					Expect(ok).To(BeTrue())
					Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("name", "CustomException"))
					Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("message", "malbona"))
					Expect(extErr.ExceptionInfo).To(HaveKeyWithValue("code", int64(390)))
				})
			})
		})
	})

	Describe("Executing a sequence of operations", func() {
		var (
			listContext, showContext, deleteContext, createContext, updateContext middleware.Context
			fakeIdentity                                                          middleware.IdentityService
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "list"
			setupTestResources()
			listContext = makeContext()
			showContext = makeContext()
			deleteContext = makeContext()
			createContext = makeContext()
			updateContext = makeContext()
			fakeIdentity = &middleware.FakeIdentity{}
		})

		JustBeforeEach(func() {
			setupAuthContext(listContext, auth, path, "list")
			setupAuthContext(showContext, auth, path, "show")
			setupAuthContext(deleteContext, auth, path, "delete")
			setupAuthContext(createContext, auth, path, "create")
			setupAuthContext(updateContext, auth, path, "update")
		})

		It("Should behave as expected", func() {
			By("Showing nothing in an empty database")
			err := resources.GetMultipleResources(
				listContext, testDB, currentSchema, map[string][]string{})

			result, _ := listContext["response"].(map[string]interface{})
			number, _ := listContext["total"].(uint64)
			Expect(err).NotTo(HaveOccurred())
			Expect(number).To(Equal(uint64(0)))
			Expect(result).To(HaveKeyWithValue("tests", BeEmpty()))
			delete(listContext, "response")

			By("Successfully adding a resource to the database")
			err = resources.CreateResource(
				createContext, testDB, fakeIdentity, currentSchema, adminResourceData)
			result = createContext["response"].(map[string]interface{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(adminResourceData)))

			By("Listing the added resource")
			err = resources.GetMultipleResources(
				listContext, testDB, currentSchema, map[string][]string{})
			result = listContext["response"].(map[string]interface{})
			number = listContext["total"].(uint64)

			Expect(err).NotTo(HaveOccurred())
			Expect(number).To(Equal(uint64(1)))
			Expect(result).To(HaveKeyWithValue("tests", ConsistOf(util.MatchAsJSON(adminResourceData))))

			By("Updating the resource")
			err = resources.UpdateResource(
				updateContext, testDB, fakeIdentity, currentSchema, adminResourceID,
				map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
			result = updateContext["response"].(map[string]interface{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue(schemaID, HaveKeyWithValue("test_string", "Steloj ne estas en ordo.")))

			By("Showing the updated resource")
			err = resources.GetSingleResource(
				showContext, testDB, currentSchema, adminResourceID)
			result = showContext["response"].(map[string]interface{})
			By(fmt.Sprintf("%s", result))
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue(schemaID, HaveKeyWithValue("test_string", "Steloj ne estas en ordo.")))

			By("Deleting the resource")
			By(adminResourceID)
			Expect(resources.DeleteResource(
				deleteContext, testDB, currentSchema, adminResourceID)).To(Succeed())

			By("Again showing nothing in an empty database")
			delete(listContext, "response")
			err = resources.GetMultipleResources(
				listContext, testDB, currentSchema, map[string][]string{})
			result = listContext["response"].(map[string]interface{})
			By(fmt.Sprintf("%s", result))
			number = listContext["total"].(uint64)
			Expect(err).NotTo(HaveOccurred())
			Expect(number).To(Equal(uint64(0)))
			Expect(result).To(HaveKeyWithValue("tests", BeEmpty()))
		})
	})
})

func makeContext() middleware.Context {
	return middleware.Context{"context": context_pkg.Background()}
}
