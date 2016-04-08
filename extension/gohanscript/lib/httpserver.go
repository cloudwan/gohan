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
	"net/http"

	"net/http/httptest"

	"github.com/cloudwan/gohan/extension/gohanscript"
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

func httpServer(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
	return func(globalContext *gohanscript.Context) (interface{}, error) {
		m := martini.Classic()
		rawBody := util.MaybeMap(stmt.RawData["http_server"])
		paths := util.MaybeMap(rawBody["paths"])
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
						context := map[string]interface{}{
							"params": p,
							"host":   r.Host,
						}
						vm.Run(context)
						serveResponse(w, context)
					})
				case "post":
					m.Post(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						requestData, _ := middleware.ReadJSON(r)
						context := map[string]interface{}{
							"params":  p,
							"host":    r.Host,
							"request": requestData,
						}
						vm.Run(context)
						serveResponse(w, context)
					})
				case "put":
					m.Put(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						requestData, _ := middleware.ReadJSON(r)
						context := map[string]interface{}{
							"params":  p,
							"host":    r.Host,
							"request": requestData,
						}
						vm.Run(context)
						serveResponse(w, context)
					})
				case "delete":
					m.Delete(path, func(w http.ResponseWriter, r *http.Request, p martini.Params) {
						context := map[string]interface{}{
							"params": p,
							"host":   r.Host,
						}
						vm.Run(context)
						serveResponse(w, context)
					})
				}
			}
		}
		testMode := stmt.Args["test"].Value(globalContext).(bool)
		if testMode {
			ts := httptest.NewServer(m)
			return ts, nil
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
