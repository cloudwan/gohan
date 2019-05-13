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
		fakeIdentity            middleware.IdentityService
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

		fakeIdentity = &middleware.FakeIdentity{}
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

	create := func(tx transaction.Transaction, resource *schema.Resource) {
		_, err := tx.Create(ctx, resource)
		Expect(err).NotTo(HaveOccurred())
	}

	createTestResources := func() {
		adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
		Expect(err).NotTo(HaveOccurred())
		memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
		Expect(err).NotTo(HaveOccurred())
		otherDomainResource, err := manager.LoadResource(currentSchema.ID, otherDomainResourceData)
		Expect(err).NotTo(HaveOccurred())
		tx, err := testDB.BeginTx()
		Expect(err).NotTo(HaveOccurred())
		defer tx.Close()
		create(tx, adminResource)
		create(tx, memberResource)
		create(tx, otherDomainResource)
		Expect(tx.Commit()).To(Succeed())
	}

	setupAndCreateTestResources := func() {
		setupTestResources()
		createTestResources()
	}

	createAndExpectSuccess := func(resourceData map[string]interface{}) interface{} {
		err := resources.CreateResource(
			context, testDB, fakeIdentity, currentSchema, resourceData)

		result := context["response"].(map[string]interface{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveKey(schemaID))
		return result[schemaID]
	}

	createAndExpectResourceError := func(resourceData map[string]interface{}) {
		err := resources.CreateResource(
			context, testDB, fakeIdentity, currentSchema, resourceData)
		Expect(err).To(HaveOccurred())
		_, ok := err.(resources.ResourceError)
		Expect(ok).To(BeTrue())
	}

	updateAndExpectSuccess := func(resourceID string, resourceData map[string]interface{}) interface{} {
		err := resources.UpdateResource(
			context, testDB, fakeIdentity, currentSchema, resourceID,
			resourceData)
		Expect(err).NotTo(HaveOccurred())
		result := context["response"].(map[string]interface{})
		theResource, ok := result[schemaID]
		Expect(ok).To(BeTrue())
		return theResource
	}

	updateAndExpectError := func(resourceID string, resourceData map[string]interface{}) {
		err := resources.UpdateResource(
			context, testDB, fakeIdentity, currentSchema, resourceID, resourceData)
		Expect(err).To(HaveOccurred())
		_, ok := err.(resources.ResourceError)
		Expect(ok).To(BeTrue())
		resource, _ := context["response"].(map[string]interface{})
		Expect(resource).To(BeNil())
	}

	deleteAndExpectSuccess := func(resourceID string) {
		Expect(resources.DeleteResource(
			context, testDB, currentSchema, resourceID)).To(Succeed())
	}

	deleteAndExpectResourceError := func(resourceID string) {
		err := resources.DeleteResource(
			context, testDB, currentSchema, resourceID)
		Expect(err).To(HaveOccurred())
		_, ok := err.(resources.ResourceError)
		Expect(ok).To(BeTrue())
	}

	JustBeforeEach(func() {
		var ok bool
		currentSchema, ok = manager.Schema(schemaID)
		Expect(ok).To(BeTrue())

		path = currentSchema.GetPluralURL()

		setupAuthContext(context, auth, path, action)

		env = otto.NewEnvironment("resource_management_test", testDB, fakeIdentity, testSync)
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

			getAndExpectResources := func(expectedResources ...interface{}) {
				schemaPluralName := schemaID + "s"

				err := resources.GetMultipleResources(
					context, testDB, currentSchema, map[string][]string{})
				result := context["response"].(map[string]interface{})
				number := context["total"].(uint64)
				Expect(err).NotTo(HaveOccurred())
				Expect(number).To(Equal(uint64(len(expectedResources))))
				Expect(result).To(HaveKeyWithValue(schemaPluralName, ConsistOf(expectedResources...)))
			}

			Context("As an admin", func() {
				It("Should return a filled list", func() {
					getAndExpectResources(adminResourceData, memberResourceData, otherDomainResourceData)
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should return a only owned resources", func() {
						getAndExpectResources(memberResourceData)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should return resources from whole domain", func() {
						getAndExpectResources(adminResourceData, memberResourceData)
					})
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should return resources only from the scoped domain", func() {
						getAndExpectResources(adminResourceData, memberResourceData)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should return resources only from the scoped domain", func() {
						getAndExpectResources(adminResourceData, memberResourceData)
					})
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

			getAndExpectResource := func(resourceID string, expectedResource map[string]interface{}) {
				err := resources.GetSingleResource(
					context, testDB, currentSchema, resourceID)
				Expect(err).NotTo(HaveOccurred())
				result := context["response"].(map[string]interface{})
				Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(expectedResource)))
			}

			getAndExpectResourceError := func(resourceID string) {
				err := resources.GetSingleResource(
					context, testDB, currentSchema, resourceID)
				Expect(err).To(HaveOccurred())
				_, ok := err.(resources.ResourceError)
				Expect(ok).To(BeTrue())
			}

			Context("As an admin", func() {
				BeforeEach(func() {
					schemaID = "test"
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})
					DescribeTable("Should be able to get",
						func(resourceID string, resourceData *map[string]interface{}) {
							getAndExpectResource(resourceID, *resourceData)
						},
						Entry("owned resource", adminResourceID, &adminResourceData),
						Entry("other tenant's resource from the same domain", memberResourceID, &memberResourceData),
						Entry("other tenant's resource from other domain", otherDomainResourceID, &otherDomainResourceData),
					)
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})
					DescribeTable("Should be able to get",
						func(resourceID string, resourceData *map[string]interface{}) {
							getAndExpectResource(resourceID, *resourceData)
						},
						Entry("owned resource", adminResourceID, &adminResourceData),
						Entry("other tenant's resource from the same domain", memberResourceID, &memberResourceData),
						Entry("other tenant's resource from other domain", otherDomainResourceID, &otherDomainResourceData),
					)
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should return owned resource", func() {
						getAndExpectResource(memberResourceID, memberResourceData)
					})

					It("Should not return not owned resource from the same domain", func() {
						getAndExpectResourceError(adminResourceID)
					})

					It("Should not return resource from other domain", func() {
						getAndExpectResourceError(otherDomainResourceID)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should return resource from the same domain", func() {
						getAndExpectResource(memberResourceID, memberResourceData)
					})

					It("Should return other tenant's resource from the same domain", func() {
						getAndExpectResource(adminResourceID, adminResourceData)
					})

					It("Should not return resource from other domain", func() {
						getAndExpectResourceError(otherDomainResourceID)
					})
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})
					DescribeTable("Should return resource from the same domain",
						func(resourceID string, data *map[string]interface{}) {
							getAndExpectResource(resourceID, *data)
						},
						Entry("Resource 1", adminResourceID, &adminResourceData),
						Entry("Resource 2", memberResourceID, &memberResourceData),
					)

					It("Should not return resource from other domain", func() {
						getAndExpectResourceError(otherDomainResourceID)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					DescribeTable("Should return resource from the same domain",
						func(resourceID string, data *map[string]interface{}) {
							getAndExpectResource(resourceID, *data)
						},
						Entry("Resource 1", adminResourceID, &adminResourceData),
						Entry("Resource 2", memberResourceID, &memberResourceData),
					)

					It("Should not return resource from other domain", func() {
						getAndExpectResourceError(otherDomainResourceID)
					})
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
				BeforeEach(func() {
					auth = adminAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})
					DescribeTable("Should be able to remove",
						deleteAndExpectSuccess,
						Entry("owned resource", adminResourceID),
						Entry("other tenant's resource from the same domain", memberResourceID),
						Entry("other tenant's resource from other domain", otherDomainResourceID),
					)
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})
					DescribeTable("Should be able to remove",
						deleteAndExpectSuccess,
						Entry("owned resource", adminResourceID),
						Entry("other tenant's resource from the same domain", memberResourceID),
						Entry("other tenant's resource from other domain", otherDomainResourceID),
					)
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should delete owned resource", func() {
						deleteAndExpectSuccess(memberResourceID)
					})

					It("Should not delete not owned resource", func() {
						deleteAndExpectResourceError(adminResourceID)
					})

					It("Should not delete resource from other domain", func() {
						deleteAndExpectResourceError(otherDomainResourceID)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should delete owned resource", func() {
						deleteAndExpectSuccess(memberResourceID)
					})

					It("Should delete other tenant's resource from the same domain", func() {
						deleteAndExpectSuccess(adminResourceID)
					})

					It("Should not delete resource from other domain", func() {
						deleteAndExpectResourceError(otherDomainResourceID)
					})
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should delete resource from the same domain", func() {
						deleteAndExpectSuccess(adminResourceID)
						deleteAndExpectSuccess(memberResourceID)
					})

					It("Should not delete resource from other domain", func() {
						deleteAndExpectResourceError(otherDomainResourceID)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should delete resource from the same domain", func() {
						deleteAndExpectSuccess(adminResourceID)
						deleteAndExpectSuccess(memberResourceID)
					})

					It("Should not delete resource from other domain", func() {
						deleteAndExpectResourceError(otherDomainResourceID)
					})
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
		JustBeforeEach(setupTestResources)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
		})

		Describe("When there are no resources in the database", func() {
			Context("As an admin", func() {
				BeforeEach(func() {
					auth = adminAuth
				})

				Context("Resources restricted with is_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should create own resource", func() {
						resource := createAndExpectSuccess(adminResourceData)
						Expect(resource).To(Equal(adminResourceData))
					})

					It("Should create not own resource", func() {
						resource := createAndExpectSuccess(memberResourceData)
						Expect(resource).To(Equal(memberResourceData))
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should create own resource", func() {
						resource := createAndExpectSuccess(adminResourceData)
						Expect(resource).To(Equal(adminResourceData))
					})

					It("Should create not own resource", func() {
						resource := createAndExpectSuccess(memberResourceData)
						Expect(resource).To(Equal(memberResourceData))
					})
				})

				It("Should fill in an id", func() {
					resourceData := map[string]interface{}{"tenant_id": adminTenantID, "domain_id": domainAID}
					createdResource := createAndExpectSuccess(resourceData)
					Expect(createdResource).To(HaveKeyWithValue("domain_id", domainAID))
					Expect(createdResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(createdResource).To(HaveKey("id"))
				})

				It("Should fill in the tenant_id", func() {
					resourceData := map[string]interface{}{
						"id":        adminResourceID,
						"domain_id": domainAID,
					}
					createdResource := createAndExpectSuccess(resourceData)
					Expect(createdResource).To(HaveKeyWithValue("domain_id", domainAID))
					Expect(createdResource).To(HaveKeyWithValue("tenant_id", auth.TenantID()))
					Expect(createdResource).To(HaveKeyWithValue("id", adminResourceID))
				})

				It("Should fill in the domain_id", func() {
					resourceData := map[string]interface{}{"id": adminResourceID, "tenant_id": adminTenantID}
					createdResourceData := createAndExpectSuccess(resourceData)
					Expect(createdResourceData).To(HaveKeyWithValue("domain_id", auth.DomainID()))
					Expect(createdResourceData).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(createdResourceData).To(HaveKeyWithValue("id", adminResourceID))
				})

				It("Should fill in id, tenant_id and domain_id", func() {
					createdResourceData := createAndExpectSuccess(map[string]interface{}{})
					Expect(createdResourceData).To(HaveKeyWithValue("domain_id", auth.DomainID()))
					Expect(createdResourceData).To(HaveKeyWithValue("tenant_id", auth.TenantID()))
					Expect(createdResourceData).To(HaveKey("id"))
				})

				It("Should replace empty id", func() {
					resourceData := map[string]interface{}{"id": ""}
					createdResourceData := createAndExpectSuccess(resourceData)
					Expect(createdResourceData).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(createdResourceData).To(HaveKey("id"))
					_, err := uuid.Parse(createdResourceData.(map[string]interface{})["id"].(string))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				Context("Resources restricted with is_owner", func() {
					It("Should create own resource", func() {
						createdResourceData := createAndExpectSuccess(memberResourceData)
						Expect(createdResourceData).To(Equal(memberResourceData))
					})

					It("Should not create not own resource", func() {
						createAndExpectResourceError(adminResourceData)
					})

					It("Should not create resource in other domain", func() {
						createAndExpectResourceError(otherDomainResourceData)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should create own resource", func() {
						createdResourceData := createAndExpectSuccess(memberResourceData)
						Expect(createdResourceData).To(Equal(memberResourceData))
					})

					It("Should create resource for another tenant in the same domain", func() {
						createdResourceData := createAndExpectSuccess(adminResourceData)
						Expect(createdResourceData).To(Equal(adminResourceData))
					})

					It("Should not create resource in other domain", func() {
						createAndExpectResourceError(otherDomainResourceData)
					})
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				It("Should fill in the domain_id", func() {
					createdResourceData := createAndExpectSuccess(map[string]interface{}{"id": adminResourceID, "tenant_id": adminTenantID})
					Expect(createdResourceData).To(HaveKeyWithValue("domain_id", domainAID))
					Expect(createdResourceData).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(createdResourceData).To(HaveKeyWithValue("id", adminResourceID))
				})

				It("Should not create resource without tenant_id", func() {
					createAndExpectResourceError(map[string]interface{}{"id": adminResourceID})
				})

				Context("Resources restricted with is_public", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					DescribeTable("Should create resource in current domain",
						func(resourceData *map[string]interface{}) {
							createdResourceData := createAndExpectSuccess(*resourceData)
							Expect(createdResourceData).To(Equal(*resourceData))
						},
						Entry("Resource 1", &adminResourceData),
						Entry("Resource 2", &memberResourceData),
					)

					It("Should not create resource in other domain", func() {
						createAndExpectResourceError(otherDomainResourceData)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					DescribeTable("Should create resource in current domain",
						func(resourceData *map[string]interface{}) {
							createdResourceData := createAndExpectSuccess(*resourceData)
							Expect(createdResourceData).To(Equal(*resourceData))
						},
						Entry("Resource 1", &adminResourceData),
						Entry("Resource 2", &memberResourceData),
					)

					It("Should not create resource in other domain", func() {
						createAndExpectResourceError(otherDomainResourceData)
					})
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
				createAndExpectResourceError(adminResourceData)
			})
		})

		Describe("When tenant_id is blacklisted", func() {
			BeforeEach(func() {
				schemaID = "blacklisted_tenant_id"
			})

			Context("A member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				unpackResponseFrom := func(context middleware.Context) map[string]interface{} {
					result := context["response"].(map[string]interface{})
					response, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					return response.(map[string]interface{})
				}

				It("Should create own resource when tenant_id isn't provided and fill it in DB",
					func() {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, map[string]interface{}{
								"id":        memberResourceID,
								"domain_id": domainAID,
							})
						Expect(err).NotTo(HaveOccurred())

						delete(context, "response")
						delete(context, "resource")

						err = resources.GetSingleResource(context, testDB, currentSchema,
							memberResourceID)
						Expect(err).NotTo(HaveOccurred())

						response := unpackResponseFrom(context)
						Expect(response).To(HaveKeyWithValue("tenant_id", auth.TenantID()))
					})

				It("Should create own resource when tenant_id isn't provided "+
					"and response should not contain it",
					func() {
						err := resources.CreateResource(
							context, testDB, fakeIdentity, currentSchema, map[string]interface{}{
								"id":        memberResourceID,
								"domain_id": domainAID,
							})
						Expect(err).NotTo(HaveOccurred())

						response := unpackResponseFrom(context)
						Expect(response).To(HaveKeyWithValue("id", memberResourceID))
						Expect(response).To(HaveKeyWithValue("domain_id", domainAID))
						Expect(response).NotTo(HaveKey("tenant_id"))
					})

				It("Should not create resource when blacklisted tenant_id is provided", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{
							"id":        memberResourceID,
							"tenant_id": memberTenantID,
							"domain_id": domainAID,
						})

					Expect(err).To(HaveOccurred())

					resErr, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
					Expect(resErr.Problem).To(Equal(resources.Unauthorized))
				})

				It("Should create and update same resource when tenant_id is not provided", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{
							"id":        memberResourceID,
							"domain_id": domainAID,
						})

					Expect(err).NotTo(HaveOccurred())

					err = resources.UpdateResource(context, testDB, fakeIdentity, currentSchema,
						memberResourceID, map[string]interface{}{})

					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Describe("Updating a resource", func() {
		BeforeEach(func() {
			schemaID = "test"
			action = "create"

			setupTestResources()
		})

		Describe("When there are no resources in the database", func() {
			It("Should return an informative error", func() {
				updateAndExpectError(adminResourceID, adminResourceData)
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
						newResourceData := map[string]interface{}{"test_string": "Ia, ia, HJPEV fhtang!"}
						updatedResourceData := updateAndExpectSuccess(resourceID, newResourceData)
						Expect(updatedResourceData).To(HaveKeyWithValue("test_string", "Ia, ia, HJPEV fhtang!"))
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

				Context("Resources restricted with is_public", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					It("Should update own resource", func() {
						resourceData := map[string]interface{}{"test_string": "Steloj ne estas en ordo."}
						updatedResourceData := updateAndExpectSuccess(memberResourceID, resourceData)
						Expect(updatedResourceData).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
					})

					It("Should not update not own resource", func() {
						resourceData := map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"}
						updateAndExpectError(adminResourceID, resourceData)
					})

					It("Should not update resource from other domain", func() {
						resourceData := map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"}
						updateAndExpectError(otherDomainResourceID, resourceData)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					It("Should update own resource", func() {
						resourceData := map[string]interface{}{"test_string": "Steloj ne estas en ordo."}
						updatedResourceData := updateAndExpectSuccess(memberResourceID, resourceData)
						Expect(updatedResourceData).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
					})

					It("Should update other tenant's resource, from the same domain", func() {
						resourceData := map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"}
						updatedResourceData := updateAndExpectSuccess(adminResourceID, resourceData)
						Expect(updatedResourceData).To(HaveKeyWithValue("test_string", "Ia, ia, HWCBN fhtang!"))
					})

					It("Should not update resource from other domain", func() {
						resourceData := map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"}
						updateAndExpectError(otherDomainResourceID, resourceData)
					})
				})
			})

			Context("As a domain-scoped user", func() {
				BeforeEach(func() {
					auth = domainScopedAuth
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "test"
					})

					DescribeTable("Should update resource in current domain",
						func(resourceID string) {
							resourceData := map[string]interface{}{"test_string": "Steloj ne estas en ordo."}
							updatedResourceData := updateAndExpectSuccess(resourceID, resourceData)
							Expect(updatedResourceData).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
						},
						Entry("Resource 1", adminResourceID),
						Entry("Resource 2", memberResourceID),
					)

					It("Should not update resource from other domain", func() {
						resourceData := map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"}
						updateAndExpectError(otherDomainResourceID, resourceData)
					})
				})

				Context("Resources restricted with is_domain_owner", func() {
					BeforeEach(func() {
						schemaID = "domain_owner_test"
					})

					DescribeTable("Should update resource in current domain",
						func(resourceID string) {
							resourceData := map[string]interface{}{"test_string": "Steloj ne estas en ordo."}
							updatedResourceData := updateAndExpectSuccess(resourceID, resourceData)
							Expect(updatedResourceData).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
						},
						Entry("Resource 1", adminResourceID),
						Entry("Resource 2", memberResourceID),
					)

					It("Should not update resource from other domain", func() {
						resourceData := map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"}
						updateAndExpectError(otherDomainResourceID, resourceData)
					})
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
			fakeAction             schema.Action
			fakeActionWithoutInput schema.Action
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
			setupTestResources()
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
			fakeAction = schema.NewAction("fake_action", "GET", "/:id/whatever", "", "", inputSchema, nil, nil, false)
			fakeActionWithoutInput = schema.NewAction("fake_action", "GET", "/:id/whatever", "", "", nil, nil, nil, false)
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
