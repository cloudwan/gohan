// Copyright (C) 2016  Juniper Networks, Inc.
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

package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

//HTTPGet fetch data using HTTP.
func HTTPGet(url string, headers map[string]interface{}) (map[string]interface{}, error) {
	return HTTPRequest(url, "GET", headers, nil)
}

//HTTPPost posts data using HTTP.
func HTTPPost(url string, headers map[string]interface{}, postData map[string]interface{}) (map[string]interface{}, error) {
	return HTTPRequest(url, "POST", headers, postData)
}

//HTTPPut puts data using HTTP.
func HTTPPut(url string, headers map[string]interface{}, postData map[string]interface{}) (map[string]interface{}, error) {
	return HTTPRequest(url, "PUT", headers, postData)
}

//HTTPPatch patches data using HTTP.
func HTTPPatch(url string, headers map[string]interface{}, postData map[string]interface{}) (map[string]interface{}, error) {
	return HTTPRequest(url, "PATCH", headers, postData)
}

//HTTPDelete deletes data using HTTP.
func HTTPDelete(url string, headers map[string]interface{}) (interface{}, error) {
	return HTTPRequest(url, "DELETE", headers, nil)
}

//HTTPRequest request HTTP.
func HTTPRequest(url string, method string, headers map[string]interface{}, postData map[string]interface{}) (map[string]interface{}, error) {
	timeout := time.Duration(20 * time.Second)
	client := &http.Client{Timeout: timeout}
	var reader io.Reader
	if postData != nil {
		jsonByte, err := json.Marshal(postData)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(jsonByte)
	}
	request, err := http.NewRequest(method, url, reader)
	for key, value := range headers {
		request.Header.Add(key, fmt.Sprintf("%v", value))
	}

	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var contents interface{}
	json.Unmarshal(body, &contents)
	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"header":      resp.Header,
		"raw_body":    string(body),
		"contents":    contents,
	}, nil
}
