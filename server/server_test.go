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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	gohan_sync "github.com/cloudwan/gohan/sync"
	gohan_etcd "github.com/cloudwan/gohan/sync/etcd"
	"github.com/cloudwan/gohan/util"
)

var (
	server            *srv.Server
	baseURL           = "http://localhost:19090"
	schemaURL         = baseURL + "/gohan/v0.1/schemas"
	networkPluralURL  = baseURL + "/v2.0/networks"
	subnetPluralURL   = baseURL + "/v2.0/subnets"
	serverPluralURL   = baseURL + "/v2.0/servers"
	testPluralURL     = baseURL + "/v2.0/tests"
	parentsPluralURL  = baseURL + "/v1.0/parents"
	childrenPluralURL = baseURL + "/v1.0/children"
	schoolsPluralURL  = baseURL + "/v1.0/schools"
	citiesPluralURL   = baseURL + "/v1.0/cities"
)

var _ = Describe("Server package test", func() {

	AfterEach(func() {
		tx, err := testDB.Begin()
		Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
		defer tx.Close()
		for _, schema := range schema.GetManager().Schemas() {
			if whitelist[schema.ID] {
				continue
			}
			err = clearTable(tx, schema)
			Expect(err).ToNot(HaveOccurred(), "Failed to clear table.")
		}
		err = tx.Commit()
		Expect(err).ToNot(HaveOccurred(), "Failed to commit transaction.")
	})

	Describe("Http request", func() {
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

			It("should get networks list", func() {
				result = testURL("GET", networkPluralURL, adminTokenID, nil, http.StatusOK)
				Expect(result).To(HaveKeyWithValue("networks", ConsistOf(util.MatchAsJSON(network))))
			})

			It("should get particular network", func() {
				result = testURL("GET", getNetworkSingularURL("red"), adminTokenID, nil, http.StatusOK)
				Expect(result).To(HaveKeyWithValue("network", util.MatchAsJSON(network)))
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

			result, resp := httpRequest("GET", networkPluralURL+"?limit=1&offset=1", adminTokenID, nil, nil)
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
			Expect(result).To(HaveLen(5))
			Expect(result).To(HaveKeyWithValue("networks", []interface{}{networkExpected}))
			Expect(result).To(HaveKey("schemas"))
			Expect(result).To(HaveKey("tests"))

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

			testURL("DELETE", getNetworkSingularURL("pink"), memberTokenID, nil, http.StatusNotFound)
		})
	})

	Describe("Sync", func() {
		It("should work", func() {
			manager := schema.GetManager()
			networkSchema, _ := manager.Schema("network")
			network := getNetwork("Red", "red")
			networkResource, err := manager.LoadResource("network", network)
			Expect(err).ToNot(HaveOccurred())
			testDB1 := &srv.DbSyncWrapper{DB: testDB}
			tx, err := testDB1.Begin()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Create(networkResource)).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			Expect(server.Sync()).To(Succeed())

			sync := gohan_etcd.NewSync(nil)

			writtenConfig, err := sync.Fetch("config/" + networkResource.Path())
			Expect(err).ToNot(HaveOccurred())

			var configContentsRaw interface{}
			Expect(json.Unmarshal([]byte(writtenConfig.Value), &configContentsRaw)).To(Succeed())
			configContents, ok := configContentsRaw.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(configContents).To(HaveKeyWithValue("version", float64(1)))
			var configNetworkRaw interface{}
			Expect(json.Unmarshal([]byte(configContents["body"].(string)), &configNetworkRaw)).To(Succeed())
			configNetwork, ok := configNetworkRaw.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(configNetwork).To(util.MatchAsJSON(network))

			tx, err = testDB1.Begin()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Delete(networkSchema, networkResource.ID())).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			Expect(server.Sync()).To(Succeed())

			_, err = sync.Fetch(networkResource.Path())
			Expect(err).To(HaveOccurred(), "Failed to sync db resource deletion to sync backend")
		})
	})

	Describe("Updating the state", func() {
		const (
			statePrefix      = "/state_watch/state"
			monitoringPrefix = "/state_watch/monitoring"
		)
		var (
			networkSchema   *schema.Schema
			networkResource *schema.Resource
			wrappedTestDB   db.DB
			possibleEvent   gohan_sync.Event
		)

		BeforeEach(func() {
			manager := schema.GetManager()
			var ok bool
			networkSchema, ok = manager.Schema("network")
			Expect(ok).To(BeTrue())
			network := getNetwork("Red", "red")
			var err error
			networkResource, err = manager.LoadResource("network", network)
			Expect(err).ToNot(HaveOccurred())
			wrappedTestDB = &srv.DbSyncWrapper{DB: testDB}
			tx, err := wrappedTestDB.Begin()
			defer tx.Close()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Create(networkResource)).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
		})

		Describe("Updating state", func() {
			Context("Invoked correctly", func() {
				It("Should work", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())

					tx, err := wrappedTestDB.Begin()
					Expect(err).ToNot(HaveOccurred())
					defer tx.Close()
					afterState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(tx.Commit()).To(Succeed())
					Expect(afterState.ConfigVersion).To(Equal(int64(1)))
					Expect(afterState.StateVersion).To(Equal(int64(1)))
					Expect(afterState.State).To(Equal("Ni malvarmetas"))
					Expect(afterState.Error).To(Equal(""))
					Expect(afterState.Monitoring).To(Equal(""))
				})

				It("Should ignore backwards updates", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(0),
							"error":   "",
							"state":   "Ni varmegas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())

					tx, err := wrappedTestDB.Begin()
					Expect(err).ToNot(HaveOccurred())
					defer tx.Close()
					afterState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(tx.Commit()).To(Succeed())
					Expect(afterState.ConfigVersion).To(Equal(int64(1)))
					Expect(afterState.StateVersion).To(Equal(int64(1)))
					Expect(afterState.State).To(Equal("Ni malvarmetas"))
					Expect(afterState.Error).To(Equal(""))
					Expect(afterState.Monitoring).To(Equal(""))
				})

				It("Should ignore status updates beyond the most recent config version", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni varmegas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())

					tx, err := wrappedTestDB.Begin()
					Expect(err).ToNot(HaveOccurred())
					defer tx.Close()
					afterState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(tx.Commit()).To(Succeed())
					Expect(afterState.ConfigVersion).To(Equal(int64(1)))
					Expect(afterState.StateVersion).To(Equal(int64(1)))
					Expect(afterState.State).To(Equal("Ni malvarmetas"))
					Expect(afterState.Error).To(Equal(""))
					Expect(afterState.Monitoring).To(Equal(""))
				})
			})

			Context("Invoked incorrectly", func() {
				It("With wrong key should return nil", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + strings.Replace(networkResource.Path(), "network", "netwerk", 1),
					}
					err := srv.StateUpdate(&possibleEvent, server)
					Expect(err).ToNot(HaveOccurred())
				})

				It("With wrong resource ID should return the proper error", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path() + "malesta",
					}
					err := srv.StateUpdate(&possibleEvent, server)
					Expect(err).To(MatchError(ContainSubstring("Failed to fetch")))
				})

				It("Without version should return the proper error", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"error": "",
							"state": "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					err := srv.StateUpdate(&possibleEvent, server)
					Expect(err).To(MatchError(ContainSubstring("No version")))
				})
			})
		})

		Context("Updating the monitoring state", func() {
			Context("Invoked correctly", func() {
				It("Should work", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version":    float64(1),
							"monitoring": "Ni rigardas tio",
						},
						Key: monitoringPrefix + networkResource.Path(),
					}
					Expect(srv.MonitoringUpdate(&possibleEvent, server)).To(Succeed())

					tx, err := wrappedTestDB.Begin()
					Expect(err).ToNot(HaveOccurred())
					defer tx.Close()
					afterMonitoring, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(tx.Commit()).To(Succeed())
					Expect(afterMonitoring.ConfigVersion).To(Equal(int64(1)))
					Expect(afterMonitoring.StateVersion).To(Equal(int64(1)))
					Expect(afterMonitoring.State).To(Equal("Ni malvarmetas"))
					Expect(afterMonitoring.Error).To(Equal(""))
					Expect(afterMonitoring.Monitoring).To(Equal("Ni rigardas tio"))
				})

				It("Should ignore updates if state is not up to date", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version":    float64(1),
							"monitoring": "Ni rigardas tio",
						},
						Key: monitoringPrefix + networkResource.Path(),
					}
					Expect(srv.MonitoringUpdate(&possibleEvent, server)).To(Succeed())

					tx, err := wrappedTestDB.Begin()
					Expect(err).ToNot(HaveOccurred())
					defer tx.Close()
					afterMonitoring, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(tx.Commit()).To(Succeed())
					Expect(afterMonitoring.ConfigVersion).To(Equal(int64(1)))
					Expect(afterMonitoring.StateVersion).To(Equal(int64(0)))
					Expect(afterMonitoring.State).To(Equal(""))
					Expect(afterMonitoring.Error).To(Equal(""))
					Expect(afterMonitoring.Monitoring).To(Equal(""))
				})

				It("Should ignore updates if version is not equal to config version", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version":    float64(9999),
							"monitoring": "Ni rigardas tio",
						},
						Key: monitoringPrefix + networkResource.Path(),
					}
					Expect(srv.MonitoringUpdate(&possibleEvent, server)).To(Succeed())

					tx, err := wrappedTestDB.Begin()
					Expect(err).ToNot(HaveOccurred())
					defer tx.Close()
					afterMonitoring, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(tx.Commit()).To(Succeed())
					Expect(afterMonitoring.ConfigVersion).To(Equal(int64(1)))
					Expect(afterMonitoring.StateVersion).To(Equal(int64(1)))
					Expect(afterMonitoring.State).To(Equal("Ni malvarmetas"))
					Expect(afterMonitoring.Error).To(Equal(""))
					Expect(afterMonitoring.Monitoring).To(Equal(""))
				})
			})

			Context("Invoked incorrectly", func() {
				It("With wrong key should return nil", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version":    float64(1),
							"monitoring": "Ni rigardas tio",
						},
						Key: monitoringPrefix + strings.Replace(networkResource.Path(), "network", "netwerk", 1),
					}
					err := srv.MonitoringUpdate(&possibleEvent, server)
					Expect(err).ToNot(HaveOccurred())
				})

				It("With wrong resource ID should return the proper error", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version":    float64(1),
							"monitoring": "Ni rigardas tio",
						},
						Key: monitoringPrefix + networkResource.Path() + "malesta",
					}
					err := srv.MonitoringUpdate(&possibleEvent, server)
					Expect(err).To(MatchError(ContainSubstring("Failed to fetch")))
				})

				It("Without monitoring should return the proper error", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
						},
						Key: monitoringPrefix + networkResource.Path(),
					}
					err := srv.MonitoringUpdate(&possibleEvent, server)
					Expect(err).To(MatchError(ContainSubstring("No monitoring")))
				})

				It("Without version should return the proper error", func() {
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"version": float64(1),
							"error":   "",
							"state":   "Ni malvarmetas",
						},
						Key: statePrefix + networkResource.Path(),
					}
					Expect(srv.StateUpdate(&possibleEvent, server)).To(Succeed())
					possibleEvent = gohan_sync.Event{
						Action: "this is ignored here",
						Data: map[string]interface{}{
							"monitoring": "Ni rigardas tio",
						},
						Key: monitoringPrefix + networkResource.Path(),
					}
					err := srv.MonitoringUpdate(&possibleEvent, server)
					Expect(err).To(MatchError(ContainSubstring("No version")))
				})
			})
		})
	})

	Describe("Resource Actions", func() {
		responderPluralURL := baseURL + "/v2.0/responders"

		BeforeEach(func() {
			responder := map[string]interface{}{
				"id":        "r1",
				"pattern":   "Hello %s!",
				"tenant_id": memberTenantID,
			}
			testURL("POST", responderPluralURL, adminTokenID, responder, http.StatusCreated)
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

		It("should work without input shema", func() {
			result := testURL("GET", responderPluralURL+"/r1/dobranoc", memberTokenID, nil, http.StatusOK)
			Expect(result).To(Equal("Dobranoc!"))
		})

		It("should be unauthorized", func() {
			testHiAction := map[string]interface{}{
				"name": "Heisenberg",
			}

			result := testURL("POST", responderPluralURL+"/r1/hi", memberTokenID, testHiAction, http.StatusUnauthorized)
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

	Describe("Message dispatch", func() {
		It("should work", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key"}

			test.startConsumers(consumerKeys, forwardingConsumer)

			test.sendMessages(consumerKeys)
			test.closeMessageDispatch()

			datum := <-test.channelsFromConsumers[0]
			Expect(datum).To(Equal("/key"))

			close(done)
		})

		It("should normalize registered key", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"///key1//key2///"}

			test.startConsumers(consumerKeys, forwardingConsumer)

			test.sendMessage("/key1/key2")
			test.closeMessageDispatch()

			datum := <-test.channelsFromConsumers[0]
			Expect(datum).To(Equal("///key1//key2///"))

			close(done)
		})

		It("should dispatch by key", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key1", "/key2"}

			test.startConsumers(consumerKeys, func(key string, id int, dispatch *srv.MessageDispatch, output chan string) {
				dispatch.Wait(key)
				output <- "ok"
			})

			test.sendMessages(consumerKeys)
			test.closeMessageDispatch()

			for _, ch := range test.channelsFromConsumers {
				status := <-ch
				Expect(status).To(Equal("ok"))
			}

			close(done)
		})

		It("should dispatch to all listeners of a key", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key1", "/key1"}
			consumerIds := []int{0, 1}

			test.startConsumersWithIds(consumerKeys, consumerIds, func(key string, id int, dispatch *srv.MessageDispatch, output chan string) {
				dispatch.Wait(key)
				output <- strconv.Itoa(id)
			})

			test.sendMessage("/key1")
			test.closeMessageDispatch()

			for i, ch := range test.channelsFromConsumers {
				id := <-ch
				Expect(id).To(Equal(strconv.Itoa(i)))
			}

			close(done)
		})

		It("should close listener channels on close when no messages send", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key1"}

			test.startConsumers(consumerKeys, forwardingConsumer)

			test.closeMessageDispatch()

			messageReceived := <-test.channelsFromConsumers[0]
			Expect(messageReceived).To(Equal("/key1"))

			close(done)
		})

		It("should notify about child key changes", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key"}

			test.startConsumers(consumerKeys, forwardingConsumer)

			test.sendMessage("/key/child/key")
			test.closeMessageDispatch()

			datum := <-test.channelsFromConsumers[0]
			Expect(datum).To(Equal("/key"))

			close(done)
		})

		It("should not block when child goroutine dies", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key1", "/key2"}

			test.startConsumers(consumerKeys, func(key string, id int, dispatch *srv.MessageDispatch, output chan string) {
				if key == consumerKeys[1] {
					dispatch.Wait(key)
					output <- key
				}
				// the other consumer never starts waiting
			})

			test.sendMessages(consumerKeys)
			test.closeMessageDispatch()

			datum := <-test.channelsFromConsumers[1]
			Expect(datum).To(Equal(consumerKeys[1]))

			close(done)
		})

		It("should not wait if hashes differ", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key"}

			const firstHash = "firstHash"
			const secondHash = "secondHash"
			const expectedData = "dummy"
			getResourceCalled := 0
			context := middleware.Context{}

			test.startConsumers(consumerKeys, func(key string, id int, dispatch *srv.MessageDispatch, output chan string) {
				getHash := func(context middleware.Context) string {
					return secondHash
				}

				getResource := func(context middleware.Context) error {
					getResourceCalled++
					context["response"] = expectedData
					return nil
				}

				newHash, _ := dispatch.GetOrWait(key, firstHash, context, getResource, getHash)
				output <- newHash
			})

			datum := <-test.channelsFromConsumers[0]
			Expect(datum).To(Equal("secondHash"))
			Expect(getResourceCalled).To(Equal(1))

			close(done)

			// ensure no broadcasts sent
			test.closeMessageDispatch()
		})

		It("should wait if hash is the same", func(done Done) {
			test := NewMessageDispatchTest()

			consumerKeys := []string{"/key"}

			const firstHash = "firstHash"
			const secondHash = "secondHash"
			const expectedData = "dummy"
			getResourceCalled := 0
			getHashCalled := 0
			context := middleware.Context{}

			test.startConsumers(consumerKeys, func(key string, id int, dispatch *srv.MessageDispatch, output chan string) {
				getHash := func(context middleware.Context) string {
					getHashCalled++
					if getHashCalled == 1 {
						return firstHash
					} else {
						return secondHash
					}
				}

				getResource := func(context middleware.Context) error {
					getResourceCalled++
					if getResourceCalled == 1 {
						context["response"] = "old response, should be overwritten"
					} else {
						context["response"] = expectedData
					}

					return nil
				}

				newHash, _ := dispatch.GetOrWait(key, firstHash, context, getResource, getHash)
				output <- newHash
			})

			test.sendMessage("/key")

			hash := <-test.channelsFromConsumers[0]
			Expect(hash).To(Equal("secondHash"))
			Expect(context["response"]).To(Equal(expectedData))
			Expect(getResourceCalled).To(Equal(2))
			Expect(getHashCalled).To(Equal(2))

			close(done)

			test.closeMessageDispatch()
		})

		It("should panic when broadcasting after close", func(done Done) {
			test := NewMessageDispatchTest()

			test.closeMessageDispatch()

			broadcast := func() {
				test.sendMessage("/key")
			}

			Expect(broadcast).Should(Panic())

			close(done)
		})

		It("should panic when waiting after close", func(done Done) {
			test := NewMessageDispatchTest()

			test.closeMessageDispatch()

			wait := func() {
				test.messageDispatch.Wait("/key")
			}

			Expect(wait).Should(Panic())

			close(done)
		})
	})

	Describe("Long polling", func() {
		const (
			longPollPrefix = "/gohan/long_poll_notifications/"
		)
		var (
			wrappedTestDB       db.DB
			testSync            gohan_sync.Sync
			testNetwork         map[string]interface{}
			testNetworkResource *schema.Resource
		)

		BeforeEach(func() {
			manager := schema.GetManager()
			_, ok := manager.Schema("network")
			Expect(ok).To(BeTrue())
			testNetwork = getNetwork("red", "red")
			var err error
			testNetworkResource, err = manager.LoadResource("network", testNetwork)
			Expect(err).ToNot(HaveOccurred())

			tempDB := &srv.DbSyncWrapper{DB: testDB}
			tx, err := tempDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			Expect(tx.Create(testNetworkResource)).To(Succeed())
			Expect(tx.Commit()).To(Succeed())
			tx.Close()

			Expect(server.Sync()).To(Succeed())

			testSync = gohan_etcd.NewSync(nil)
			_, err = testSync.Fetch("config/" + testNetworkResource.Path())
			Expect(err).ToNot(HaveOccurred())

			wrappedTestDB = &srv.DbLongPollNotifierWrapper{DB: &srv.DbSyncWrapper{DB: testDB}, Sync: testSync}

			tx, err = wrappedTestDB.Begin()
			Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
			defer tx.Close()
			for _, schema := range schema.GetManager().Schemas() {
				if whitelist[schema.ID] {
					continue
				}
				err = clearTable(tx, schema)
				Expect(err).ToNot(HaveOccurred(), "Failed to clear table.")
			}
			err = tx.Commit()
			Expect(err).ToNot(HaveOccurred(), "Failed to commit transaction.")

			response := testURL("POST", networkPluralURL, adminTokenID, testNetwork, http.StatusCreated)
			Expect(response).To(HaveKeyWithValue("network", util.MatchAsJSON(testNetwork)))
		})

		AfterEach(func() {
			_, err := testSync.Fetch(longPollPrefix + testNetworkResource.Path())
			if err == nil {
				err = testSync.Delete(longPollPrefix + testNetworkResource.Path())
				Expect(err).ToNot(HaveOccurred())
				_, err = testSync.Fetch(longPollPrefix + testNetworkResource.Path())
			}
			Expect(err).To(HaveOccurred())
		})

		AssertRespondsImmediately := func(getRequestURL, getCurrentResourceVersion func() string) func() {
			return func() {
				responseChan := make(chan *http.Response, 1)
				doneChan := make(chan bool, 1)
				go asyncLongPoll(getRequestURL(), getCurrentResourceVersion(), doneChan, responseChan)

				Eventually(doneChan, time.Second*10).Should(Receive())
			}
		}

		AssertVersionsDifferent := func(getRequestURL, getCurrentResourceVersion func() string) func() {
			return func() {
				originalResourceVersion := getCurrentResourceVersion()

				responseChan := make(chan *http.Response, 1)
				doneChan := make(chan bool, 1)
				go asyncLongPoll(getRequestURL(), originalResourceVersion, doneChan, responseChan)
				<-doneChan
				response := <-responseChan
				newResourceVersion := getEtag(response)
				Expect(originalResourceVersion).ToNot(Equal(newResourceVersion))
			}
		}

		AssertHangsIfVersionsSameUntilNotified := func(getRequestURL, getCurrentResourceVersion func() string) func() {
			return func() {
				responseChan := make(chan *http.Response, 1)
				doneChan := make(chan bool, 1)
				go asyncLongPoll(getRequestURL(), getCurrentResourceVersion(), doneChan, responseChan)
				Consistently(doneChan).ShouldNot(Receive())
				server.LongPoll().Broadcast(testNetworkResource.Path())
				Eventually(doneChan, time.Second*10).Should(Receive())
			}
		}

		AssertHangsAndRespondsWithNewVersionsIfResourceModified := func(getRequestURL, getCurrentResourceVersion func() string) func() {
			return func() {
				originalResourceVersion := getCurrentResourceVersion()

				responseChan := make(chan *http.Response, 1)
				doneChan := make(chan bool, 1)
				go asyncLongPoll(getRequestURL(), originalResourceVersion, doneChan, responseChan)
				Consistently(doneChan).ShouldNot(Receive())

				networkUpdate := map[string]interface{}{
					"name": "NetworkRed2",
				}
				testURL("PUT", getNetworkSingularURL("red"), adminTokenID, networkUpdate, http.StatusOK)

				server.LongPoll().Broadcast(testNetworkResource.Path())
				Eventually(doneChan, time.Second*10).Should(Receive())

				response := <-responseChan
				newResourceVersion := getEtag(response)
				Expect(originalResourceVersion).ToNot(Equal(newResourceVersion))
			}
		}

		Context("with requests of single resource", func() {
			getRequestURL := func() string {
				return getNetworkSingularURL("red")
			}

			Context("without or with empty long polling header", func() {
				getOriginalResourceEtag := func() string {
					return ""
				}

				It("should respond immediately", AssertRespondsImmediately(getRequestURL, getOriginalResourceEtag))

				It("should respond new version", AssertVersionsDifferent(getRequestURL, getOriginalResourceEtag))
			})

			Context("versions are different", func() {
				getOriginalResourceEtag := func() string {
					return "some-resource-version"
				}

				It("should respond immediately", AssertRespondsImmediately(getRequestURL, getOriginalResourceEtag))

				It("should respond new version", AssertVersionsDifferent(getRequestURL, getOriginalResourceEtag))
			})

			Context("versions are same", func() {
				getOriginalResourceEtag := func() string {
					_, response := testLongPollURL("GET", getRequestURL(), adminTokenID, "", http.StatusOK)
					return getEtag(response)
				}

				It("should hang if versions are same until notified", AssertHangsIfVersionsSameUntilNotified(getRequestURL, getOriginalResourceEtag))

				It("should hang and respond with new version if resource was modified", AssertHangsAndRespondsWithNewVersionsIfResourceModified(getRequestURL, getOriginalResourceEtag))
			})
		})

		Context("with requests of resource list", func() {
			getRequestURL := func() string {
				return networkPluralURL
			}

			Context("list versions are different", func() {
				getOriginalResourceEtag := func() string {
					return "some-resource-list-version"
				}

				It("should respond immediately", AssertRespondsImmediately(getRequestURL, getOriginalResourceEtag))

				It("should respond new version", AssertVersionsDifferent(getRequestURL, getOriginalResourceEtag))
			})

			Context("versions are same", func() {
				getOriginalResourceEtag := func() string {
					_, response := testLongPollURL("GET", getRequestURL(), adminTokenID, "", http.StatusOK)
					return getEtag(response)
				}

				It("should hang if versions are same until notified by a change to a child resource", AssertHangsIfVersionsSameUntilNotified(getRequestURL, getOriginalResourceEtag))

				It("should hang and respond with new version if child resource was modified", AssertHangsAndRespondsWithNewVersionsIfResourceModified(getRequestURL, getOriginalResourceEtag))
			})
		})

		Context("notifying a path", func() {

			It("should notify all subscibers of resource and subcribers of directories above it (parents+)", func() {
				pathSubscribers := make(map[string]chan bool)

				pathSubscribers[getNetworkSingularURL("red")] = make(chan bool, 1)
				pathSubscribers[baseURL+"/v2.0/networks"] = make(chan bool, 1)
				pathSubscribers[baseURL+"/v2.0/subnets"] = make(chan bool, 1)

				responseChan := make(chan *http.Response, len(pathSubscribers))
				for subURL, subChan := range pathSubscribers {
					_, response := testLongPollURL("GET", subURL, adminTokenID, "", http.StatusOK)
					pathEtag := getEtag(response)

					go func(url string, ch chan bool) {
						asyncLongPoll(url, pathEtag, ch, responseChan)
					}(subURL, subChan)
				}

				time.Sleep(time.Millisecond * 100) // wait for longpolling to start

				var wg sync.WaitGroup

				asyncEventually := func(URL string) {
					defer GinkgoRecover()
					defer wg.Done()
					Eventually(pathSubscribers[URL], time.Second*10).Should(Receive())
				}
				asyncConsistently := func(URL string) {
					defer GinkgoRecover()
					defer wg.Done()
					Consistently(pathSubscribers[URL]).ShouldNot(Receive())
				}

				server.LongPoll().Broadcast(testNetworkResource.Path())

				wg.Add(3)
				go asyncEventually(getNetworkSingularURL("red"))
				go asyncEventually(baseURL + "/v2.0/networks")
				go asyncConsistently(baseURL + "/v2.0/subnets")
				wg.Wait()

				time.Sleep(time.Millisecond * 200)
				server.LongPoll().Broadcast("/v2.0/subnets")

				wg.Add(3)
				go asyncConsistently(getNetworkSingularURL("red"))
				go asyncConsistently(baseURL + "/v2.0/networks")
				go asyncEventually(baseURL + "/v2.0/subnets")
				wg.Wait()
			})
		})

		Context("requesting not existing resource", func() {

			It("should not work", func() {
				_, response := testLongPollURL("GET", getNetworkSingularURL("nonexisting_resource"), adminTokenID, "", http.StatusNotFound)
				Expect(getEtag(response)).To(Equal(""))
			})
		})
	})
})

