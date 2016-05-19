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

package otto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"text/template"
	"time"

	"github.com/dop251/otto"
	"github.com/twinj/uuid"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

const (
	noContextMessage                 = "No context provided"
	unknownSchemaErrorMesssageFormat = "Unknown schema '%s'"
)

func init() {
	gohanUtilInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_http": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) == 4 {
					defaultOpaque, _ := otto.ToValue(false)
					call.ArgumentList = append(call.ArgumentList, defaultOpaque)
				}
				VerifyCallArguments(&call, "gohan_http", 5)
				method, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				url, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				rawHeaders, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				// A string or a map[string]interface{}
				data := ConvertOttoToGo(call.Argument(3))
				opaque, err := GetBool(call.Argument(4))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				log.Debug("gohan_http  [%s] %s %s %s", method, rawHeaders, url, opaque)
				code, headers, body, err := gohanHTTP(method, url, rawHeaders, data, opaque)
				log.Debug("response code %d", code)
				resp := map[string]interface{}{}
				if err != nil {
					resp["status"] = "err"
					resp["error"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["status_code"] = fmt.Sprint(code)
					resp["body"] = body
					resp["headers"] = headers
				}
				log.Debug("response code %d", code)
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_schemas": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_schemas", 0)
				manager := schema.GetManager()
				response := []interface{}{}
				for _, schema := range manager.OrderedSchemas() {
					response = append(response, schema)
				}
				value, _ := vm.ToValue(response)
				return value
			},
			"gohan_schema_url": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_schema_url", 1)
				schemaID, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				schema, err := getSchema(schemaID)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				value, _ := vm.ToValue(schema.URL)
				return value
			},
			"gohan_policies": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_policies", 0)
				manager := schema.GetManager()
				response := []interface{}{}
				for _, policy := range manager.Policies() {
					response = append(response, policy.RawData)
				}
				value, _ := vm.ToValue(response)
				return value
			},
			"gohan_uuid": func(call otto.FunctionCall) otto.Value {
				value, _ := vm.ToValue(uuid.NewV4().String())
				return value
			},
			"gohan_sleep": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_sleep", 1)
				rawSleep, _ := call.Argument(0).Export()
				var sleep time.Duration
				switch rawSleep.(type) {
				case int:
					sleep = time.Duration(rawSleep.(int))
				case int64:
					sleep = time.Duration(rawSleep.(int64))
				}
				time.Sleep(sleep * time.Millisecond)
				return otto.NullValue()
			},
			"gohan_template": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_template", 2)
				templateString, err := GetString(call.Argument(0))
				if err != nil {
					return call.Argument(0)
				}
				data := ConvertOttoToGo(call.Argument(1))
				t := template.Must(template.New("tmpl").Parse(templateString))
				b := bytes.NewBuffer(make([]byte, 0, 100))
				t.Execute(b, data)
				value, _ := vm.ToValue(b.String())
				return value
			},
			"gohan_exec": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_exec", 2)
				command, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				stringArgs, err := GetStringList(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				out, err := exec.Command(command, stringArgs...).Output()
				resp := map[string]string{}
				if err != nil {
					resp["status"] = "error"
					resp["output"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["output"] = string(out)
				}
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_config": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_exec", 2)
				configKey, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				defaultValue, err := call.Argument(1).Export()
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				config := util.GetConfig()
				result := config.GetParam(configKey, defaultValue)
				value, _ := vm.ToValue(result)
				return value
			},
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}

	}
	RegisterInit(gohanUtilInit)
}

func gohanHTTP(method, rawURL string, headers map[string]interface{},
	postData interface{}, opaque bool) (int, http.Header, string, error) {

	client := &http.Client{}
	var reader io.Reader

	if postData != nil {
		log.Debug("post data %v", postData)
		var requestData []byte
		var err error

		contentType := ""
		if c, ok := headers["content-type"]; ok {
			contentType = c.(string)
		}

		if contentType == "text/plain" {
			if d, ok := postData.(string); ok {
				requestData = []byte(d)
			}
		} else {
			requestData, err = json.Marshal(postData)
			if err != nil {
				return 0, http.Header{}, "", err
			}
			// set application/json as content-type because of json.Marshal
			if headers != nil {
				headers["content-type"] = "application/json"
			}
		}
		log.Debug("request data: %s", string(requestData))
		reader = bytes.NewBuffer(requestData)

	}

	request, err := http.NewRequest(method, rawURL, reader)
	if headers != nil {
		for key, value := range headers {
			request.Header.Add(key, value.(string))
		}
	}
	if err != nil {
		return 0, http.Header{}, "", err
	}

	if opaque {
		request.URL = &url.URL{
			Scheme: request.URL.Scheme,
			Host:   request.URL.Host,
			Opaque: rawURL,
		}
	}
	resp, err := client.Do(request)
	if err != nil {
		return 0, http.Header{}, "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, resp.Header, "", err
	}
	return resp.StatusCode, resp.Header, string(body), nil
}
