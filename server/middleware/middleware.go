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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/gophercloud/gophercloud"

	"github.com/cloudwan/gohan/cloud"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

const webuiPATH = "/webui/"

var (
	adminTenant = schema.Tenant{
		ID:   "admin",
		Name: "admin",
	}
	nobodyTenant = schema.Tenant{
		ID:   "nobody",
		Name: "nobody",
	}
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

func (rh *responseHijacker) CloseNotify() <-chan bool {
	return rh.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

//Logging logs requests and responses
func Logging() martini.Handler {
	return func(req *http.Request, rw http.ResponseWriter, c martini.Context, requestContext Context) {
		if strings.HasPrefix(req.URL.Path, webuiPATH) {
			c.Next()
			return
		}
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

		log.Info("[%s] Started %s %s for client %s data: %s",
			requestContext["trace_id"], req.Method, req.URL.String(), addr, string(reqData))
		log.Debug("[%s] Request headers: %v", requestContext["trace_id"], filterHeaders(req.Header))
		log.Debug("[%s] Request cookies: %v", requestContext["trace_id"], filterCookies(req.Cookies()))
		log.Debug("[%s] Request body: %s", requestContext["trace_id"], string(reqData))

		rh := newResponseHijacker(rw.(martini.ResponseWriter))
		c.MapTo(rh, (*http.ResponseWriter)(nil))
		c.MapTo(rh, (*martini.ResponseWriter)(nil))

		c.Next()

		response, _ := ioutil.ReadAll(rh.Response)
		log.Debug("[%s] Response headers: %v", requestContext["trace_id"], rh.Header())
		log.Debug("[%s] Response body: %s", requestContext["trace_id"], string(response))
		log.Info("[%s] Completed %v %s in %v", requestContext["trace_id"], rh.Status(), http.StatusText(rh.Status()), time.Since(start))
	}
}

func Metrics() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, c martini.Context) {
		c.Next()

		rw := res.(martini.ResponseWriter)
		metrics.UpdateCounter(1, "http.%s.status.%d", req.Method, rw.Status())
		if 200 <= rw.Status() && rw.Status() < 300 {
			metrics.UpdateCounter(1, "http.%s.ok", req.Method)
		} else {
			metrics.UpdateCounter(1, "http.%s.failed", req.Method)
		}

	}
}

func filterHeaders(headers http.Header) http.Header {
	filtered := http.Header{}
	for k, v := range headers {
		if k == "X-Auth-Token" || k == "Cookie" {
			filtered[k] = []string{"***"}
			continue
		}
		filtered[k] = v
	}

	return filtered
}

func filterCookies(cookies []*http.Cookie) []*http.Cookie {
	filtered := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie.Name == "Auth-Token" {
			newCookie := &http.Cookie{}
			*newCookie = *cookie
			cookie = newCookie

			cookie.Value = "***"
		}
		filtered = append(filtered, cookie)
	}
	return filtered
}

//IdentityService for user authentication & authorization
type IdentityService interface {
	GetTenantID(string) (string, error)
	GetTenantName(string) (string, error)
	VerifyToken(string) (schema.Authorization, error)
	GetServiceAuthorization() (schema.Authorization, error)
	GetServiceTokenID() string
	ValidateTenantID(string) (bool, error)
}

// CreateIdentityServiceFromConfig creates keystone identity from config
func CreateIdentityServiceFromConfig(config *util.Config) (IdentityService, error) {
	//TODO(marcin) remove this
	if config.GetBool("keystone/use_keystone", false) {
		if config.GetBool("keystone/fake", false) {
			//TODO(marcin) requests to fake server also get authenticated
			//             we need a separate routing Group
			log.Info("Debug Mode with Fake Keystone Server")
			return &FakeIdentity{}, nil

		}
		log.Info("Keystone backend server configured")
		var keystoneIdentity IdentityService
		keystoneIdentity, err := cloud.NewKeystoneIdentity(
			config.GetString("keystone/auth_url", "http://localhost:35357/v3"),
			config.GetString("keystone/user_name", "admin"),
			config.GetString("keystone/password", "password"),
			config.GetString("keystone/domain_name", "Default"),
			config.GetString("keystone/tenant_name", "admin"))
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed to create keystone identity service, err: %s", err))
		}
		if config.GetBool("keystone/use_auth_cache", false) {
			ttl, err := time.ParseDuration(config.GetString("keystone/cache_ttl", "15m"))
			if err != nil {
				log.Fatal("Failed to parse keystone cache TTL")
			}
			keystoneIdentity = NewCachedIdentityService(keystoneIdentity, ttl)
		}
		return keystoneIdentity, nil
	}
	return nil, fmt.Errorf("No identity service defined in config")
}

//NobodyResourceService contains a definition of nobody resources (that do not require authorization)
type NobodyResourceService interface {
	VerifyResourcePath(string) bool
}

// DefaultNobodyResourceService contains a definition of default nobody resources
type DefaultNobodyResourceService struct {
	resourcePathRegexes []*regexp.Regexp
}

// VerifyResourcePath verifies resource path
func (nrs *DefaultNobodyResourceService) VerifyResourcePath(resourcePath string) bool {
	for _, regex := range nrs.resourcePathRegexes {
		if regex.MatchString(resourcePath) {
			return true
		}
	}
	return false
}

// NewNobodyResourceService is a constructor for NobodyResourceService
func NewNobodyResourceService(nobodyResourcePathRegexes []*regexp.Regexp) NobodyResourceService {
	return &DefaultNobodyResourceService{resourcePathRegexes: nobodyResourcePathRegexes}
}