type MessageDispatchTest struct {
	messageDispatch       *srv.MessageDispatch
	channelsFromConsumers []chan string
}

func NewMessageDispatchTest() MessageDispatchTest {
	test := MessageDispatchTest{srv.NewMessageDispatch(), nil}
	return test
}

type ConsumerFunction func(key string, id int, md *srv.MessageDispatch, output chan string)

func (test *MessageDispatchTest) startConsumers(consumerKeys []string, consumer ConsumerFunction) {
	test.startConsumersWithIds(consumerKeys, make([]int, len(consumerKeys)), consumer)
}

func (test *MessageDispatchTest) startConsumersWithIds(consumerKeys []string, customerIds []int, consumer ConsumerFunction) {
	test.channelsFromConsumers = make([]chan string, len(consumerKeys))
	for i, _ := range test.channelsFromConsumers {
		test.channelsFromConsumers[i] = make(chan string)
	}

	consumerGoroutine := func(output chan string, key string, id int, consumer ConsumerFunction) {
		consumer(key, id, test.messageDispatch, output)
		close(output)
	}

	for i, key := range consumerKeys {
		go consumerGoroutine(test.channelsFromConsumers[i], key, customerIds[i], consumer)
	}

	time.Sleep(time.Millisecond * 100) // is there any better way to wait until all consumers start waiting on a cond?
}

