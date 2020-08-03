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

package cloud

import (
	"net/http"

	"github.com/cloudwan/gohan/schema"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Keystone client", func() {
	var (
		server     *ghttp.Server
		client     KeystoneIdentity
		username   = "admin"
		password   = "password"
		domainName = "domain"
		tenantName = "admin"
	)

	setupV3Client := func() {
		var err error
		client, err = NewKeystoneIdentity(server.URL()+"/v3", username, password, domainName, tenantName)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Keystone validation", func() {

		expectTokenAuthScopedToDomain := func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncoded(201, getV3TokensScopedToDomainResponse()),
			)
		}

		expectTokenAuthScopedToTenant := func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncoded(201, getV3TokensScopedToTenantResponse()),
			)
		}

		expectTenantList := func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
			)
		}

		expectDomainList := func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncoded(200, getV3DomainsResponse()),
			)
		}

		Context("TenantID and DomainID pair validation", func() {
			It("should reject valid tenantID within different domain", func() {
				expectTokenAuthScopedToDomain()
				setupV3Client()

				expectDomainList()
				expectTenantList()

				valid, err := client.ValidateTenantIDAndDomainIDPair("3456", "default")
				Expect(valid).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should reject invalid tenantID and valid DomainID", func() {
				expectTokenAuthScopedToDomain()
				setupV3Client()

				expectDomainList()
				expectTenantList()

				valid, err := client.ValidateTenantIDAndDomainIDPair("some-random-id", "default")
				Expect(valid).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should succeed when valid tenantID within correct domain is provided", func() {
				expectTokenAuthScopedToDomain()
				setupV3Client()

				expectDomainList()
				expectTenantList()

				valid, err := client.ValidateTenantIDAndDomainIDPair("1234", "default")
				Expect(valid).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error when keystone is not operational", func() {
				valid, err := client.ValidateTenantIDAndDomainIDPair("1234", "default")
				Expect(err).To(HaveOccurred())
				Expect(valid).To(BeFalse())
			})
		})

		Context("Tenant ID validation", func() {
			It("should reject tenant id is not known", func() {
				expectTokenAuthScopedToTenant()
				setupV3Client()

				expectTenantList()

				valid, err := client.ValidateTenantID("some-random-id")
				Expect(valid).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should succeed when tenant id is known to backend", func() {
				expectTokenAuthScopedToTenant()
				setupV3Client()

				expectTenantList()

				valid, err := client.ValidateTenantID("1234")
				Expect(err).ToNot(HaveOccurred())
				Expect(valid).To(BeTrue())
			})

			It("should return an error when keystone is not operational", func() {
				valid, err := client.ValidateTenantID("1234")
				Expect(err).To(HaveOccurred())
				Expect(valid).To(BeFalse())
			})

		})

		Context("Domain ID validation", func() {

			It("should reject when domain id is not known to backend", func() {
				expectTokenAuthScopedToDomain()
				setupV3Client()

				expectDomainList()

				valid, err := client.ValidateDomainID("some-random-id")
				Expect(valid).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should succeed when domain id is known to backend", func() {
				expectTokenAuthScopedToDomain()
				setupV3Client()

				expectDomainList()

				valid, err := client.ValidateDomainID("default")
				Expect(err).ToNot(HaveOccurred())
				Expect(valid).To(BeTrue())
			})

			It("should return an error when keystone is not operational", func() {
				valid, err := client.ValidateDomainID("default")
				Expect(err).To(HaveOccurred())
				Expect(valid).To(BeFalse())
			})
		})
	})

	Describe("Match version from auth URL", func() {
		It("Should match v3 version successfully", func() {
			res := matchVersionFromAuthURL("http://example.com:5000/v3")
			Expect(res).To(Equal("v3"))
			res = matchVersionFromAuthURL("http://example.com:5000/v3/")
			Expect(res).To(Equal("v3"))
		})

		It("Should should match no version", func() {
			res := matchVersionFromAuthURL("http://example.com:5000/nonsense")
			Expect(res).To(Equal(""))
		})
	})

	Describe("Tenant ID <-> Tenant Name Mapper", func() {
		Context("Keystone v3", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(201, getV3TokensScopedToTenantResponse()),
				)
				setupV3Client()
			})

			It("Should map Tenant Name to Tenant ID successfully", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
					ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
				)
				tenantID, err := client.GetTenantID("admin")
				Expect(err).ToNot(HaveOccurred())
				Expect(tenantID).To(Equal("1234"))

				tenantID, err = client.GetTenantID("demo")
				Expect(err).ToNot(HaveOccurred())
				Expect(tenantID).To(Equal("3456"))
			})

			It("Should map Tenant ID to Tenant Name successfully", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
					ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
				)
				tenantName, err := client.GetTenantName("1234")
				Expect(err).ToNot(HaveOccurred())
				Expect(tenantName).To(Equal("admin"))

				tenantName, err = client.GetTenantName("3456")
				Expect(err).ToNot(HaveOccurred())
				Expect(tenantName).To(Equal("demo"))
			})

			It("Should show error - tenant with provided id not found", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
				)
				tenantID, err := client.GetTenantID("santa")
				Expect(tenantID).To(Equal(""))
				Expect(err).To(MatchError("Tenant with name 'santa' not found"))
			})

			It("Should show error - tenant with provided name not found", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV3TenantsResponse()),
				)
				tenantName, err := client.GetTenantName("santa")
				Expect(tenantName).To(Equal(""))
				Expect(err).To(MatchError("Tenant with ID 'santa' not found"))
			})

			Context("With expired service token and successful re-authentication", func() {
				var (
					serviceTokenRequest map[string]interface{}
					newServiceToken     = "new-service-token"
					invalidUserToken    = "invalid-user-token"
					validUserToken      = "valid-user-token"
				)

				BeforeEach(func() {
					serviceTokenRequest = map[string]interface{}{
						"auth": map[string]interface{}{
							"identity": map[string]interface{}{
								"methods": []interface{}{
									"password",
								},
								"password": map[string]interface{}{
									"user": map[string]interface{}{
										"password": password,
										"name":     username,
										"domain": map[string]interface{}{
											"name": domainName,
										},
									},
								},
							},
							"scope": map[string]interface{}{
								"project": map[string]interface{}{
									"domain": map[string]interface{}{
										"name": domainName,
									},
									"name": tenantName,
								},
							},
						},
					}

					server.AppendHandlers(
						ghttp.RespondWithJSONEncoded(401, getV3Unauthorized()),
						ghttp.CombineHandlers(
							ghttp.VerifyJSONRepresenting(serviceTokenRequest),
							ghttp.RespondWithJSONEncoded(201, getV3TokensScopedToTenantResponse(), http.Header{"X-Subject-Token": {newServiceToken}}),
						),
					)
				})

				It("reject invalid user token", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyHeader(http.Header{
								"X-Auth-Token":    {newServiceToken},
								"X-Subject-Token": {invalidUserToken},
							}),
							ghttp.RespondWith(404, ""),
						),
					)

					_, err := client.VerifyToken(invalidUserToken)
					Expect(err).To(MatchError("Invalid token"))
				})

				It("accept valid user token", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyHeader(http.Header{
								"X-Auth-Token":    {newServiceToken},
								"X-Subject-Token": {validUserToken},
							}),
							ghttp.RespondWithJSONEncoded(200, getV3TokensScopedToTenantResponse()),
						),
					)
					auth, err := client.VerifyToken(validUserToken)
					Expect(err).To(BeNil())
					Expect(auth.TenantID()).To(Equal("acme-id"))
					Expect(auth.TenantName()).To(Equal("acme"))
					Expect(auth.DomainID()).To(Equal("domain-id"))
					Expect(auth.DomainName()).To(Equal("domain"))
					Expect(auth.Roles()).To(Equal([]*schema.Role{{Name: "Member"}}))
				})
			})
		})
	})

	Context("Token scope", func() {
		var token = "token"

		setResponses := func(response interface{}) {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncoded(201, response),
				ghttp.RespondWithJSONEncoded(200, response),
			)
		}

		Describe("Keystone v3", func() {
			It("Should read tokens with tenant scope", func() {
				setResponses(getV3TokensScopedToTenantResponse())
				setupV3Client()

				auth, err := client.VerifyToken(token)

				Expect(err).To(BeNil())
				Expect(auth.TenantID()).To(Equal("acme-id"))
				Expect(auth.TenantName()).To(Equal("acme"))
				Expect(auth.DomainID()).To(Equal("domain-id"))
				Expect(auth.DomainName()).To(Equal("domain"))
				Expect(auth.IsAdmin()).To(BeFalse())
			})

			It("Should read tokens with domain scope", func() {
				setResponses(getV3TokensScopedToDomainResponse())
				setupV3Client()

				auth, err := client.VerifyToken(token)

				Expect(err).To(BeNil())
				Expect(auth.TenantID()).To(Equal(""))
				Expect(auth.TenantName()).To(Equal(""))
				Expect(auth.DomainID()).To(Equal("domain-id"))
				Expect(auth.DomainName()).To(Equal("domain"))
				Expect(auth.IsAdmin()).To(BeFalse())
			})

			It("Should read tokens scoped to admin project", func() {
				setResponses(getV3TokensAdminResponse())
				setupV3Client()

				auth, err := client.VerifyToken(token)

				Expect(err).To(BeNil())
				Expect(auth.TenantID()).To(Equal("admin-project-id"))
				Expect(auth.TenantName()).To(Equal("admin-project"))
				Expect(auth.DomainID()).To(Equal("default"))
				Expect(auth.DomainName()).To(Equal("default"))
				Expect(auth.IsAdmin()).To(BeTrue())
			})

			DescribeTable("Admin token rejection cases",
				func(roles ...interface{}) {
					setResponses(getV3TokensAdminResponseWithRoles(roles))
					setupV3Client()

					_, err := client.VerifyToken(token)

					Expect(err).To(HaveOccurred())
				},
				Entry("Reject multiple roles", getMemberRole(), getAdminRole()),
				Entry("Reject non-admin role", getMemberRole()),
				Entry("Reject no roles"),
			)
		})
	})
})
