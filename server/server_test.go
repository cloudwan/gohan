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

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
	sync_util "github.com/cloudwan/gohan/sync/util"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var (
	server *srv.Server
)

const (
	baseURL                   = "http://localhost:19090"
	schemaURL                 = baseURL + "/gohan/v0.1/schemas"
	networkPluralURL          = baseURL + "/v2.0/networks"
	subnetPluralURL           = baseURL + "/v2.0/subnets"
	serverPluralURL           = baseURL + "/v2.0/servers"
	testPluralURL             = baseURL + "/v2.0/tests"
	parentsPluralURL          = baseURL + "/v1.0/parents"
	childrenPluralURL         = baseURL + "/v1.0/children"
	schoolsPluralURL          = baseURL + "/v1.0/schools"
	citiesPluralURL           = baseURL + "/v1.0/cities"
	profilingURL              = baseURL + "/debug/pprof/"
	filterTestPluralURL       = baseURL + "/v2.0/filter_tests"
	visibilityTestPluralURL   = baseURL + "/v2.0/visible_properties_tests"
	attacherPluralURL         = baseURL + "/v2.0/attachers"
	attacherWildcardPluralURL = baseURL + "/v2.0/wildcard_attachers"
	attachTargetPluralURL     = baseURL + "/v2.0/attach_targets"
)

