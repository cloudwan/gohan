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
)

const (
	noTransactionErrorMessage        = "No transaction"
	noContextMessage                 = "No context provided"
	unknownSchemaErrorMesssageFormat = "Unknown schema '%s'"
)

func init() {
	gohanUtilInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_http": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) == 4 {
					emptyOpts, _ := otto.ToValue(map[string]interface{}{})
					call.ArgumentList = append(call.ArgumentList, emptyOpts)
				}
				VerifyCallArguments(&call, "gohan_http", 5)
				method := call.Argument(0).String()
				url := call.Argument(1).String()
				headers := ConvertOttoToGo(call.Argument(2))
				data := ConvertOttoToGo(call.Argument(3))
				options := ConvertOttoToGo(call.Argument(4))
				log.Debug("gohan_http  [%s] %s %s %s", method, headers, url, options)
				code, headers, body, err := gohanHTTP(method, url, headers, data, options)
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
					response = append(response, schema.RawData)
				}
				value, _ := vm.ToValue(response)
				return value
			},
			"gohan_schema_url": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_schema_url", 1)
				schemaID := call.Argument(0).String()
				manager := schema.GetManager()
				schema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
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
				if len(call.ArgumentList) != 1 {
					panic("Wrong number of arguments in gohan_schema_url call.")
				}
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
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_db_create call.")
				}
				rawTemplate, _ := call.Argument(0).Export()
				templateString, ok := rawTemplate.(string)
				if !ok {
					value, _ := vm.ToValue(rawTemplate)
					return value
				}
				data := ConvertOttoToGo(call.Argument(1))
				t := template.Must(template.New("tmpl").Parse(templateString))
				b := bytes.NewBuffer(make([]byte, 0, 100))
				t.Execute(b, data)
				value, _ := vm.ToValue(b.String())
				return value
			},
			"gohan_exec": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_db_create call.")
				}
				rawCommand, _ := call.Argument(0).Export()
				command, ok := rawCommand.(string)
				if !ok {
					return otto.NullValue()
				}
				stringArgs := []string{}
				rawArgs, _ := call.Argument(1).Export()
				args, ok := rawArgs.([]interface{})
				if !ok {
					return otto.NullValue()
				}
				for _, arg := range args {
					stringArgs = append(stringArgs, arg.(string))
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
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}

	}
	RegisterInit(gohanUtilInit)
}

func gohanHTTP(method, rawURL string, headers interface{}, postData interface{}, options interface{}) (int, http.Header, string, error) {
	client := &http.Client{}
	var reader io.Reader
	if postData != nil {
		log.Debug("post data %v", postData)
		jsonByte, err := json.Marshal(postData)
		if err != nil {
			return 0, http.Header{}, "", err
		}
		log.Debug("request data: %s", string(jsonByte))
		reader = bytes.NewBuffer(jsonByte)
	}
	request, err := http.NewRequest(method, rawURL, reader)
	if headers != nil {
		headerMap := headers.(map[string]interface{})
		for key, value := range headerMap {
			request.Header.Add(key, value.(string))
		}
	}
	if err != nil {
		return 0, http.Header{}, "", err
	}

	if options != nil {
		if value, ok := options.(map[string]interface{})["opaque_url"]; ok {
			if value.(bool) == true {
				request.URL = &url.URL{
					Scheme: request.URL.Scheme,
					Host:   request.URL.Host,
					Opaque: rawURL,
				}
			}
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
