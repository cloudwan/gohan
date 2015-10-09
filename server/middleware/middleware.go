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

package middleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwan/gohan/schema"
	"github.com/go-martini/martini"
)

type responseHijacker struct {
	martini.ResponseWriter
	Response *bytes.Buffer
}

func newResponseHijacker(rw martini.ResponseWriter) *responseHijacker {
	return &responseHijacker{rw, bytes.NewBuffer(nil)}
}

func (rh *responseHijacker) Write(b []byte) (int, error) {
	rh.Response.Write(b)
	return rh.ResponseWriter.Write(b)
}

//Logging logs requests and responses
func Logging() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, c martini.Context) {
		start := time.Now()

		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = req.RemoteAddr
			}
		}

		reqData, _ := ioutil.ReadAll(req.Body)
		buff := ioutil.NopCloser(bytes.NewBuffer(reqData))
		req.Body = buff

		log.Info("Started %s %s for client %s data: %s",
			req.Method, req.URL.String(), addr, string(reqData))
		log.Debug("Request headers: %v", filterHeaders(req.Header))
		log.Debug("Request body: %s", string(reqData))

		rw := res.(martini.ResponseWriter)
		rh := newResponseHijacker(rw)
		c.MapTo(rh, (*http.ResponseWriter)(nil))
		c.MapTo(rh, (*martini.ResponseWriter)(nil))

		c.Next()

		response, _ := ioutil.ReadAll(rh.Response)
		log.Debug("Response headers: %v", rh.Header())
		log.Debug("Response body: %s", string(response))
		log.Info("Completed %v %s in %v", rw.Status(), http.StatusText(rw.Status()), time.Since(start))
	}
}

func filterHeaders(headers http.Header) http.Header {
	filtered := http.Header{}
	for k, v := range headers {
		if k == "X-Auth-Token" {
			filtered[k] = []string{"***"}
			continue
		}
		filtered[k] = v
	}
	return filtered
}

//IdentityService for user authentication & authorization
type IdentityService interface {
	GetTenantID(string) (string, error)
	GetTenantName(string) (string, error)
	VerifyToken(string) (schema.Authorization, error)
	GetServiceAuthorization() (schema.Authorization, error)
}

//HTTPJSONError helper for returning JSON errors
func HTTPJSONError(res http.ResponseWriter, err string, code int) {
	errorMessage := ""
	if code == http.StatusInternalServerError {
		log.Error(err)
	} else {
		errorMessage = err
		log.Notice(err)
	}
	response := map[string]interface{}{"error": errorMessage}
	responseJSON, _ := json.Marshal(response)
	http.Error(res, string(responseJSON), code)
}

//Authentication authenticates user using keystone
func Authentication() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, identityService IdentityService, c martini.Context) {
		if req.Method == "OPTIONS" {
			c.Next()
			return
		}
		//TODO(nati) make this configureable
		if strings.HasPrefix(req.URL.Path, "/webui/") {
			c.Next()
			return
		}
		if req.URL.Path == "/v2.0/tokens" {
			c.Next()
			return
		}
		authToken := req.Header.Get("X-Auth-Token")
		if authToken == "" {
			HTTPJSONError(res, "No X-Auth-Token", http.StatusUnauthorized)
			return
		}

		auth, err := identityService.VerifyToken(authToken)
		if err != nil {
			HTTPJSONError(res, err.Error(), http.StatusUnauthorized)
		}
		c.Map(auth)
		c.Next()
	}
}

//Context type
type Context map[string]interface{}

//WithContext injects new empty context object
func WithContext() martini.Handler {
	return func(c martini.Context) {
		c.Map(Context{})
	}
}

//Authorization checks user permissions against policy
func Authorization(action string) martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, auth schema.Authorization, context Context) {
		context["tenant_id"] = auth.TenantID()
		context["tenant_name"] = auth.TenantName()
		context["auth_token"] = auth.AuthToken()
		context["catalog"] = auth.Catalog()
		context["auth"] = auth
	}
}

// JSONURLs strips ".json" suffixes added to URLs
func JSONURLs() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, c martini.Context) {
		if !strings.Contains(req.URL.Path, "gohan") && !strings.Contains(req.URL.Path, "webui") {
			req.URL.Path = strings.TrimSuffix(req.URL.Path, ".json")
		}
		c.Next()
	}
}