var _ = Describe("Server package test", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(db.WithinTx(testDB, func(tx transaction.Transaction) error {
			for _, schema := range schema.GetManager().Schemas() {
				if whitelist[schema.ID] {
					continue
				}
				Expect(dbutil.ClearTable(ctx, tx, schema)).ToNot(HaveOccurred(), "Failed to clear table.")
			}
			return nil
		})).ToNot(HaveOccurred(), "Failed to create or commit transaction.")
	})

	Describe("HTTP request", func() {
		Context("with invalid request body", func() {
			malformedRequestBody := "malformed"

			It("should not create network", func() {
				data := testURL("POST", networkPluralURL, adminTokenID,
					malformedRequestBody, http.StatusBadRequest)
				Expect(data).To(HaveKeyWithValue("error", ContainSubstring("parse data")))
			})

			It("should not create network using PUT", func() {
				data := testURL("PUT", getNetworkSingularURL("yellow"), adminTokenID,
					malformedRequestBody, http.StatusBadRequest)
				Expect(data).To(HaveKeyWithValue("error", ContainSubstring("parse data")))
			})

			It("should not update network", func() {
				network := getNetwork("yellow", adminTenantID)
				testURL("POST", networkPluralURL, adminTokenID, network, http.StatusCreated)

				data := testURL("PUT", getNetworkSingularURL("yellow"),
					adminTokenID, malformedRequestBody, http.StatusBadRequest)
				Expect(data).To(HaveKeyWithValue("error", ContainSubstring("parse data")))
			})
		})

		Context("getting from baseURL", func() {
			It("should return 200", func() {
				testURL("GET", baseURL, adminTokenID, nil, http.StatusOK)
			})
		})

		Context("getting networks while no networks", func() {
			It("should return 200(OK) status code", func() {
				testURL("GET", networkPluralURL, adminTokenID, nil, http.StatusOK)
			})
		})

		It("should not authorize getting networks with no token", func() {
			testURL("GET", networkPluralURL, "", nil, http.StatusUnauthorized)
		})

		Context("having one network", func() {
			var result interface{}

			network := getNetwork("red", "red")

			BeforeEach(func() {
				result = testURL("POST", networkPluralURL, adminTokenID, network, http.StatusCreated)
				Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(network)))
			})

			It("Should return 2 fields", func() {
				requestURL := networkPluralURL + "?_fields=id&_fields=name"
				result := testURL("GET", requestURL, adminTokenID, nil, http.StatusOK)
				res := result.(map[string]interface{})
				networks := res["networks"].([]interface{})
				n0 := networks[0].(map[string]interface{})
				Expect(len(n0)).To(Equal(2))
			})
			It("should get networks list", func() {
				result = testURL("GET", networkPluralURL, adminTokenID, nil, http.StatusOK)
				Expect(result).To(HaveKeyWithValue("networks", ConsistOf(util.MatchAsJSON(network))))
			})

			It("should get particular network", func() {
				result = testURL("GET", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusOK)
				Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(network)))
			})
			It("should get particular network with filtered fields", func() {
				result = testURL("GET", getNetworkSingularURL("red")+"?_fields=description&_fields=name&_fields=shared", adminTokenID, nil, http.StatusOK)
				subresult := result.(map[string]interface{})
				fields := subresult["network"].(map[string]interface{})
				Expect(len(fields)).To(Equal(3))
			})
			It("should not get invalid network", func() {
				testURL("GET", baseURL+"/v2.0/network/unknownID", adminTokenID, nil, http.StatusNotFound)
			})

			It("should delete particular network", func() {
				testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
			})

			Describe("updating network using PUT", func() {
				networkUpdate := map[string]interface{}{
					"name": "NetworkRed2",
				}
				invalidNetwork := map[string]interface{}{
					"id":   10,
					"name": "NetworkRed",
				}
				networkUpdated := network
				networkUpdated["name"] = "NetworkRed2"

				It("should not update network with invalid or the same network", func() {
					testURL("PUT", getNetworkSingularURL("red"), adminTokenID, invalidNetwork, http.StatusBadRequest)
					testURL("PUT", getNetworkSingularURL("red"), adminTokenID, network, http.StatusBadRequest)
				})

				It("should update and get updated network", func() {
					result = testURL("PUT", getNetworkSingularURL("red"), adminTokenID, networkUpdate, http.StatusOK)
					Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(networkUpdated)))
					result = testURL("GET", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusOK)
					Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(networkUpdated)))
				})
			})

			Describe("updating network using PATCH", func() {
				networkUpdate := map[string]interface{}{
					"name": "NetworkRed2",
				}
				invalidNetwork := map[string]interface{}{
					"id":   10,
					"name": "NetworkRed",
				}
				networkUpdated := network
				networkUpdated["name"] = "NetworkRed2"

				It("should not update network with invalid or the same network", func() {
					testURL("PATCH", getNetworkSingularURL("red"), adminTokenID, invalidNetwork, http.StatusBadRequest)
					testURL("PATCH", getNetworkSingularURL("red"), adminTokenID, network, http.StatusBadRequest)
				})

				It("should update and get updated network", func() {
					result = testURL("PATCH", getNetworkSingularURL("red"), adminTokenID, networkUpdate, http.StatusOK)
					By(fmt.Sprintf("%s", result))
					Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(networkUpdated)))
					result = testURL("GET", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusOK)
					Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(networkUpdated)))
				})
			})
		})

		It("should create network using PUT and GET it", func() {
			network := getNetwork("red", "red")
			result := testURL("PUT", getNetworkSingularURL("red"), adminTokenID, network, http.StatusCreated)
			Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(network)))
			result = testURL("GET", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(network)))
		})

		Context("trying to create network with no tenant_id", func() {
			It("should add adminTenantID as a default.", func() {
				networkRed := getNetwork("red", "red")
				delete(networkRed, "tenant_id")

				data := testURL("POST", networkPluralURL, adminTokenID, networkRed, http.StatusCreated)
				Expect(data).To(HaveKeyWithValue("network", HaveKeyWithValue("tenant_id", adminTenantID)))
			})
		})
	})

	Describe("PaginationAndSorting", func() {
		It("should work", func() {
			By("creating 2 networks")
			networkRed := getNetwork("red", "red")
			testURL("POST", networkPluralURL, adminTokenID, networkRed, http.StatusCreated)
			networkBlue := getNetwork("blue", "red")
			testURL("POST", networkPluralURL, adminTokenID, networkBlue, http.StatusCreated)

			By("assuring 2 networks were returned")
			result := testURL("GET", networkPluralURL, adminTokenID, nil, http.StatusOK)
			res := result.(map[string]interface{})
			networks := res["networks"].([]interface{})
			Expect(networks).To(HaveLen(2))

			By("assuring returned networks are sorted")
			res = result.(map[string]interface{})
			networks = res["networks"].([]interface{})
			n0, n1 := networks[0].(map[string]interface{}), networks[1].(map[string]interface{})
			Expect(n0).To(HaveKeyWithValue("id", "networkblue"))
			Expect(n1).To(HaveKeyWithValue("id", "networkred"))

			By("assuring pagination works")
			result = testURL("GET", networkPluralURL+"?limit=1&offset=1&sort_order=desc", adminTokenID, nil, http.StatusOK)
			res = result.(map[string]interface{})
			networks = res["networks"].([]interface{})
			n0 = networks[0].(map[string]interface{})
			Expect(networks).To(HaveLen(1))
			Expect(n0).To(HaveKeyWithValue("id", "networkblue"))

			result, resp := httpRequest("GET", networkPluralURL+"?limit=1&offset=1", adminTokenID, nil)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			res = result.(map[string]interface{})
			networks = res["networks"].([]interface{})
			n0 = networks[0].(map[string]interface{})
			Expect(networks).To(HaveLen(1))
			Expect(n0).To(HaveKeyWithValue("id", "networkred"))

			testURL("GET", networkPluralURL+"?limit=-1", adminTokenID, nil, http.StatusBadRequest)
			testURL("GET", networkPluralURL+"?offset=-1", adminTokenID, nil, http.StatusBadRequest)
			testURL("GET", networkPluralURL+"?sort_key=bad_key", adminTokenID, nil, http.StatusBadRequest)
			testURL("GET", networkPluralURL+"?sort_order=bad_order", adminTokenID, nil, http.StatusBadRequest)

			Expect(resp.Header.Get("X-Total-Count")).To(Equal("2"))
			testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("blue"), adminTokenID, nil, http.StatusNoContent)
		})
	})

	Describe("TwoSameResourceRelations", func() {
		It("should work", func() {
			By("creating 2 cities")
			austinCity := getCity("austin")
			bexarCity := getCity("bexar")
			testURL("POST", citiesPluralURL, adminTokenID, austinCity, http.StatusCreated)
			testURL("POST", citiesPluralURL, adminTokenID, bexarCity, http.StatusCreated)

			By("creating 2 schools")
			austinSchool := getSchool("austin", austinCity["id"].(string))
			bexarSchool := getSchool("bexar", bexarCity["id"].(string))
			testURL("POST", schoolsPluralURL, adminTokenID, austinSchool, http.StatusCreated)
			testURL("POST", schoolsPluralURL, adminTokenID, bexarSchool, http.StatusCreated)

			By("creating 2 children")
			aliceChild := getChild("alice", austinSchool["id"].(string))
			bobChild := getChild("bob", bexarSchool["id"].(string))
			testURL("POST", childrenPluralURL, adminTokenID, aliceChild, http.StatusCreated)
			testURL("POST", childrenPluralURL, adminTokenID, bobChild, http.StatusCreated)

			By("creating 1 parent")
			charlieParent := getParent("charlie", bobChild["id"].(string), aliceChild["id"].(string))
			testURL("POST", parentsPluralURL, adminTokenID, charlieParent, http.StatusCreated)

			By("assuring 1 parent was returned without error")
			result := testURL("GET", parentsPluralURL, adminTokenID, nil, http.StatusOK)
			res := result.(map[string]interface{})
			parents := res["parents"].([]interface{})
			Expect(parents).To(HaveLen(1))

			By("assuring related resources are all available")
			parent := parents[0].(map[string]interface{})
			Expect(parent).To(HaveKeyWithValue("id", charlieParent["id"]))

			boy := parent["boy"].(map[string]interface{})
			Expect(boy).To(HaveKeyWithValue("id", bobChild["id"]))
			girl := parent["girl"].(map[string]interface{})
			Expect(girl).To(HaveKeyWithValue("id", aliceChild["id"]))

			boySchool := boy["school"].(map[string]interface{})
			Expect(boySchool).To(HaveKeyWithValue("id", bexarSchool["id"]))
			girlSchool := girl["school"].(map[string]interface{})
			Expect(girlSchool).To(HaveKeyWithValue("id", austinSchool["id"]))

			boySchoolCity := boySchool["city"].(map[string]interface{})
			Expect(boySchoolCity).To(HaveKeyWithValue("id", bexarCity["id"]))
			girlSchoolCity := girlSchool["city"].(map[string]interface{})
			Expect(girlSchoolCity).To(HaveKeyWithValue("id", austinCity["id"]))
		})
	})

	Describe("Subnets", func() {
		It("should work", func() {
			network := getNetwork("red", "red")
			testURL("POST", networkPluralURL, adminTokenID, network, http.StatusCreated)

			subnet := getSubnet("red", "red", "")

			delete(subnet, "network_id")

			var result interface{}
			testURL("POST", subnetPluralURL, adminTokenID, subnet, http.StatusBadRequest)
			result = testURL("POST", getSubnetFullPluralURL("red"), adminTokenID, subnet, http.StatusCreated)

			subnet["network_id"] = "networkred"
			Expect(result).To(HaveKeyWithValue("subnet", util.MatchAsJSON(subnet)))

			result = testURL("GET", getSubnetSingularURL("red"), adminTokenID, subnet, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("subnet", util.MatchAsJSON(subnet)))

			noCidrSubnet := getSubnet("NoCIDR", "red", "networkred")
			delete(noCidrSubnet, "cidr")
			testURL("POST", getSubnetFullPluralURL("red"), adminTokenID, noCidrSubnet, http.StatusBadRequest)

			subnetUpdate := map[string]interface{}{
				"name": "subnetRed2",
			}
			testURL("PUT", getSubnetSingularURL("red"), adminTokenID, subnetUpdate, http.StatusOK)

			testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusConflict)
			testURL("DELETE", getSubnetSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
			result = testURL("GET", networkPluralURL, adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", BeEmpty()))
			testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusNotFound)
		})
	})

	Describe("NullableProperties", func() {
		It("should work", func() {
			network := getNetwork("red", "red")
			testURL("POST", networkPluralURL, adminTokenID, network, http.StatusCreated)

			// Create subnet with null name. Ensure it's not defaulted to ""
			subnet := getSubnet("red", "red", "networkred")
			subnet["name"] = nil
			testURL("POST", subnetPluralURL, adminTokenID, subnet, http.StatusCreated)

			result := testURL("GET", getSubnetSingularURL("red"), adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKey("subnet"))
			Expect(result.(map[string]interface{})["subnet"]).To(HaveKeyWithValue("name", BeNil()))

			subnetUpdateName := map[string]interface{}{
				"name": "Red network",
			}
			testURL("PUT", getSubnetSingularURL("red"), adminTokenID, subnetUpdateName, http.StatusOK)
			result = testURL("GET", getSubnetSingularURL("red"), adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("subnet", HaveKeyWithValue("name", subnetUpdateName["name"])))

			// Test setting nullable property to null
			subnetUpdateNullName := map[string]interface{}{
				"name": nil,
			}
			testURL("PUT", getSubnetSingularURL("red"), adminTokenID, subnetUpdateNullName, http.StatusOK)
			result = testURL("GET", getSubnetSingularURL("red"), adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKey("subnet"))
			Expect(result.(map[string]interface{})["subnet"]).To(HaveKeyWithValue("name", BeNil()))

			testURL("DELETE", getSubnetSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
		})
	})

	Describe("MemberToken", func() {
		It("should work", func() {
			testURL("GET", baseURL, memberTokenID, nil, http.StatusOK)
			testURL("GET", networkPluralURL, memberTokenID, nil, http.StatusOK)
			testURL("GET", networkPluralURL, "", nil, http.StatusUnauthorized)
			testURL("GET", schemaURL, memberTokenID, nil, http.StatusOK)

			network := map[string]interface{}{
				"id":   "networkred",
				"name": "Networkred",
			}
			networkExpected := map[string]interface{}{
				"id":          "networkred",
				"name":        "Networkred",
				"description": "",
				"tenant_id":   memberTenantID,
			}

			invalidNetwork := getNetwork("red", "demo")
			invalidNetwork["tenant_id"] = "demo"

			resultExpected := map[string]interface{}{
				"network": networkExpected,
			}

			testURL("POST", networkPluralURL, memberTokenID, invalidNetwork, http.StatusUnauthorized)
			result := testURL("POST", networkPluralURL, memberTokenID, network, http.StatusCreated)
			Expect(result).To(util.MatchAsJSON(resultExpected))

			result = testURL("GET", networkPluralURL, memberTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(util.MatchAsJSON(networkExpected))))

			result = testURL("GET", getNetworkSingularURL("red"), memberTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("network", networkExpected))

			result = testURL("GET", baseURL+"/_all", memberTokenID, nil, http.StatusOK)
			Expect(result).To(HaveLen(9))
			Expect(result).To(HaveKeyWithValue("networks", []interface{}{networkExpected}))
			Expect(result).To(HaveKey("schemas"))
			Expect(result).To(HaveKey("tests"))
			Expect(result).To(HaveKey("attachers"))
			Expect(result).To(HaveKey("wildcard_attachers"))
			Expect(result).To(HaveKey("attach_targets"))

			testURL("GET", baseURL+"/v2.0/network/unknownID", memberTokenID, nil, http.StatusNotFound)

			testURL("POST", subnetPluralURL, memberTokenID, getSubnet("red", "red", "networkred"), http.StatusUnauthorized)
			testURL("GET", getSubnetSingularURL("red"), memberTokenID, nil, http.StatusNotFound)
			testURL("PUT", getSubnetSingularURL("red"), memberTokenID, getSubnet("red", "red", "networkred"), http.StatusUnauthorized)

			testURL("PUT", getNetworkSingularURL("red"), memberTokenID, invalidNetwork, http.StatusUnauthorized)
			testURL("PUT", getNetworkSingularURL("red"), memberTokenID, network, http.StatusBadRequest)

			testURL("DELETE", getSubnetSingularURL("red"), memberTokenID, nil, http.StatusNotFound)
			testURL("DELETE", getNetworkSingularURL("red"), memberTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("red"), memberTokenID, nil, http.StatusNotFound)
		})
		It("should work with property based condition", func() {
			network := map[string]interface{}{
				"id":   "networkred",
				"name": "Networkred",
			}
			testURL("POST", networkPluralURL, memberTokenID, network, http.StatusCreated)

			serverID := "serverRed"
			serverNG := map[string]interface{}{
				"id":         serverID,
				"name":       "Server Red",
				"network_id": "networkred",
				"status":     "ERROR",
			}
			serverOK := map[string]interface{}{
				"id":         serverID,
				"name":       "Server Red",
				"network_id": "networkred",
				"status":     "ACTIVE",
			}
			serverUpdate := map[string]interface{}{
				"name":   "Server Red2",
				"status": "ERROR",
			}
			testURL("GET", serverPluralURL, memberTokenID, nil, http.StatusOK)
			testURL("POST", serverPluralURL, memberTokenID, serverNG, http.StatusUnauthorized)
			testURL("POST", serverPluralURL, memberTokenID, serverOK, http.StatusCreated)
			testURL("PUT", serverPluralURL+"/"+serverID, memberTokenID, serverUpdate, http.StatusOK)
			testURL("PUT", serverPluralURL+"/"+serverID, memberTokenID, serverUpdate, http.StatusUnauthorized)
			testURL("DELETE", serverPluralURL+"/"+serverID, memberTokenID, nil, http.StatusUnauthorized)
			testURL("DELETE", serverPluralURL+"/"+serverID, adminTokenID, nil, http.StatusNoContent)
		})

		Context("Visiblity of properties", func() {
			const (
				resourceType   = "visible_properties_test"
				resourceID     = "resource_id"
				visibleTokenID = "visible_token"
				hiddenTokenID  = "hidden_token"
			)

			var url = visibilityTestPluralURL + "/" + resourceID

			Context("Read", func() {
				BeforeEach(func() {
					testResource := map[string]interface{}{
						"id": resourceID,
						"a":  "a",
						"b":  "b",
					}
					testURL("POST", visibilityTestPluralURL, adminTokenID, testResource, http.StatusCreated)
				})

				It("Should see visible fields", func() {
					res := testURL("GET", url, visibleTokenID, nil, http.StatusOK)

					actual := res.(map[string]interface{})[resourceType]
					expected := map[string]interface{}{"a": "a"}

					Expect(expected).To(Equal(actual))
				})

				It("Should not see hidden fields", func() {
					res := testURL("GET", url, hiddenTokenID, nil, http.StatusOK)

					actual := res.(map[string]interface{})[resourceType]
					expected := map[string]interface{}{"b": "b"}

					Expect(expected).To(Equal(actual))
				})
			})

			DescribeTable("Create",
				func(property string, token string, status int) {
					testResource := map[string]interface{}{
						"id":     resourceID,
						property: property,
					}
					testURL("POST", visibilityTestPluralURL, token, testResource, status)
				},
				Entry("Should create exposed with correct fields", "a", visibleTokenID, http.StatusCreated),
				Entry("Should not create exposed with incorrect fields", "b", visibleTokenID, http.StatusUnauthorized),
				Entry("Should create forbidden with correct fields", "b", hiddenTokenID, http.StatusCreated),
				Entry("Should not create forbidden with incorrect fields", "a", hiddenTokenID, http.StatusUnauthorized),
			)

			DescribeTable("Update",
				func(property string, token string, status int) {
					testResource := map[string]interface{}{
						"id": resourceID,
					}
					testURL("POST", visibilityTestPluralURL, token, testResource, http.StatusCreated)

					testResource = map[string]interface{}{
						property: property,
					}
					testURL("PUT", visibilityTestPluralURL+"/"+resourceID, token, testResource, status)
				},
				Entry("Should update exposed with correct fields", "a", visibleTokenID, http.StatusOK),
				Entry("Should not update exposed with incorrect fields", "b", visibleTokenID, http.StatusUnauthorized),
				Entry("Should update forbidden with correct fields", "b", hiddenTokenID, http.StatusOK),
				Entry("Should not update forbidden with incorrect fields", "a", hiddenTokenID, http.StatusUnauthorized),
			)
		})

		Context("Filter based policy condition", func() {
			Context("Policy for get single resource", func() {
				const (
					private = "private"
					public  = "public"
				)

				BeforeEach(func() {
					testPrivate := map[string]interface{}{
						"id":    private,
						"state": "UP",
						"level": 0,
					}
					testPublic := map[string]interface{}{
						"id":    public,
						"state": "UP",
						"level": 3,
					}

					testURL("POST", filterTestPluralURL, powerUserTokenID, testPrivate, http.StatusCreated)
					testURL("POST", filterTestPluralURL, powerUserTokenID, testPublic, http.StatusCreated)
				})

				It("should not get private resource as member", func() {
					testURL("GET", filterTestPluralURL+"/"+private, memberTokenID, nil, http.StatusNotFound)
				})

				It("should get public resource as member", func() {
					testURL("GET", filterTestPluralURL+"/"+public, memberTokenID, nil, http.StatusOK)
				})

				It("should get own private resource", func() {
					testURL("GET", filterTestPluralURL+"/"+private, powerUserTokenID, nil, http.StatusOK)
				})

				It("should get own public resource", func() {
					testURL("GET", filterTestPluralURL+"/"+public, powerUserTokenID, nil, http.StatusOK)
				})
			})

			It("should work for get", func() {
				// Tests query: `SELECT ... WHERE tenant_id.. OR (state = UP AND level IN (2,3)
				expectedToContainTest := func(expectedCount int, res interface{}) {
					Expect(res.(map[string]interface{})["filter_tests"]).To(HaveLen(expectedCount))
				}
				testUp := map[string]interface{}{
					"state": "UP",
					"level": 2,
				}
				testUpLevel3 := map[string]interface{}{
					"state": "UP",
					"level": 3,
				}
				testUpLevel0ID := "testUP"
				testUpLevel0 := map[string]interface{}{
					"id":    testUpLevel0ID,
					"state": "UP",
					"level": 0,
				}
				testDownID := "testDOWN"
				testDown := map[string]interface{}{
					"id":    testDownID,
					"state": "DOWN",
					"level": 2,
				}
				testUpdate := map[string]interface{}{
					"state": "UP",
					"level": 2,
				}

				var res interface{}
				testURL("POST", filterTestPluralURL, memberTokenID, testUp, http.StatusCreated)
				testURL("POST", filterTestPluralURL, memberTokenID, testUpLevel3, http.StatusCreated)
				testURL("POST", filterTestPluralURL, memberTokenID, testUpLevel0, http.StatusCreated)
				testURL("POST", filterTestPluralURL, memberTokenID, testDown, http.StatusCreated)
				res = testURL("GET", filterTestPluralURL, memberTokenID, nil, http.StatusOK)
				expectedToContainTest(4, res)
				res = testURL("GET", filterTestPluralURL, powerUserTokenID, nil, http.StatusOK)
				expectedToContainTest(2, res)

				testURL("PUT", filterTestPluralURL+"/"+testDownID, memberTokenID, testUpdate, http.StatusOK)
				res = testURL("GET", filterTestPluralURL, memberTokenID, nil, http.StatusOK)
				expectedToContainTest(4, res)
				res = testURL("GET", filterTestPluralURL, powerUserTokenID, nil, http.StatusOK)
				expectedToContainTest(3, res)

				testURL("PUT", filterTestPluralURL+"/"+testUpLevel0ID, memberTokenID, testUpdate, http.StatusOK)
				res = testURL("GET", filterTestPluralURL, memberTokenID, nil, http.StatusOK)
				expectedToContainTest(4, res)
				res = testURL("GET", filterTestPluralURL, powerUserTokenID, nil, http.StatusOK)
				expectedToContainTest(4, res)
			})
			It("should work for update and delete", func() {
				// Update and delete allowed only for owner and status != INVALID
				testID := "test"
				testUp := map[string]interface{}{
					"id":    testID,
					"state": "UP",
					"level": 2,
				}
				testInvalid := map[string]interface{}{
					"state": "INVALID",
				}
				testDown := map[string]interface{}{
					"state": "DOWN",
				}

				testURL("POST", filterTestPluralURL, memberTokenID, testUp, http.StatusCreated)
				testURL("GET", filterTestPluralURL+"/"+testID, powerUserTokenID, nil, http.StatusOK)
				testURL("PUT", filterTestPluralURL+"/"+testID, powerUserTokenID, testInvalid, http.StatusForbidden)
				testURL("DELETE", filterTestPluralURL+"/"+testID, powerUserTokenID, nil, http.StatusForbidden)
				testURL("PUT", filterTestPluralURL+"/"+testID, memberTokenID, testInvalid, http.StatusOK)
				testURL("GET", filterTestPluralURL+"/"+testID, memberTokenID, nil, http.StatusOK)
				testURL("PUT", filterTestPluralURL+"/"+testID, memberTokenID, testDown, http.StatusForbidden)
				testURL("DELETE", filterTestPluralURL+"/"+testID, memberTokenID, nil, http.StatusForbidden)
				testURL("GET", filterTestPluralURL+"/"+testID, powerUserTokenID, nil, http.StatusNotFound)
				testURL("DELETE", filterTestPluralURL+"/"+testID, powerUserTokenID, nil, http.StatusNotFound)
				testURL("PUT", filterTestPluralURL+"/"+testID, adminTokenID, testDown, http.StatusOK)
				testURL("DELETE", filterTestPluralURL+"/"+testID, memberTokenID, nil, http.StatusNoContent)
				testURL("DELETE", filterTestPluralURL+"/"+testID, memberTokenID, nil, http.StatusNotFound)
			})
			It("should return 401 unauthorized when deleting resource without delete policy", func() {
				resource := map[string]interface{}{
					"id": "test",
				}
				testURL("POST", visibilityTestPluralURL, adminTokenID, resource, http.StatusCreated)
				testURL("DELETE", visibilityTestPluralURL+"/test", memberTokenID, nil, http.StatusUnauthorized)
			})
		})
	})

	Describe("StringQueries", func() {
		It("should work", func() {
			testURL("POST", networkPluralURL, adminTokenID, getNetwork("red", "red"), http.StatusCreated)
			testURL("POST", networkPluralURL, adminTokenID, getNetwork("red1", "red"), http.StatusCreated)

			testURL("POST", networkPluralURL, adminTokenID, getNetwork("red2", "blue"), http.StatusCreated)
			testURL("POST", networkPluralURL, adminTokenID, getNetwork("red3", "blue"), http.StatusCreated)

			result := testURL("GET", networkPluralURL, adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(getNetwork("red", "red")),
				util.MatchAsJSON(getNetwork("red1", "red")),
				util.MatchAsJSON(getNetwork("red2", "blue")),
				util.MatchAsJSON(getNetwork("red3", "blue")))))

			result = testURL("GET", networkPluralURL+"?tenant_id=red", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(getNetwork("red", "red")),
				util.MatchAsJSON(getNetwork("red1", "red")))))

			result = testURL("GET", networkPluralURL+"?id=networkred&id=networkred1", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(getNetwork("red", "red")),
				util.MatchAsJSON(getNetwork("red1", "red")))))

			result = testURL("GET", networkPluralURL+"?id=networkred&id=networkred1&id=networkred2&tenant_id=blue", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(getNetwork("red2", "blue")))))
			testURL("DELETE", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("red1"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("red2"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getNetworkSingularURL("red3"), adminTokenID, nil, http.StatusNoContent)
		})
	})

	Describe("BoolQueries", func() {
		It("should work", func() {
			network1 := getNetwork("red1", "red")
			network1["shared"] = true
			network2 := getNetwork("red2", "red")
			network2["shared"] = true
			network3 := getNetwork("red3", "red")
			network4 := getNetwork("red4", "red")

			testURL("POST", networkPluralURL, adminTokenID, network1, http.StatusCreated)
			testURL("POST", networkPluralURL, adminTokenID, network2, http.StatusCreated)
			testURL("POST", networkPluralURL, adminTokenID, network3, http.StatusCreated)
			testURL("POST", networkPluralURL, adminTokenID, network4, http.StatusCreated)

			result := testURL("GET", networkPluralURL+"?shared=true", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(network1),
				util.MatchAsJSON(network2))))
			result = testURL("GET", networkPluralURL+"?shared=True", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(network1),
				util.MatchAsJSON(network2))))

			result = testURL("GET", networkPluralURL+"?shared=false", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(network3),
				util.MatchAsJSON(network4))))
			result = testURL("GET", networkPluralURL+"?shared=False", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("networks", ConsistOf(
				util.MatchAsJSON(network3),
				util.MatchAsJSON(network4))))
		})
	})

	Describe("FullParentPath", func() {
		It("should work", func() {
			networkRed := getNetwork("red", "red")
			networkBlue := getNetwork("blue", "red")
			subnetRed := getSubnet("red", "red", "networkred")
			subnetBlue := getSubnet("blue", "red", "networkred")
			subnetYellow := getSubnet("yellow", "red", "networkblue")

			testURL("POST", networkPluralURL, adminTokenID, networkRed, http.StatusCreated)
			testURL("POST", networkPluralURL, adminTokenID, networkBlue, http.StatusCreated)
			testURL("POST", getSubnetFullPluralURL("red"), adminTokenID, subnetRed, http.StatusCreated)
			testURL("POST", getSubnetFullPluralURL("red"), adminTokenID, subnetBlue, http.StatusCreated)
			testURL("POST", getSubnetFullPluralURL("blue"), adminTokenID, subnetYellow, http.StatusCreated)
			result := testURL("GET", getSubnetFullPluralURL("red"), adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("subnets", ConsistOf(
				util.MatchAsJSON(subnetBlue),
				util.MatchAsJSON(subnetRed))))

			subnetRed["name"] = "subnetRedUpdated"

			testURL("PUT", getSubnetFullSingularURL("red", "red"), adminTokenID, map[string]interface{}{"name": "subnetRedUpdated"}, http.StatusOK)
			result = testURL("GET", getSubnetFullPluralURL("red"), adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("subnets", ConsistOf(
				util.MatchAsJSON(subnetBlue),
				util.MatchAsJSON(subnetRed))))

			testURL("DELETE", getSubnetFullSingularURL("red", "red"), adminTokenID, nil, http.StatusNoContent)
			testURL("DELETE", getSubnetFullSingularURL("red", "blue"), adminTokenID, nil, http.StatusNoContent)
		})
	})

	Describe("ExtensionErrorReporting", func() {
		It("should work", func() {
			dummyError := map[string]interface{}{
				"error": "Dummy error.",
			}
			result := testURL("POST", testPluralURL, adminTokenID, map[string]interface{}{
				"id": "dummyid",
			}, 390)
			Expect(result).To(util.MatchAsJSON(dummyError))
		})
	})

	Describe("WrappedResourceRequests", func() {
		It("should work", func() {
			testURL("GET", getNetworkSingularURL("cyan"), adminTokenID, nil, http.StatusNotFound)

			network := getNetwork("cyan", adminTenantID)
			wrappedRequest := map[string]interface{}{"network": network}
			testURL("POST", networkPluralURL, adminTokenID, wrappedRequest, http.StatusCreated)
			defer testURL("DELETE", getNetworkSingularURL("cyan"), adminTokenID, nil, http.StatusNoContent)

			wrappedRequest = map[string]interface{}{
				"network": map[string]interface{}{
					"name": "UpdatedName",
				},
			}
			response := testURL("PUT", getNetworkSingularURL("cyan"), adminTokenID, wrappedRequest, http.StatusOK)
			Expect(response).To(HaveKeyWithValue("network", HaveKeyWithValue("name", "UpdatedName")))
		})
	})

	Describe("Error codes", func() {
		It("should return BadRequest(400) when creating a resource with reference to an invalid resource", func() {
			someNetwork := map[string]interface{}{
				"id":   "networkred",
				"name": "networkred",
			}
			testURL("POST", networkPluralURL, memberTokenID, someNetwork, http.StatusCreated)
			someServer := map[string]interface{}{
				"id":         "serverred",
				"name":       "serverred",
				"network_id": "networkmagenta",
				"status":     "ACTIVE",
			}
			testURL("POST", serverPluralURL, memberTokenID, someServer, http.StatusBadRequest)
		})

		It("should return BadRequest(400) when updating a resource with reference to an invalid resource", func() {
			someNetwork := map[string]interface{}{
				"id":   "networkred",
				"name": "networkred",
			}
			testURL("POST", networkPluralURL, memberTokenID, someNetwork, http.StatusCreated)
			someServer := map[string]interface{}{
				"id":         "serverred",
				"name":       "serverred",
				"network_id": "networkmagenta",
				"status":     "ACTIVE",
			}
			testURL("POST", serverPluralURL, memberTokenID, someServer, http.StatusBadRequest)
			updatedServer := map[string]interface{}{
				"network_id": "networkmagenta",
				"status":     "ACTIVE",
			}
			testURL("PUT", getServerSingularURL("red"), memberTokenID, updatedServer, http.StatusBadRequest)
		})
	})

	Describe("ResourceSharing", func() {
		It("should work", func() {
			memberNetwork := map[string]interface{}{
				"id":          "networkbeige",
				"name":        "Networkbeige",
				"description": "The Beige Network",
				"tenant_id":   memberTenantID,
			}
			testURL("POST", networkPluralURL, powerUserTokenID, memberNetwork, http.StatusCreated)

			powerUserNetwork := getNetwork("pink", powerUserTenantID)
			testURL("POST", networkPluralURL, powerUserTokenID, powerUserNetwork, http.StatusCreated)
			defer testURL("DELETE", getNetworkSingularURL("pink"), powerUserTokenID, nil, http.StatusNoContent)

			expectedNetworks := []interface{}{
				HaveKeyWithValue("tenant_id", memberTenantID),
				HaveKeyWithValue("tenant_id", powerUserTenantID),
			}
			memberNetworks := testURL("GET", networkPluralURL, memberTokenID, nil, http.StatusOK)
			Expect(memberNetworks).To(HaveKeyWithValue("networks", ConsistOf(expectedNetworks...)))
			powerUserNetworks := testURL("GET", networkPluralURL, powerUserTokenID, nil, http.StatusOK)
			Expect(powerUserNetworks).To(HaveKeyWithValue("networks", ConsistOf(expectedNetworks...)))

			pinkUpdate := map[string]interface{}{
				"description": "Updated Pink Network",
			}
			testURL("PUT", getNetworkSingularURL("pink"), memberTokenID, pinkUpdate, http.StatusOK)
			beigeUpdate := map[string]interface{}{
				"description": "Updated Beige Network",
			}
			testURL("PUT", getNetworkSingularURL("beige"), powerUserTokenID, beigeUpdate, http.StatusOK)

			testURL("GET", getNetworkSingularURL("pink"), memberTokenID, nil, http.StatusOK)
			testURL("DELETE", getNetworkSingularURL("pink"), memberTokenID, nil, http.StatusForbidden)
		})
	})

	Describe("PreCreate", func() {
		It("should set data in pre_create", func() {
			memberNetwork := map[string]interface{}{
				"id":          "networkbeige",
				"name":        "Networkbeige",
				"description": "The Beige Network",
				"tenant_id":   memberTenantID,
			}
			testURL("POST", networkPluralURL, powerUserTokenID, memberNetwork, http.StatusCreated)

			powerUserNetwork := getNetwork("test", powerUserTenantID)
			testURL("POST", networkPluralURL, powerUserTokenID, powerUserNetwork, http.StatusCreated)

			data := testURL("GET", getNetworkSingularURL("test"), memberTokenID, nil, http.StatusOK)
			Expect(data.(map[string]interface{})["network"]).To(HaveKeyWithValue("name", "Networktest"))
			testURL("DELETE", getNetworkSingularURL("test"), powerUserTokenID, nil, http.StatusNoContent)

			powerUserNetwork = getNetwork("test", powerUserTenantID)
			powerUserNetwork["name"] = "run-pre-create"
			testURL("POST", networkPluralURL, powerUserTokenID, powerUserNetwork, http.StatusCreated)
			data = testURL("GET", getNetworkSingularURL("test"), memberTokenID, nil, http.StatusOK)
			Expect(data.(map[string]interface{})["network"]).To(HaveKeyWithValue("name", "set-in-pre-create"))
		})
	})

	Describe("Resource Actions", func() {
		responderPluralURL := baseURL + "/v2.0/responders"
		responderParentPluralURL := baseURL + "/v2.0/responder_parents"

		BeforeEach(func() {
			responderParent := map[string]interface{}{
				"id": "p1",
			}
			testURL("POST", responderParentPluralURL, adminTokenID, responderParent, http.StatusCreated)

			responder := map[string]interface{}{
				"id":                  "r1",
				"pattern":             "Hello %s!",
				"tenant_id":           memberTenantID,
				"responder_parent_id": "p1",
			}
			testURL("POST", responderPluralURL, adminTokenID, responder, http.StatusCreated)
		})

		It("Request data not available in context on GET action", func() {
			testURL("GET", responderPluralURL+"/r1", adminTokenID, nil, http.StatusOK)
		})

		It("Request data not available in context on DELETE action", func() {
			testURL("DELETE", responderPluralURL+"/r1", adminTokenID, nil, http.StatusNoContent)
		})

		It("Request data available in context on custom action", func() {
			requestInput := map[string]interface{}{
				"hello": "world",
			}
			result := testURL("POST", responderPluralURL+"/r1/verify_request_data_in_context", memberTokenID, requestInput, http.StatusOK)
			Expect(result).To(Equal(map[string]interface{}{
				"ok": true,
			}))
		})

		It("should work", func() {
			testHelloAction := map[string]interface{}{
				"name": "Heisenberg",
			}

			result := testURL("POST", responderPluralURL+"/r1/hello", memberTokenID, testHelloAction, http.StatusOK)
			Expect(result).To(Equal(map[string]interface{}{
				"output": "Hello, Heisenberg!",
			}))

			testHiAction := map[string]interface{}{
				"name": "Heisenberg",
			}

			result = testURL("POST", responderPluralURL+"/r1/hi", adminTokenID, testHiAction, http.StatusOK)
			Expect(result).To(Equal([]interface{}{"Hi", "Heisenberg", "!"}))
		})

		It("should work with parent prefix", func() {
			testHelloAction := map[string]interface{}{
				"name": "Heisenberg",
			}

			result := testURL("POST", responderParentPluralURL+"/p1/responders/r1/hello", memberTokenID, testHelloAction, http.StatusOK)
			Expect(result).To(Equal(map[string]interface{}{
				"output": "Hello, Heisenberg!",
			}))
		})

		It("should work without input shema", func() {
			result := testURL("GET", responderPluralURL+"/r1/dobranoc", memberTokenID, nil, http.StatusOK)
			Expect(result).To(Equal("Dobranoc!"))
		})

		It("should propagare custom errors", func() {
			result := testURL("GET", responderPluralURL+"/r1/test_throw", memberTokenID, nil, 499)
			Expect(result.(map[string]interface{})).To(HaveKeyWithValue("error", "tested exception"))
		})

		It("should propagare custom errors", func() {
			result := testURL("GET", responderPluralURL+"/r1/test_throw", memberTokenID, nil, 499)
			Expect(result.(map[string]interface{})).To(HaveKeyWithValue("error", "tested exception"))
		})

		It("should be unauthorized", func() {
			testHiAction := map[string]interface{}{
				"name": "Heisenberg",
			}

			result := testURL("POST", responderPluralURL+"/r1/hi", memberTokenID, testHiAction, http.StatusUnauthorized)
			Expect(result).To(HaveKey("error"))
			result = testURL("POST", responderParentPluralURL+"/p1/responders/r1/hi", memberTokenID, testHiAction, http.StatusUnauthorized)
			Expect(result).To(HaveKey("error"))
		})

		It("should be invalid requests", func() {
			badAction1 := map[string]interface{}{}
			result := testURL("POST", responderPluralURL+"/r1/hello", memberTokenID, badAction1, http.StatusBadRequest)
			Expect(result).To(HaveKey("error"))

			badAction2 := map[string]interface{}{
				"hello": "Heisenberg",
				"hi":    "Heisenberg",
			}
			result = testURL("POST", responderPluralURL+"/r1/hello", memberTokenID, badAction2, http.StatusBadRequest)
			Expect(result).To(HaveKey("error"))

			badAction3 := map[string]interface{}{
				"hello": map[string]interface{}{
					"familyName": "Heisenberg",
				},
			}
			result = testURL("POST", responderPluralURL+"/r1/hello", memberTokenID, badAction3, http.StatusBadRequest)
			Expect(result).To(HaveKey("error"))

			unknownAction := map[string]interface{}{
				"name": "Heisenberg",
			}
			testURL("POST", responderPluralURL+"/r1/dzien_dobry", memberTokenID, unknownAction, http.StatusNotFound)
			testURL("POST", responderPluralURL+"/r1/dzien_dobry", adminTokenID, unknownAction, http.StatusNotFound)
		})

		It("should deny action for member", func() {
			testURL("GET", responderPluralURL+"/r1/denied_action", memberTokenID, nil, http.StatusUnauthorized)
		})

		It("should deny action for admin", func() {
			testURL("GET", responderPluralURL+"/r1/denied_action", adminTokenID, nil, http.StatusUnauthorized)
		})
	})

	Describe("Nobody resource paths", func() {
		nobodyResourcePathRegexes := []*regexp.Regexp{
			regexp.MustCompile("/unk.own"),
			regexp.MustCompile("/test[1-3]*"),
		}

		var nobodyResourceService middleware.NobodyResourceService

		BeforeEach(func() {
			nobodyResourceService = middleware.NewNobodyResourceService(nobodyResourcePathRegexes)
		})

		Context("validate nobody resource path", func() {
			It("should not verify", func() {
				Expect(nobodyResourceService.VerifyResourcePath("/path")).To(BeFalse())
			})

			It("should verify", func() {
				Expect(nobodyResourceService.VerifyResourcePath("/unknown")).To(BeTrue())
				Expect(nobodyResourceService.VerifyResourcePath("/test56")).To(BeTrue())
			})
		})
	})

	Describe("Profiling", func() {
		Context("Checking pprof server works", func() {
			It("should return 200", func() {
				testURL("GET", profilingURL, "", nil, http.StatusOK)
			})
		})
	})

	Describe("Resync command test", func() {
		It("Should resync syncable resources", func() {
			var err error
			config := util.GetConfig()
			manager := schema.GetManager()

			syncConn, err := sync_util.CreateFromConfig(config)
			if err != nil {
				Fail(err.Error())
			}
			syncConn.Delete("/config/v2.0/networks/resync-test-net1", false)
			syncConn.Delete("/config/v2.0/networks/resync-test-net2", false)
			syncConn.Delete("/config/v2.0/subnets/test-subnet1-id", false)

			networkSchema, _ := manager.Schema("network")
			subnetSchema, _ := manager.Schema("subnet")

			tx, err := testDB.BeginTx()
			if err != nil {
				Fail(err.Error())
			}
			net1 := schema.NewResource(networkSchema, map[string]interface{}{
				"id":                "resync-test-net1",
				"route_targets":     []string{"123", "345"},
				"name":              "test-net1-name",
				"providor_networks": map[string]interface{}{"segmentation_id": 12, "segmentation_type": "vlan"},
				"description":       "",
				"shared":            false,
				"tenant_id":         "tenant1",
			})

			net2 := schema.NewResource(networkSchema, map[string]interface{}{
				"id":                "resync-test-net2",
				"route_targets":     []string{},
				"name":              "test-net2-name",
				"providor_networks": map[string]interface{}{"segmentation_id": 12, "segmentation_type": "vlan"},
				"description":       "",
				"shared":            false,
				"tenant_id":         "tenant2",
			})

			subnet1 := schema.NewResource(subnetSchema, map[string]interface{}{
				"id":          "test-subnet1-id",
				"name":        "test-subnet1-name",
				"description": "",
				"network_id":  "resync-test-net1",
				"cidr":        "10.11.23.0/24",
				"tenant_id":   "tenant1",
			})
			Expect(tx.Create(ctx, net1)).To(Succeed())
			Expect(tx.Create(ctx, subnet1)).To(Succeed())
			Expect(tx.Create(ctx, net2)).To(Succeed())
			Expect(tx.Commit()).To(Succeed())

			if err != nil {
				Fail(err.Error())
			}

			var _ *sync.Node
			_, err = syncConn.Fetch("/config/v2.0/networks/resync-test-net1")
			Expect(err).Should(HaveOccurred())
			_, err = syncConn.Fetch("/config/v2.0/networks/resync-test-net2")
			Expect(err).Should(HaveOccurred())
			_, err = syncConn.Fetch("/config/v2.0/subnets/test-subnet1-id")
			Expect(err).Should(HaveOccurred())

			srv.Resync(testDB, syncConn)

			_, err = syncConn.Fetch("/config/v2.0/networks/resync-test-net1")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = syncConn.Fetch("/config/v2.0/networks/resync-test-net2")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = syncConn.Fetch("/config/v2.0/subnets/test-subnet1-id")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Attach policy tests", func() {
		It("should validate create/update of resource through attach policies", func() {
			getResourceURL := func(schemaID, id string) string {
				s, _ := schema.GetManager().Schema(schemaID)
				return baseURL + s.URL + "/" + id
			}

			attachTargetOfMember := map[string]interface{}{
				"id":            "target_member",
				"accessibility": "owner_only",
				"block_flag":    false,
			}
			attachTargetOfPowerUser := map[string]interface{}{
				"id":            "target_power_user",
				"accessibility": "everybody",
				"block_flag":    false,
			}
			attachTargetBlocked := map[string]interface{}{
				"id":            "target_blocked",
				"accessibility": "everybody",
				"block_flag":    true,
			}
			testURL("POST", attachTargetPluralURL, memberTokenID, attachTargetOfMember, http.StatusCreated)
			testURL("POST", attachTargetPluralURL, powerUserTokenID, attachTargetOfPowerUser, http.StatusCreated)
			testURL("POST", attachTargetPluralURL, powerUserTokenID, attachTargetBlocked, http.StatusCreated)

			// Member creates attacher and attaches to their resource
			// Should pass, because the target is owned by the member
			attacherOfMember := map[string]interface{}{
				"id": "attacher_member",
				"attach_if_accessible_id": attachTargetOfMember["id"],
			}
			testURL("POST", attacherPluralURL, memberTokenID, attacherOfMember, http.StatusCreated)

			// Member updates his resource to attach to power user's attach target
			// Should pass, because the target is accessible to everyone
			attacherOfMemberUpdate := map[string]interface{}{
				"attach_if_accessible_id": attachTargetOfPowerUser["id"],
			}
			testURL("PUT", getResourceURL("attacher", "attacher_member"), memberTokenID, attacherOfMemberUpdate, http.StatusOK)

			// Member tries to attach power user's attach target via attach_if_same_owner_id
			// Should fail, because it is not theirs resource
			attacherOfMemberUpdate = map[string]interface{}{
				"attach_if_same_owner_id": attachTargetOfPowerUser["id"],
			}
			testURL("PUT", getResourceURL("attacher", "attacher_member"), memberTokenID, attacherOfMemberUpdate, http.StatusBadRequest)

			// Power user tries to create attacher and attach it to member's attach target
			// Should fail, because member's attach target is not accessible to the power user
			attacherOfPowerUser := map[string]interface{}{
				"id": "attacher_power_user",
				"attach_if_accessible_id": attachTargetOfMember["id"],
			}
			testURL("POST", attacherPluralURL, powerUserTokenID, attacherOfPowerUser, http.StatusBadRequest)

			// Power user creates attacher and attaches it to power user's attach target
			// Should succeed, because they are the target's owner
			attacherOfPowerUser = map[string]interface{}{
				"id": "attacher_power_user",
				"attach_if_accessible_id": attachTargetOfPowerUser["id"],
			}
			testURL("POST", attacherPluralURL, powerUserTokenID, attacherOfPowerUser, http.StatusCreated)

			// Power user tries to attach existing resource to member's target, but should fail
			attacherOfPowerUserUpdate := map[string]interface{}{
				"attach_if_accessible_id": attachTargetOfMember["id"],
			}
			dataBeforeUpdate := testURL("GET", getResourceURL("attacher", "attacher_power_user"), powerUserTokenID, nil, http.StatusOK)
			testURL("PUT", getResourceURL("attacher", "attacher_power_user"), powerUserTokenID, attacherOfPowerUserUpdate, http.StatusBadRequest)
			dataAfterUpdate := testURL("GET", getResourceURL("attacher", "attacher_power_user"), powerUserTokenID, nil, http.StatusOK)
			Expect(dataAfterUpdate).To(Equal(dataBeforeUpdate))

			// Power user tries to attach to target_blocked
			// They fail, because a deny policy was defined for this case
			attacherOfPowerUserUpdate = map[string]interface{}{
				"attach_if_accessible_id": attachTargetBlocked["id"],
			}
			dataBeforeUpdate = dataAfterUpdate
			testURL("PUT", getResourceURL("attacher", "attacher_power_user"), powerUserTokenID, attacherOfPowerUserUpdate, http.StatusBadRequest)
			dataAfterUpdate = testURL("GET", getResourceURL("attacher", "attacher_power_user"), powerUserTokenID, nil, http.StatusOK)
			Expect(dataAfterUpdate).To(Equal(dataBeforeUpdate))
		})

		It("should validate create/update of resource through attach policies with wildcards", func() {
			getResourceURL := func(schemaID, id string) string {
				s, _ := schema.GetManager().Schema(schemaID)
				return baseURL + s.URL + "/" + id
			}

			attachTargetOfMember := map[string]interface{}{
				"id":            "target_member",
				"accessibility": "owner_only",
				"block_flag":    false,
			}
			attachTargetOfPowerUser := map[string]interface{}{
				"id":            "target_power_user",
				"accessibility": "owner_only",
				"block_flag":    false,
			}
			testURL("POST", attachTargetPluralURL, memberTokenID, attachTargetOfMember, http.StatusCreated)
			testURL("POST", attachTargetPluralURL, powerUserTokenID, attachTargetOfPowerUser, http.StatusCreated)

			// Member tries to create attachment to power user's resource
			// Should fail
			attacherWildcardOfMember := map[string]interface{}{
				"id":          "wildcard_attacher_member",
				"attach_a_id": attachTargetOfPowerUser["id"],
			}
			testURL("POST", attacherWildcardPluralURL, memberTokenID, attacherWildcardOfMember, http.StatusBadRequest)

			// Member tries to create attachment to their resource
			// Should succeed
			attacherWildcardOfMember = map[string]interface{}{
				"id":          "wildcard_attacher_member",
				"attach_a_id": attachTargetOfMember["id"],
			}
			testURL("POST", attacherWildcardPluralURL, memberTokenID, attacherWildcardOfMember, http.StatusCreated)

			// Member tries to update other field to create attachment to power user's resoruce
			// Should fail
			attacherWildcardOfMemberUpdate := map[string]interface{}{
				"attach_b_id": attachTargetOfPowerUser["id"],
			}
			dataBeforeUpdate := testURL("GET", getResourceURL("wildcard_attacher", "wildcard_attacher_member"), memberTokenID, nil, http.StatusOK)
			testURL("PUT", getResourceURL("wildcard_attacher", "attacher_power_user"), memberTokenID, attacherWildcardOfMemberUpdate, http.StatusBadRequest)
			dataAfterUpdate := testURL("GET", getResourceURL("wildcard_attacher", "wildcard_attacher_member"), memberTokenID, nil, http.StatusOK)
			Expect(dataAfterUpdate).To(Equal(dataBeforeUpdate))

			// Member tries to update other field to create attachment to their resource
			// Should succeed
			attacherWildcardOfMemberUpdate = map[string]interface{}{
				"attach_b_id": attachTargetOfMember["id"],
			}
			testURL("PUT", getResourceURL("wildcard_attacher", "wildcard_attacher_member"), memberTokenID, attacherWildcardOfMemberUpdate, http.StatusOK)
		})
	})

	Describe("Error messages", func() {
		It("should not return db error when creating resource with non-existing key", func() {
			responderPluralURL := baseURL + "/v2.0/responders"
			responder := map[string]interface{}{
				"id":                  "r1",
				"pattern":             "Hello %s!",
				"tenant_id":           memberTenantID,
				"responder_parent_id": "not-existing-id",
			}
			jsonData, _ := json.Marshal(responder)
			expectedMessage := fmt.Sprintf("Related resource does not exist. Please check your request: %s", string(jsonData))
			testURLErrorMessage("POST", responderPluralURL, adminTokenID, responder, http.StatusBadRequest, expectedMessage)
		})
	})
})