func (test *MessageDispatchTest) sendMessages(messages []string) {
	for _, msg := range messages {
		test.messageDispatch.Broadcast(msg)
	}
}

func (test *MessageDispatchTest) sendMessage(message string) {
	test.messageDispatch.Broadcast(message)
}

func (test *MessageDispatchTest) closeMessageDispatch() {
	test.messageDispatch.Close()
}

func getEtag(r *http.Response) string {
	return r.Header.Get(srv.LongPollEtag)
}

func asyncLongPoll(requestURL, currentResourceVersion string, done chan bool, response chan *http.Response) {
	_, r := testLongPollURL("GET", requestURL, adminTokenID, currentResourceVersion, http.StatusOK)
	response <- r
	done <- true
}

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
		httpRequest("POST", networkPluralURL, adminTokenID, nil, network)
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
	httpRequest("POST", networkPluralURL, adminTokenID, nil, network)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		httpRequest("GET", getNetworkSingularURL("red"), adminTokenID, nil, nil)
	}
}

func initBenchmarkDatabase() error {
	schema.ClearManager()
	manager := schema.GetManager()
	manager.LoadSchemasFromFiles("../tests/test_abstract_schema.yaml", "../tests/test_schema.yaml", "../etc/schema/gohan.json")
	err := db.InitDBWithSchemas("mysql", "root@tcp(localhost:3306)/gohan_test", false, false)
	if err != nil {
		return err
	}
	return nil
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
			return
		}
	}()
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
	data, resp := httpRequest(method, url, token, nil, postData)
	jsonData, _ := json.MarshalIndent(data, "", "    ")
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectedCode), string(jsonData))
	return data
}