//NoIdentityService for disabled auth
type NoIdentityService struct {
}

//GetTenantID returns always admin
func (i *NoIdentityService) GetTenantID(string) (string, error) {
	return "admin", nil
}

//GetTenantName returns always admin
func (i *NoIdentityService) GetTenantName(string) (string, error) {
	return "admin", nil
}

//VerifyToken returns always authorization for admin
func (i *NoIdentityService) VerifyToken(string) (schema.Authorization, error) {
	auth := schema.NewAuthorizationBuilder().
		WithTenant(adminTenant).
		WithRoleIDs("admin").
		BuildAdmin()
	return auth, nil
}

//GetServiceAuthorization returns always authorization for admin
func (i *NoIdentityService) GetServiceAuthorization() (schema.Authorization, error) {
	auth := schema.NewAuthorizationBuilder().
		WithTenant(adminTenant).
		WithRoleIDs("admin").
		BuildAdmin()
	return auth, nil
}

//GetClient returns always nil
func (i *NoIdentityService) GetClient() *gophercloud.ServiceClient {
	return nil
}

//NobodyIdentityService for nobody auth
type NobodyIdentityService struct {
}

//GetTenantID returns always nobody
func (i *NobodyIdentityService) GetTenantID(string) (string, error) {
	return "nobody", nil
}

//GetTenantName returns always nobody
func (i *NobodyIdentityService) GetTenantName(string) (string, error) {
	return "nobody", nil
}

//VerifyToken returns always authorization for nobody
func (i *NobodyIdentityService) VerifyToken(string) (schema.Authorization, error) {
	auth := schema.NewAuthorizationBuilder().
		WithTenant(nobodyTenant).
		WithRoleIDs("Nobody").
		BuildScopedToTenant()
	return auth, nil
}

//GetServiceAuthorization returns always authorization for nobody
func (i *NobodyIdentityService) GetServiceAuthorization() (schema.Authorization, error) {
	auth := schema.NewAuthorizationBuilder().
		WithTenant(nobodyTenant).
		WithRoleIDs("Nobody").
		BuildScopedToTenant()
	return auth, nil
}

//GetClient returns always nil
func (i *NobodyIdentityService) GetClient() *gophercloud.ServiceClient {
	return nil
}

func (i *NobodyIdentityService) GetServiceTokenID() string {
	return ""
}

func (i *NobodyIdentityService) ValidateTenantID(id string) (bool, error) {
	return false, fmt.Errorf("invalid operation")
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
	return func(res http.ResponseWriter, req *http.Request, identityService IdentityService, nobodyResourceService NobodyResourceService, c martini.Context) {
		if req.Method == "OPTIONS" {
			c.Next()
			return
		}
		//TODO(nati) make this configurable
		if strings.HasPrefix(req.URL.Path, webuiPATH) {
			c.Next()
			return
		}

		if req.URL.Path == "/" || req.URL.Path == "/webui" {
			http.Redirect(res, req, webuiPATH, http.StatusTemporaryRedirect)
			return
		}

		if req.URL.Path == "/v2.0/tokens" {
			c.Next()
			return
		}

		if strings.HasPrefix(req.URL.Path, "/debug/pprof/") {
			c.Next()
			return
		}

		auth, err := authenticate(req, identityService, nobodyResourceService)
		if err != nil {
			HTTPJSONError(res, err.Error(), http.StatusUnauthorized)
			return
		}

		c.Map(auth)
		c.Next()
	}
}

func authenticate(req *http.Request, identityService IdentityService, nobodyResourceService NobodyResourceService) (schema.Authorization, error) {
	defer metrics.UpdateTimer(time.Now(), "req.auth")

	targetIdentityService, err := getIdentityService(req, identityService, nobodyResourceService)
	if err != nil {
		return nil, err
	}

	return targetIdentityService.VerifyToken(getAuthToken(req))
}

func getIdentityService(req *http.Request, identityService IdentityService, nobodyResourceService NobodyResourceService) (IdentityService, error) {
	if hasAuthToken(req) {
		return identityService, nil
	}

	if nobodyResourceService.VerifyResourcePath(req.URL.Path) {
		return &NobodyIdentityService{}, nil
	}

	return nil, fmt.Errorf("No X-Auth-Token")
}

func hasAuthToken(req *http.Request) bool {
	return getAuthToken(req) != ""
}

func getAuthToken(req *http.Request) string {
	authToken := req.Header.Get("X-Auth-Token")
	if authToken == "" {
		if authTokenCookie, err := req.Cookie("Auth-Token"); err == nil {
			authToken = authTokenCookie.Value
		}
	}
	return authToken
}

//Context type
type Context map[string]interface{}

//WithContext injects new empty context object
func WithContext() martini.Handler {
	return func(c martini.Context, res http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		requestContext := Context{
			"context": ctx,
		}
		c.Map(requestContext)

		go func(requestClosedCh <-chan bool) {
			select {
			case <-requestClosedCh:
				metrics.UpdateCounter(1, "req.peer_disconnect")
				cancel()
			case <-ctx.Done():
				break
			}
		}(res.(http.CloseNotifier).CloseNotify())

		c.Next()
	}
}

func Tracing() martini.Handler {
	return func(ctx Context) {
		ctx["trace_id"] = util.NewTraceID()
	}
}

//Authorization checks user permissions against policy
func Authorization(action string) martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, auth schema.Authorization, context Context) {
		context["tenant_id"] = auth.TenantID()
		context["domain_id"] = auth.DomainID()
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