func BenchmarkPOSTAPI(b *testing.B) {
	err := initBenchmarkDatabase()
	if err != nil {
		b.Fatal(err)
	}

	err = startTestServer("./server_test_mysql_config.yaml")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		network := getNetwork("red"+strconv.Itoa(i), "red")
		httpRequest("POST", networkPluralURL, adminTokenID, network)
	}
}

func BenchmarkGETAPI(b *testing.B) {
	err := initBenchmarkDatabase()
	if err != nil {
		b.Fatal(err)
	}

	err = startTestServer("./server_test_mysql_config.yaml")
	if err != nil {
		b.Fatal(err)
	}

	network := getNetwork("red", "red")
	httpRequest("POST", networkPluralURL, adminTokenID, network)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		httpRequest("GET", getNetworkSingularURL("red"), adminTokenID, nil)
	}
}

func initBenchmarkDatabase() error {
	schema.ClearManager()
	manager := schema.GetManager()
	manager.LoadSchemasFromFiles("../tests/test_abstract_schema.yaml", "../tests/test_schema.yaml", "../etc/schema/gohan.json")
	return dbutil.InitDBWithSchemas("mysql", "root@tcp(localhost:3306)/gohan_test", db.DefaultTestInitDBParams())
}

func startTestServer(config string) error {
	var err error
	server, err = srv.NewServer(config)
	if err != nil {
		return err
	}

	go func() {
		err := server.Start()
		if err != nil {
			panic(err)
		}
	}()

	retry := 3
	for {
		conn, err := net.Dial("tcp", server.Address())
		if err == nil {
			conn.Close()
			break
		}
		retry--
		if retry == 0 {
			return errors.New("server not started")
		}
		time.Sleep(50 * time.Millisecond)
	}
	server.SetRunning(true)

	return nil
}