func testLongPollURL(method, url, token, longPoll string, expectedCode int) (interface{}, *http.Response) {
	headers := make(map[string]string)
	headers[srv.LongPollHeader] = longPoll

	data, resp := httpRequest(method, url, token, headers, nil)
	jsonData, _ := json.MarshalIndent(data, "", "    ")
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectedCode), string(jsonData))
	return data, resp
}

func httpRequest(method, url, token string, headers map[string]string, postData interface{}) (interface{}, *http.Response) {
	client := &http.Client{}
	var reader io.Reader
	if postData != nil {
		jsonByte, err := json.Marshal(postData)
		Expect(err).ToNot(HaveOccurred())
		reader = bytes.NewBuffer(jsonByte)
	}
	request, err := http.NewRequest(method, url, reader)
	Expect(err).ToNot(HaveOccurred())
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	request.Header.Set("X-Auth-Token", token)
	var data interface{}
	resp, err := client.Do(request)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&data)
	return data, resp
}

func clearTable(tx transaction.Transaction, s *schema.Schema) error {
	if s.IsAbstract() {
		return nil
	}
	for _, schema := range schema.GetManager().Schemas() {
		if schema.ParentSchema == s {
			err := clearTable(tx, schema)
			if err != nil {
				return err
			}
		} else {
			for _, property := range schema.Properties {
				if property.Relation == s.Singular {
					err := clearTable(tx, schema)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	resources, _, err := tx.List(s, nil, nil)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		err = tx.Delete(s, resource.ID())
		if err != nil {
			return err
		}
	}
	return nil
}

func forwardingConsumer(key string, id int, dispatch *srv.MessageDispatch, output chan string) {
	dispatch.Wait(key)
	output <- key
}
