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
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/schema"
	srv "github.com/cloudwan/gohan/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment", func() {
	const (
		conn         = "test.db"
		dbType       = "sqlite3"
		baseURL      = "http://localhost:19090"
		adminTokenID = "admin_token"
	)

	var (
		testDB db.DB
		server *srv.Server
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

	BeforeEach(func() {
		var err error
		testDB, err = db.ConnectDB(dbType, conn, db.DefaultMaxOpenConn, options.Default())
		Expect(err).ToNot(HaveOccurred(), "Failed to connect database.")
		err = startTestServer("../test_data/test_config.yaml")
		Expect(err).ToNot(HaveOccurred(), "Failed to start test server.")
	})

	AfterEach(func() {
		schema.ClearManager()
		os.Remove(conn)
	})

	Context("Requests", func() {
		It("invokes registered Golang handlers", func() {
			res := testURL("POST", baseURL+"/v0.1/tests/echo", adminTokenID, map[string]interface{}{"test": "success"}, http.StatusOK)
			Expect(res.(map[string]interface{})).To(HaveKeyWithValue("test", "success"))
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