func getNetwork(color string, tenant string) map[string]interface{} {
	return map[string]interface{}{
		"id":                "network" + color,
		"name":              "Network" + color,
		"description":       "The " + color + " Network",
		"tenant_id":         tenant,
		"route_targets":     []string{"1000:10000", "2000:20000"},
		"shared":            false,
		"providor_networks": map[string]interface{}{"segmentation_id": 12, "segmentation_type": "vlan"},
		"config": map[string]interface{}{
			"default_vlan": map[string]interface{}{
				"vlan_id": float64(1),
				"name":    "default_vlan",
			},
			"empty_vlan": map[string]interface{}{},
			"vpn_vlan": map[string]interface{}{
				"name": "vpn_vlan",
			},
		},
	}
}

func getSubnet(color string, tenant string, parent string) map[string]interface{} {
	return map[string]interface{}{
		"id":          "subnet" + color,
		"name":        "Subnet" + color,
		"description": "The " + color + " Subnet",
		"tenant_id":   tenant,
		"cidr":        "10.0.0.0/24",
		"network_id":  parent,
	}
}

func getCity(name string) map[string]interface{} {
	return map[string]interface{}{
		"id":   "city" + name,
		"name": "City" + name,
	}
}

func getSchool(name, cityID string) map[string]interface{} {
	return map[string]interface{}{
		"id":      "school" + name,
		"name":    "School" + name,
		"city_id": cityID,
	}
}

