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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Keystone client", func() {
	var (
		server     *ghttp.Server
		client     KeystoneClient
		username   string = "admin"
		password   string = "password"
		domainName string = "domain"
		tenantName string = "admin"
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Match verion from auth URL", func() {
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
		})
	})
})
