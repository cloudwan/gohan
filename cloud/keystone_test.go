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
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Keystone client", func() {
	var (
		server     *ghttp.Server
		client     KeystoneClient
		username   = "admin"
		password   = "password"
		domainName = "domain"
		tenantName = "admin"
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Match version from auth URL", func() {
		It("Should match v2 version successfully", func() {
			res := matchVersionFromAuthURL("http://example.com:5000/v2.0")
			Expect(res).To(Equal("v2.0"))
			res = matchVersionFromAuthURL("http://example.com:5000/v2.0/")
			Expect(res).To(Equal("v2.0"))
		})

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
		Context("Keystone v2", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV2TokensResponse()),
				)
				client, _ = NewKeystoneV2Client(server.URL()+"/v2.0", username, password, tenantName)
			})

			It("Should map Tenant Name to Tenant ID successfully", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV2TenantsResponse()),
					ghttp.RespondWithJSONEncoded(200, getV2TenantsResponse()),
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
					ghttp.RespondWithJSONEncoded(200, getV2TenantsResponse()),
					ghttp.RespondWithJSONEncoded(200, getV2TenantsResponse()),
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
					ghttp.RespondWithJSONEncoded(200, getV2TenantsResponse()),
				)
				tenantID, err := client.GetTenantID("santa")
				Expect(tenantID).To(Equal(""))
				Expect(err).To(MatchError("Tenant with name 'santa' not found"))
			})

			It("Should show error - tenant with provided name not found", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, getV2TenantsResponse()),
				)
				tenantName, err := client.GetTenantName("santa")
				Expect(tenantName).To(Equal(""))
				Expect(err).To(MatchError("Tenant with ID 'santa' not found"))
			})
		})

		Context("Keystone v3", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(201, getV3TokensResponse()),
				)
				client, _ = NewKeystoneV3Client(server.URL()+"/v3", username, password, domainName, tenantName)
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
							ghttp.RespondWithJSONEncoded(201, getV3TokensResponse(), http.Header{"X-Subject-Token": {newServiceToken}}),
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
							ghttp.RespondWithJSONEncoded(200, getV3TokensResponse()),
						),
					)
					auth, err := client.VerifyToken(validUserToken)
					Expect(err).To(BeNil())
					Expect(auth.TenantID()).To(Equal("acme-id"))
					Expect(auth.TenantName()).To(Equal("acme"))
					Expect(auth.Roles()).To(Equal([]*schema.Role{{"member"}}))
				})
			})
		})
	})
})
