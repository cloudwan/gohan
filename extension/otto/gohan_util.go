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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/twinj/uuid"
	"github.com/xyproto/otto"
)

const (
	noContextMessage                 = "No context provided"
	unknownSchemaErrorMesssageFormat = "Unknown schema '%s'"
	defaultHTTPRequestTimeout        = 3000
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
				if len(call.ArgumentList) == 5 {
					defaultTimeout, _ := otto.ToValue(defaultHTTPRequestTimeout)
					call.ArgumentList = append(call.ArgumentList, defaultTimeout)
				}
				VerifyCallArguments(&call, "gohan_http", 6)
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
				timeout, err := GetInt64(call.Argument(5))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				log.Debug("gohan_http  [%s] %s %s %s %s", method, rawHeaders, url, opaque, timeout)

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
				defer cancel()

				var (
					code    int
					headers http.Header
					body    string
				)

				done := make(chan struct{})
				go func() {
					code, headers, body, err = GohanHTTP(ctx, method, url, rawHeaders, data, opaque)
					close(done)
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_http")
					cancel()
					interrupt()
				case <-done:
				}

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
			"gohan_raw_http": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_raw_http", 4)
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
				rawData, err := GetString(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				//TODO: pass Transport options like timeouts

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// prepare request
				req, err := http.NewRequest(method, url, strings.NewReader(rawData))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				req = req.WithContext(ctx)

				// set headers
				for header, rawValue := range rawHeaders {
					value, ok := rawValue.(string)
					if !ok {
						ThrowOttoException(&call, fmt.Sprintf(
							"Header '%s' value must be a string type", header))
					}
					req.Header.Set(header, value)
				}

				var resp *http.Response

				// run query
				done := make(chan struct{})
				go func() {
					resp, err = http.DefaultTransport.RoundTrip(req)
					close(done)
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_raw_http")
					cancel()
					interrupt()
				case <-done:
				}

				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				// process resp
				result := map[string]interface{}{}
				result["status"] = resp.Status
				result["status_code"] = resp.StatusCode
				result["headers"] = resp.Header

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				result["body"] = string(body)

				value, _ := vm.ToValue(result)
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
				var sleep time.Duration

				s, _ := call.Argument(0).Export()
				switch t := s.(type) {
				case int:
					sleep = time.Duration(t) * time.Millisecond
				case int64:
					sleep = time.Duration(t) * time.Millisecond
				}
				log.Debug("Sleep %s", sleep)
				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_sleep")
					interrupt()
				case <-time.NewTimer(sleep).C:
				}

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

				cmd := exec.Command(command, stringArgs...)
				var stdout bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stdout

				done := make(chan struct{})
				go func() {
					if err = cmd.Run(); err != nil {
						log.Debug("Run %s %s error: %s", command, stringArgs, err)
					}
					close(done)
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_exec")
					if cmd.Process != nil {
						if err := cmd.Process.Kill(); err != nil {
							log.Debug("Kill %s %s failed: %s", command, stringArgs, err)
						}
					}
					interrupt()
				case <-done:
				}

				resp := map[string]string{}
				if err != nil {
					resp["status"] = "error"
					resp["output"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["output"] = stdout.String()
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
			"gohan_get_env": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_get_env", 2)
				key, err := GetString(call.Argument(0))
				ThrowWithMessageIfHappened(&call, err, "Expected one string argument")
				defaultValue, err := GetString(call.Argument(1))
				ThrowWithMessageIfHappened(&call, err, "Expected default value")
				result := os.Getenv(key)
				if result == "" {
					result = defaultValue
				}
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

// GohanHTTP performs a HTTP request
func GohanHTTP(ctx context.Context, method, rawURL string, headers map[string]interface{},
	postData interface{}, opaque bool) (int, http.Header, string, error) {

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
		log.Debug("reqest data: %s", string(requestData))
		reader = bytes.NewBuffer(requestData)
	}

	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		return 0, http.Header{}, "", err
	}
	req = req.WithContext(ctx)

	if headers != nil {
		for key, value := range headers {
			req.Header.Add(key, value.(string))
		}
	}

	if opaque {
		req.URL = &url.URL{
			Scheme: req.URL.Scheme,
			Host:   req.URL.Host,
			Opaque: rawURL,
		}
	}
	resp, err := http.DefaultClient.Do(req)
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