func getChild(name, schoolID string) map[string]interface{} {
	return map[string]interface{}{
		"id":        name,
		"school_id": schoolID,
	}
}

func getParent(name, boyID, girlID string) map[string]interface{} {
	return map[string]interface{}{
		"id":      "parent" + name,
		"boy_id":  boyID,
		"girl_id": girlID,
	}
}

func getNetworkSingularURL(color string) string {
	s, _ := schema.GetManager().Schema("network")
	return baseURL + s.URL + "/network" + color
}

func getServerSingularURL(color string) string {
	s, _ := schema.GetManager().Schema("server")
	return baseURL + s.URL + "/server" + color
}

func getSubnetSingularURL(color string) string {
	s, _ := schema.GetManager().Schema("subnet")
	return baseURL + s.URL + "/subnet" + color
}

func getSubnetFullSingularURL(networkColor, subnetColor string) string {
	return getSubnetFullPluralURL(networkColor) + "/subnet" + subnetColor
}

func getSubnetFullPluralURL(networkColor string) string {
	s, _ := schema.GetManager().Schema("network")
	return baseURL + s.URL + "/network" + networkColor + "/subnets"
}

func testURL(method, url, token string, postData interface{}, expectedCode int) interface{} {
	data, resp := httpRequest(method, url, token, postData)
	jsonData, _ := json.MarshalIndent(data, "", "    ")
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectedCode), string(jsonData))
	return data
}

func testURLErrorMessage(method, url, token string, postData interface{}, expectedCode int, expectedMessage string) interface{} {
	data, resp := httpRequest(method, url, token, postData)
	jsonData, _ := json.MarshalIndent(data, "", "    ")
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectedCode), string(jsonData))
	Expect(data).To(HaveKeyWithValue("error", expectedMessage))
	return data
}

func httpRequest(method, url, token string, postData interface{}) (interface{}, *http.Response) {
	client := &http.Client{}
	var reader io.Reader
	if postData != nil {
		jsonByte, err := json.Marshal(postData)
		Expect(err).ToNot(HaveOccurred())
		reader = bytes.NewBuffer(jsonByte)
	}
	request, err := http.NewRequest(method, url, reader)
	Expect(err).ToNot(HaveOccurred())
	request.Header.Set("X-Auth-Token", token)
	var data interface{}
	resp, err := client.Do(request)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&data)
	return data, resp
}
