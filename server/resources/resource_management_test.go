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
	"fmt"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/server/resources"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/twinj/uuid"
)

var _ = Describe("Resource manager", func() {
	const (
		resourceID1 = "6660fbf8-ca60-4cb0-a42e-9a913beafbaf"
		resourceID2 = "6660fbf8-ca60-4cb0-a42e-9a913beafbae"
	)

	var (
		manager       *schema.Manager
		adminAuth     schema.Authorization
		memberAuth    schema.Authorization
		auth          schema.Authorization
		context       middleware.Context
		schemaID      string
		path          string
		action        string
		currentSchema *schema.Schema
		extensions    []*schema.Extension
		env           extension.Environment
		events        map[string]string
		timeLimit     time.Duration
		timeLimits    []*schema.PathEventTimeLimit
	)

	BeforeEach(func() {
		manager = schema.GetManager()

		adminAuth = schema.NewAuthorization(adminTenantID, "admin", adminTokenID, []string{"admin"}, nil)
		memberAuth = schema.NewAuthorization(memberTenantID, "demo", memberTokenID, []string{"Member"}, nil)
		auth = adminAuth
		context = middleware.Context{}
		events = map[string]string{}
		timeLimit = time.Duration(10) * time.Second
		timeLimits = []*schema.PathEventTimeLimit{}
	})

	environmentManager := extension.GetManager()

	JustBeforeEach(func() {
		var ok bool
		currentSchema, ok = manager.Schema(schemaID)
		Expect(ok).To(BeTrue())

		path = currentSchema.GetPluralURL()

		policy, role := manager.PolicyValidate(action, path, auth)
		Expect(policy).NotTo(BeNil())
		context["policy"] = policy
		context["role"] = role
		context["tenant_id"] = auth.TenantID()
		context["tenant_name"] = auth.TenantName()
		context["auth_token"] = auth.AuthToken()
		context["catalog"] = auth.Catalog()
		context["auth"] = auth

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
		Expect(db.Within(testDB, func(tx transaction.Transaction) error {

			environmentManager.UnRegisterEnvironment(schemaID)
			environmentManager.UnRegisterEnvironment("network")
			environmentManager.UnRegisterEnvironment("nil_test")
			for _, schema := range schema.GetManager().Schemas() {
				if whitelist[schema.ID] {
					continue
				}
				Expect(clearTable(tx, schema)).ToNot(HaveOccurred(), "Failed to clear table.")
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
					trimmedSchema, err := server.GetSchema(currentSchema, auth)
					Expect(err).NotTo(HaveOccurred())
					rawSchema := currentSchema.JSON()
					fullSchema, err := schema.NewResource(currentSchema, rawSchema)
					Expect(err).ToNot(HaveOccurred())
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
				trimmedSchema, err := server.GetSchema(currentSchema, auth)
				Expect(err).NotTo(HaveOccurred())
				Expect(trimmedSchema).NotTo(BeNil())
				rawSchema := schemaSchema.JSON()
				fullSchema, err := schema.NewResource(currentSchema, rawSchema)
				Expect(err).ToNot(HaveOccurred())
				Expect(trimmedSchema).To(util.MatchAsJSON(fullSchema))
			})

			It("Should return trimmed schema when appropriate", func() {
				By("Fetching the schema")
				networkSchema, ok := manager.Schema("network")
				Expect(ok).To(BeTrue())
				trimmedSchema, err := server.GetSchema(networkSchema, auth)
				Expect(err).NotTo(HaveOccurred())
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
				trimmedSchema, err := server.GetSchema(testSchema, auth)
				Expect(err).NotTo(HaveOccurred())
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
			var (
				adminResourceData, memberResourceData map[string]interface{}
			)

			BeforeEach(func() {
				adminResourceData = map[string]interface{}{
					"id":           resourceID1,
					"tenant_id":    adminTenantID,
					"test_string":  "Steloj estas en ordo.",
					"test_number":  0.5,
					"test_integer": 1,
					"test_bool":    false,
				}
				memberResourceData = map[string]interface{}{
					"id":           resourceID2,
					"tenant_id":    powerUserTenantID,
					"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
					"test_number":  0.5,
					"test_integer": 1,
					"test_bool":    false,
				}
			})

			JustBeforeEach(func() {
				adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
				Expect(err).NotTo(HaveOccurred())
				memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
				Expect(err).NotTo(HaveOccurred())
				transaction, err := testDB.Begin()
				Expect(err).NotTo(HaveOccurred())
				defer transaction.Close()
				Expect(transaction.Create(adminResource)).To(Succeed())
				Expect(transaction.Create(memberResource)).To(Succeed())
				Expect(transaction.Commit()).To(Succeed())
			})

			Context("As an admin", func() {
				It("Should return a filled list", func() {
					err := resources.GetMultipleResources(
						context, testDB, currentSchema, map[string][]string{})
					result := context["response"].(map[string]interface{})
					number := context["total"].(uint64)
					Expect(err).NotTo(HaveOccurred())
					Expect(number).To(Equal(uint64(2)))
					Expect(result).To(HaveKeyWithValue("tests", ConsistOf(adminResourceData, memberResourceData)))
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
					Expect(result).To(HaveKeyWithValue("tests", ConsistOf(adminResourceData)))
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
					context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
			var (
				adminResourceData, memberResourceData map[string]interface{}
			)

			BeforeEach(func() {
				adminResourceData = map[string]interface{}{
					"id":           resourceID1,
					"tenant_id":    adminTenantID,
					"test_string":  "Steloj estas en ordo.",
					"test_number":  0.5,
					"test_integer": 1,
					"test_bool":    false,
				}
				memberResourceData = map[string]interface{}{
					"id":           resourceID2,
					"tenant_id":    powerUserTenantID,
					"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
					"test_number":  0.5,
					"test_integer": 1,
					"test_bool":    false,
				}
			})

			JustBeforeEach(func() {
				adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
				Expect(err).NotTo(HaveOccurred())
				memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
				Expect(err).NotTo(HaveOccurred())
				transaction, err := testDB.Begin()
				Expect(err).NotTo(HaveOccurred())
				defer transaction.Close()
				Expect(transaction.Create(adminResource)).To(Succeed())
				Expect(transaction.Create(memberResource)).To(Succeed())
				Expect(transaction.Commit()).To(Succeed())
			})

			Context("As an admin", func() {
				It("Should return owned resource", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, resourceID1)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(adminResourceData)))
				})

				It("Should return not owned resource", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, resourceID2)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(memberResourceData)))
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should return owned resource", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, resourceID1)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(adminResourceData)))
				})

				It("Should not return not owned resource", func() {
					err := resources.GetSingleResource(
						context, testDB, currentSchema, resourceID2)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
					context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
						Expect(err).To(HaveOccurred())
						_, ok := err.(resources.ResourceError)
						Expect(ok).To(BeTrue())
					})
				})
			})
		})

		Describe("When there are resources in the database", func() {
			var (
				adminResourceData, memberResourceData map[string]interface{}
			)

			BeforeEach(func() {
				adminResourceData = map[string]interface{}{
					"id":           resourceID1,
					"tenant_id":    adminTenantID,
					"test_string":  "Steloj estas en ordo.",
					"test_number":  0.5,
					"test_integer": 1,
					"test_bool":    false,
				}
				memberResourceData = map[string]interface{}{
					"id":           resourceID2,
					"tenant_id":    powerUserTenantID,
					"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
					"test_number":  0.5,
					"test_integer": 1,
					"test_bool":    false,
				}
			})

			JustBeforeEach(func() {
				adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
				Expect(err).NotTo(HaveOccurred())
				memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
				Expect(err).NotTo(HaveOccurred())
				transaction, err := testDB.Begin()
				Expect(err).NotTo(HaveOccurred())
				defer transaction.Close()
				Expect(transaction.Create(adminResource)).To(Succeed())
				Expect(transaction.Create(memberResource)).To(Succeed())
				Expect(transaction.Commit()).To(Succeed())
			})

			Context("As an admin", func() {
				It("Should delete owned resource", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, resourceID1)).To(Succeed())
				})

				It("Should delete not owned resource", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, resourceID1)).To(Succeed())
				})
			})

			Context("As a member", func() {
				BeforeEach(func() {
					auth = memberAuth
				})

				It("Should delete owned resource", func() {
					Expect(resources.DeleteResource(
						context, testDB, currentSchema, resourceID1)).To(Succeed())
				})

				It("Should not delete not owned resource", func() {
					err := resources.DeleteResource(
						context, testDB, currentSchema, resourceID2)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
							context, testDB, currentSchema, resourceID1)
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
			adminResourceData, memberResourceData map[string]interface{}
			fakeIdentity                          middleware.IdentityService
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
			adminResourceData = map[string]interface{}{
				"id":           resourceID1,
				"tenant_id":    adminTenantID,
				"test_string":  "Steloj estas en ordo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}
			memberResourceData = map[string]interface{}{
				"id":           resourceID2,
				"tenant_id":    powerUserTenantID,
				"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}
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
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"tenant_id": adminTenantID})
					Expect(err).NotTo(HaveOccurred())
					result := context["response"].(map[string]interface{})
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKey("id"))
				})

				It("Should fill in the tenant_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{"id": resourceID1})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("tenant_id", adminTenantID))
					Expect(theResource).To(HaveKeyWithValue("id", resourceID1))
				})

				It("Should fill in id and tenant_id", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, map[string]interface{}{})
					Expect(err).NotTo(HaveOccurred())
					result := context["response"].(map[string]interface{})
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
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
						context, testDB, fakeIdentity, currentSchema, adminResourceData)
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(HaveKeyWithValue(schemaID, util.MatchAsJSON(adminResourceData)))
				})

				It("Should not create not own resource", func() {
					err := resources.CreateResource(
						context, testDB, fakeIdentity, currentSchema, memberResourceData)
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
			JustBeforeEach(func() {
				adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
				Expect(err).NotTo(HaveOccurred())
				memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
				Expect(err).NotTo(HaveOccurred())
				transaction, err := testDB.Begin()
				Expect(err).NotTo(HaveOccurred())
				defer transaction.Close()
				Expect(transaction.Create(adminResource)).To(Succeed())
				Expect(transaction.Create(memberResource)).To(Succeed())
				Expect(transaction.Commit()).To(Succeed())
			})

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
			adminResourceData, memberResourceData map[string]interface{}
			fakeIdentity                          middleware.IdentityService
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
			adminResourceData = map[string]interface{}{
				"id":           resourceID1,
				"tenant_id":    adminTenantID,
				"test_string":  "Steloj estas en ordo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}
			memberResourceData = map[string]interface{}{
				"id":           resourceID2,
				"tenant_id":    powerUserTenantID,
				"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}

			fakeIdentity = &middleware.FakeIdentity{}
		})

		Describe("When there are no resources in the database", func() {
			It("Should return an informative error", func() {
				err := resources.UpdateResource(
					context, testDB, fakeIdentity, currentSchema, resourceID1, adminResourceData)
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
							context, testDB, fakeIdentity, currentSchema, resourceID1, adminResourceData)
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
							context, testDB, fakeIdentity, currentSchema, resourceID1, adminResourceData)
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
							context, testDB, fakeIdentity, currentSchema, resourceID1, adminResourceData)
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
							context, testDB, fakeIdentity, currentSchema, resourceID1, adminResourceData)
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
			It("Should receive id and tenat_id but should not update them", func() {
				err := resources.CreateResource(
					context, testDB, fakeIdentity, currentSchema, adminResourceData)
				Expect(err).To(Succeed())
				delete(adminResourceData, "id")
				delete(adminResourceData, "tenant_id")

				err = resources.UpdateResource(
					context, testDB, fakeIdentity, currentSchema, resourceID1, adminResourceData)
				Expect(err).To(Succeed())

			})
		})

		Describe("When there are resources in the database", func() {
			JustBeforeEach(func() {
				adminResource, err := manager.LoadResource(currentSchema.ID, adminResourceData)
				Expect(err).NotTo(HaveOccurred())
				memberResource, err := manager.LoadResource(currentSchema.ID, memberResourceData)
				Expect(err).NotTo(HaveOccurred())
				transaction, err := testDB.Begin()
				Expect(err).NotTo(HaveOccurred())
				defer transaction.Close()
				Expect(transaction.Create(adminResource)).To(Succeed())
				Expect(transaction.Create(memberResource)).To(Succeed())
				Expect(transaction.Commit()).To(Succeed())
			})

			Context("As an admin", func() {
				var (
					adminNetworkData, adminNetworkUpdate map[string]interface{}
				)

				BeforeEach(func() {
					adminNetworkData = map[string]interface{}{
						"id":            resourceID2,
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

				It("Should update own resource", func() {
					err := resources.UpdateResource(
						context, testDB, fakeIdentity, currentSchema, resourceID1,
						map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
				})

				It("Should update not own resource", func() {
					err := resources.UpdateResource(
						context, testDB, fakeIdentity, currentSchema, resourceID1,
						map[string]interface{}{"test_string": "Ia, ia, HJPEV fhtang!"})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("test_string", "Ia, ia, HJPEV fhtang!"))
				})

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

					Expect(resources.UpdateResource(context, testDB, fakeIdentity, networkSchema, resourceID2, adminNetworkUpdate)).To(Succeed())

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

					Expect(resources.UpdateResource(context, testDB, fakeIdentity, networkSchema, resourceID2, map[string]interface{}{
						"route_targets": []interface{}{"testTarget2", "testTarget3"},
					})).To(Succeed())

					result = context["response"].(map[string]interface{})
					network, found = result["network"].(map[string]interface{})
					Expect(found).To(BeTrue())
					Expect(network["route_targets"]).To(Equal([]interface{}{"testTarget2", "testTarget3"}))
				})

				It("Should properly update nil objects", func() {
					testSchema, _ := manager.Schema("nil_test")
					Expect(resources.CreateResource(context, testDB, fakeIdentity, testSchema, map[string]interface{}{"id": resourceID2})).To(Succeed())
					result := context["response"].(map[string]interface{})
					mainObject, found := result["nil_test"].(map[string]interface{})
					Expect(found).To(BeTrue())
					subObject := mainObject["nested_obj"]
					Expect(subObject).To(BeNil())

					Expect(resources.UpdateResource(context, testDB, fakeIdentity, testSchema, resourceID2, map[string]interface{}{
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
						context, testDB, fakeIdentity, currentSchema, resourceID1,
						map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
					result := context["response"].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					theResource, ok := result[schemaID]
					Expect(ok).To(BeTrue())
					Expect(theResource).To(HaveKeyWithValue("test_string", "Steloj ne estas en ordo."))
				})

				It("Should not update not own resource", func() {
					err := resources.UpdateResource(
						context, testDB, fakeIdentity, currentSchema, resourceID2,
						map[string]interface{}{"test_string": "Ia, ia, HWCBN fhtang!"})
					resource, _ := context["response"].(map[string]interface{})
					Expect(resource).To(BeNil())
					Expect(err).To(HaveOccurred())
					_, ok := err.(resources.ResourceError)
					Expect(ok).To(BeTrue())
				})
			})

			Describe("With extensions", func() {
				Context("Only pre_update", func() {
					BeforeEach(func() {
						events["pre_update"] = `throw new CustomException("bona", 390);`
					})

					It("Should run the extension", func() {
						err := resources.UpdateResource(
							context, testDB, fakeIdentity, currentSchema, resourceID1,
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
							context, testDB, fakeIdentity, currentSchema, resourceID1,
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
							context, testDB, fakeIdentity, currentSchema, resourceID1,
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
							context, testDB, fakeIdentity, currentSchema, resourceID1,
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
			adminResourceData      map[string]interface{}
			fakeIdentity           middleware.IdentityService
			fakeAction             schema.Action
			fakeActionWithoutInput schema.Action
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "create"
			adminResourceData = map[string]interface{}{
				"id":           resourceID1,
				"tenant_id":    adminTenantID,
				"test_string":  "Steloj estas en ordo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}
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
					context, testDB, fakeIdentity, currentSchema, fakeAction, resourceID1,
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
						context, testDB, fakeIdentity, currentSchema, fakeActionWithoutInput, resourceID1,
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
			adminResourceData, memberResourceData                                 map[string]interface{}
			listContext, showContext, deleteContext, createContext, updateContext middleware.Context
			fakeIdentity                                                          middleware.IdentityService
		)

		BeforeEach(func() {
			schemaID = "test"
			action = "list"
			adminResourceData = map[string]interface{}{
				"id":           resourceID1,
				"tenant_id":    adminTenantID,
				"test_string":  "Steloj estas en ordo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}
			memberResourceData = map[string]interface{}{
				"id":           resourceID2,
				"tenant_id":    powerUserTenantID,
				"test_string":  "Mi estas la pordo, mi estas la sxlosilo.",
				"test_number":  0.5,
				"test_integer": 1,
				"test_bool":    false,
			}
			listContext = middleware.Context{}
			showContext = middleware.Context{}
			deleteContext = middleware.Context{}
			createContext = middleware.Context{}
			updateContext = middleware.Context{}
			fakeIdentity = &middleware.FakeIdentity{}
		})

		JustBeforeEach(func() {
			policy, role := manager.PolicyValidate("list", path, auth)
			Expect(policy).NotTo(BeNil())
			listContext["policy"] = policy
			listContext["role"] = role
			listContext["tenant_id"] = auth.TenantID()
			listContext["tenant_name"] = auth.TenantName()
			listContext["auth_token"] = auth.AuthToken()
			listContext["catalog"] = auth.Catalog()
			listContext["auth"] = auth
			policy, role = manager.PolicyValidate("show", path, auth)
			Expect(policy).NotTo(BeNil())
			showContext["policy"] = policy
			showContext["role"] = role
			showContext["tenant_id"] = auth.TenantID()
			showContext["tenant_name"] = auth.TenantName()
			showContext["auth_token"] = auth.AuthToken()
			showContext["catalog"] = auth.Catalog()
			showContext["auth"] = auth
			policy, role = manager.PolicyValidate("delete", path, auth)
			Expect(policy).NotTo(BeNil())
			deleteContext["policy"] = policy
			deleteContext["role"] = role
			deleteContext["tenant_id"] = auth.TenantID()
			deleteContext["tenant_name"] = auth.TenantName()
			deleteContext["auth_token"] = auth.AuthToken()
			deleteContext["catalog"] = auth.Catalog()
			deleteContext["auth"] = auth
			policy, role = manager.PolicyValidate("create", path, auth)
			Expect(policy).NotTo(BeNil())
			createContext["policy"] = policy
			createContext["role"] = role
			createContext["tenant_id"] = auth.TenantID()
			createContext["tenant_name"] = auth.TenantName()
			createContext["auth_token"] = auth.AuthToken()
			createContext["catalog"] = auth.Catalog()
			createContext["auth"] = auth
			policy, role = manager.PolicyValidate("update", path, auth)
			Expect(policy).NotTo(BeNil())
			updateContext["policy"] = policy
			updateContext["role"] = role
			updateContext["tenant_id"] = auth.TenantID()
			updateContext["tenant_name"] = auth.TenantName()
			updateContext["auth_token"] = auth.AuthToken()
			updateContext["catalog"] = auth.Catalog()
			updateContext["auth"] = auth
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
				updateContext, testDB, fakeIdentity, currentSchema, resourceID1,
				map[string]interface{}{"test_string": "Steloj ne estas en ordo."})
			result = updateContext["response"].(map[string]interface{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue(schemaID, HaveKeyWithValue("test_string", "Steloj ne estas en ordo.")))

			By("Showing the updated resource")
			err = resources.GetSingleResource(
				showContext, testDB, currentSchema, resourceID1)
			result = showContext["response"].(map[string]interface{})
			By(fmt.Sprintf("%s", result))
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue(schemaID, HaveKeyWithValue("test_string", "Steloj ne estas en ordo.")))

			By("Deleting the resource")
			By(resourceID1)
			Expect(resources.DeleteResource(
				deleteContext, testDB, currentSchema, resourceID1)).To(Succeed())

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
