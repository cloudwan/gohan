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
	"io/ioutil"
	"net/http"
	//"github.com/k0kubun/pp"
	"net/http/httptest"
	"sync"

	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/server"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/util"
	"github.com/drone/routes"
	"github.com/go-martini/martini"
)

func init() {
	gohanscript.RegisterStmtParser("http_server", httpServer)
}

func serveResponse(w http.ResponseWriter, context map[string]interface{}) {
	response := context["response"]
	responseCode, ok := context["code"].(int)
	if !ok {
		responseCode = 200
	}
	if 200 <= responseCode && responseCode < 300 {
		w.WriteHeader(responseCode)
		routes.ServeJson(w, response)
	} else {
		message := util.MaybeMap(context["exception"])
		middleware.HTTPJSONError(w, message["message"].(string), responseCode)
	}
}

func fillInContext(context middleware.Context,
	r *http.Request, w http.ResponseWriter, p martini.Params) {
	context["path"] = r.URL.Path
	context["http_request"] = r
	context["http_response"] = w
	context["params"] = p
	context["host"] = r.Host
	context["method"] = r.Method
}

func httpServer(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
	return func(globalContext *gohanscript.Context) (interface{}, error) {
		m := martini.Classic()
		var mutex = &sync.Mutex{}
		history := []interface{}{}
		server := map[string]interface{}{
			"history": history,
		}
		m.Handlers()
		m.Use(middleware.WithContext())
		m.Use(middleware.Tracing())
		m.Use(middleware.Logging())
		m.Use(martini.Recovery())
		rawBody := util.MaybeMap(stmt.RawData["http_server"])
		paths := util.MaybeMap(rawBody["paths"])
		middlewareCode := util.MaybeString(rawBody["middleware"])
		if middlewareCode != "" {
			vm := gohanscript.NewVM()
			err := vm.LoadString(stmt.File, middlewareCode)

			if err != nil {
				return nil, err
			}
			m.Use(func(w http.ResponseWriter, r *http.Request) {
				context := globalContext.Extend(nil)
				fillInContext(context.Data(), r, w, nil)

				reqData, _ := ioutil.ReadAll(r.Body)
				buff := ioutil.NopCloser(bytes.NewBuffer(reqData))
				r.Body = buff
				var data interface{}
				if reqData != nil {
					json.Unmarshal(reqData, &data)
				}

				context.Set("request", data)
				vm.Run(context.Data())
			})
		}
		m.Use(func(w http.ResponseWriter, r *http.Request) {
			reqData, _ := ioutil.ReadAll(r.Body)
			buff := ioutil.NopCloser(bytes.NewBuffer(reqData))
			r.Body = buff
			var data interface{}
			if reqData != nil {
				json.Unmarshal(reqData, &data)
			}
			mutex.Lock()
			server["history"] = append(server["history"].([]interface{}),
				map[string]interface{}{
					"method": r.Method,
					"path":   r.URL.String(),
					"data":   data,
				})
			mutex.Unlock()
		})
		for path, body := range paths {
			methods, ok := body.(map[string]interface{})
			if !ok {
				continue
			}
			for method, rawRouteBody := range methods {
				routeBody, ok := rawRouteBody.(map[string]interface{})
				if !ok {
					continue
				}
				code := util.MaybeString(routeBody["code"])
				vm := gohanscript.NewVM()
				err := vm.LoadString(stmt.File, code)
				if err != nil {
					return nil, err
				}
				switch method {
				case "get":
					m.Get(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						context := globalContext.Extend(nil)
						fillInContext(context.Data(), r, w, p)
						vm.Run(context.Data())
						serveResponse(w, context.Data())
					})
				case "post":
					m.Post(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						context := globalContext.Extend(nil)
						fillInContext(context.Data(), r, w, p)
						requestData, _ := middleware.ReadJSON(r)
						context.Set("request", requestData)
						vm.Run(context.Data())
						serveResponse(w, context.Data())
					})
				case "put":
					m.Put(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						context := globalContext.Extend(nil)
						fillInContext(context.Data(), r, w, p)
						requestData, _ := middleware.ReadJSON(r)
						context.Set("request", requestData)
						vm.Run(context.Data())
						serveResponse(w, context.Data())
					})
				case "delete":
					m.Delete(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						context := globalContext.Extend(nil)
						fillInContext(context.Data(), r, w, p)
						vm.Run(context.Data())
						serveResponse(w, context.Data())
					})
				}
			}
		}
		testMode := stmt.Args["test"].Value(globalContext).(bool)
		if testMode {
			ts := httptest.NewServer(m)
			server["server"] = ts
			return server, nil
		}
		m.RunOnAddr(stmt.Args["address"].Value(globalContext).(string))
		return nil, nil
	}, nil
}

//GetTestServerURL returns URL of ts
func GetTestServerURL(server *httptest.Server) string {
	return server.URL
}

//StopTestServer stops test server
func StopTestServer(server *httptest.Server) {
	server.Close()
}

//GohanServer starts gohan server from configFile
func GohanServer(configFile string, test bool) (interface{}, error) {
	s, err := server.NewServer(configFile)
	if err != nil {
		return nil, err
	}
	if test {
		ts := httptest.NewServer(s.Router())
		return map[string]interface{}{
			"server": ts,
			"queue":  s.Queue(),
		}, nil
	}
	s.Start()
	return nil, nil
}
