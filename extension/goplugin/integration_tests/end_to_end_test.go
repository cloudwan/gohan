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

package goplugin_integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment", func() {
	const (
		conn         = "../test_data/test.db"
		dbType       = "sqlite3"
		baseURL      = "http://localhost:19090"
		adminTokenID = "admin_token"
	)

	var (
		testDB    db.DB
		server    *srv.Server
		whitelist = map[string]bool{
			"schema":    true,
			"policy":    true,
			"extension": true,
			"namespace": true,
		}
		ctx context.Context
	)

	startTestServer := func(config string) error {
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

	BeforeSuite(func() {
		removeFileDb(conn)
		var err error
		testDB, err = dbutil.ConnectDB(dbType, conn, db.DefaultMaxOpenConn, options.Default())
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		err = startTestServer("../test_data/test_config.yaml")
		Expect(err).ToNot(HaveOccurred(), "Failed to start test server.")
	})

	AfterSuite(func() {
		schema.ClearManager()
		removeFileDb(conn)
	})

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

	Context("Requests", func() {
		It("invokes registered Golang handlers", func() {
			res := testURL("POST", baseURL+"/v0.1/tests/echo", adminTokenID, map[string]interface{}{"test": "success"}, http.StatusOK)
			Expect(res.(map[string]interface{})).To(HaveKeyWithValue("test", "success"))
		})

		It("triggers JS handlers from Golang plugins", func() {
			res := testURL("POST", baseURL+"/v0.1/tests/invoke_js", adminTokenID, map[string]interface{}{}, http.StatusOK)
			Expect(res.(map[string]interface{})).To(HaveKeyWithValue("js_called", true))
			Expect(res.(map[string]interface{})).To(HaveKeyWithValue("id_valid", true))
		})
	})

	Context("Sync", func() {
		It("handles context.Cancel", func() {
			res := testURL("POST", baseURL+"/v0.1/tests/sync_context_cancel", adminTokenID, map[string]interface{}{"test": "success"}, http.StatusOK)
			Expect(res.(map[string]interface{})).To(HaveKeyWithValue("test", "success"))
		})
	})

	Context("Resource creation", func() {
		It("Creates resources", func() {
			resource := map[string]interface{}{
				"id":            "testId",
				"description":   "test description",
				"test_suite_id": nil,
				"subobject":     nil,
				"name":          nil,
			}

			expectedResponse := map[string]interface{}{
				"id":            "testId",
				"description":   "test description",
				"test_suite_id": nil,
				"name":          "abc",
				"enumerations":  []interface{}{},
			}

			result := testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusBadRequest)
			Expect(result).To(HaveKeyWithValue("error", "Validation error: Json validation error:\n\tname: Invalid type. Expected: string, given: null,"))
			resource["name"] = "a"
			result = testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusBadRequest)
			Expect(result).To(HaveKeyWithValue("error", "Validation error: Json validation error:\n\tname: String length must be greater than or equal to 3,"))
			resource["name"] = "abc"
			result = testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusCreated)
			Expect(result).To(HaveKeyWithValue("test", expectedResponse))
		})

		It("Fails to create when string field given as int", func() {
			resource := map[string]interface{}{
				"test": map[string]interface{}{
					"id":          "testId",
					"description": 1,
				},
			}

			result := testURL("POST", baseURL+"/v0.1/tests", adminTokenID, resource, http.StatusBadRequest)
			Expect(result).To(HaveKeyWithValue("error", ContainSubstring("invalid type")))
		})
	})

	Context("Resource update", func() {
		var expectedResponse = map[string]interface{}{
			"id":            "testId",
			"description":   "test description",
			"test_suite_id": nil,
			"name":          "abc",
			"subobject":     nil,
			"enumerations":  []interface{}{},
		}

		BeforeEach(func() {
			resource := map[string]interface{}{
				"id":            "testId",
				"description":   "test description",
				"test_suite_id": nil,
				"subobject":     nil,
				"name":          "abc",
				"enumerations":  []interface{}{},
			}

			testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusCreated)
			result := testURL("GET", baseURL+"/v0.1/tests/testId", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("test", expectedResponse))
		})

		It("Update resources", func() {
			resource := map[string]interface{}{
				"name": nil,
			}

			result := testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusBadRequest)
			Expect(result).To(HaveKeyWithValue("error", "Validation error: Json validation error:\n\tname: Invalid type. Expected: string, given: null,"))
			resource["name"] = "a"
			result = testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusBadRequest)
			Expect(result).To(HaveKeyWithValue("error", "Validation error: Json validation error:\n\tname: String length must be greater than or equal to 3,"))
			resource["name"] = "abcd"
			expectedResponse["name"] = "abcd"
			result = testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("test", expectedResponse))
		})
	})

	Context("Fetching resources", func() {
		It("fetches existing resources", func() {
			resource := map[string]interface{}{
				"id":            "testId",
				"description":   "test description",
				"test_suite_id": nil,
				"subobject":     nil,
				"enumerations":  []test.EnumerationSubobject{},
				"name":          "abc",
			}

			testURL("PUT", baseURL+"/v0.1/tests/testId", adminTokenID, resource, http.StatusCreated)

			result := testURL("GET", baseURL+"/v0.1/tests/testId", adminTokenID, nil, http.StatusOK)
			Expect(result).To(HaveKeyWithValue("test", util.MatchAsJSON(resource)))
		})
	})
})

func testURL(method, url, token string, postData interface{}, expectedCode int) interface{} {
	data, resp := httpRequest(method, url, token, postData)
	jsonData, _ := json.MarshalIndent(data, "", "    ")
	ExpectWithOffset(1, resp.StatusCode).To(Equal(expectedCode), string(jsonData))
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

func removeFileDb(fileName string) {
	os.Remove(fileName)
}
