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

package otto_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	ottopkg "github.com/xyproto/otto"

	"os"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/server/resources"
	"github.com/cloudwan/gohan/util"
)

func newEnvironment() *otto.Environment {
	return otto.NewEnvironment("otto_test", testDB, &middleware.FakeIdentity{}, testSync)
}

var _ = Describe("Otto extension manager", func() {
	var (
		manager            *schema.Manager
		environmentManager *extension.Manager

		timeLimit  time.Duration
		timeLimits []*schema.PathEventTimeLimit
	)

	BeforeEach(func() {
		manager = schema.GetManager()
		environmentManager = extension.GetManager()

		timeLimit = time.Duration(1) * time.Second
		timeLimits = []*schema.PathEventTimeLimit{}
	})

	AfterEach(func() {
		tx, err := testDB.Begin()
		Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
		defer tx.Close()
		for _, schema := range schema.GetManager().Schemas() {
			if whitelist[schema.ID] {
				continue
			}
			err = clearTable(tx, schema)
			Expect(err).ToNot(HaveOccurred(), "Failed to clear table.")
		}
		err = tx.Commit()
		Expect(err).ToNot(HaveOccurred(), "Failed to commit transaction.")

		extension.ClearManager()
	})

	Describe("Loading an extension", func() {
		Context("When extension is not a valid JavaScript", func() {
			It("returns a meaningful compilation error", func() {
				goodExtension, err := schema.NewExtension(map[string]interface{}{
					"id":   "good_extension",
					"code": `gohan_register_handler("test_event", function(context) {});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				goodExtension.URL = "good_extension.js"

				badExtension, err := schema.NewExtension(map[string]interface{}{
					"id":   "bad_extension",
					"code": `gohan_register_handler("test_event", function(context {});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				badExtension.URL = "bad_extension.js"

				extensions := []*schema.Extension{goodExtension, badExtension}
				env := newEnvironment()
				err = env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")
				Expect(err).To(HaveOccurred(), "Expected compilation errors.")

				pattern := regexp.MustCompile(`^(?P<file>[^:]+).*Line\s(?P<line>\d+).*`)
				match := pattern.FindStringSubmatch(err.Error())
				Expect(len(match)).To(Equal(3))

				groups := make(map[string]string)
				for i, name := range pattern.SubexpNames() {
					groups[name] = match[i]
				}

				Expect(groups).To(HaveKeyWithValue("file", "bad_extension.js"))
				Expect(groups).To(HaveKeyWithValue("line", "1"))
			})
		})

		Context("When extension URL uses file:// protocol", func() {
			It("should read the file and run the extension", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id":   "test_extension",
					"url":  "file://../tests/sample_extension.js",
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())

				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["resp"]).ToNot(BeNil())
			})
		})

		Context("When extension URL uses http:// protocol", func() {
			It("should download and run the extension", func() {
				server := ghttp.NewServer()
				code := `
					gohan_register_handler("test_event", function(context){
						context.resp = "Hello";
					});
				`
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/extension.js"),
					ghttp.RespondWith(200, code),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id":   "test_extension",
					"url":  server.URL() + "/extension.js",
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["resp"]).ToNot(BeNil())
				server.Close()
			})
		})
	})

	Describe("Running an extension", func() {
		Context("When a runtime error occurs", func() {
			It("should return a meaningful error", func() {
				goodExtension, err := schema.NewExtension(map[string]interface{}{
					"id":   "good_extension",
					"code": `gohan_register_handler("test_event", function(context) {});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				goodExtension.URL = "good_extension.js"

				badExtension, err := schema.NewExtension(map[string]interface{}{
					"id": "bad_extension",
					"code": `gohan_register_handler("test_event", function foo(context) {
					var a = 5;
					console.log(b);
				});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				badExtension.URL = "bad_extension.js"

				extensions := []*schema.Extension{goodExtension, badExtension}
				env := newEnvironment()
				err = env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

				context := map[string]interface{}{
					"id": "test",
				}
				err = env.HandleEvent("test_event", context)
				Expect(err).To(HaveOccurred())

				Expect(regexp.MatchString(`ReferenceError:\s'b'`, err.Error())).To(BeTrue())

				pattern := regexp.MustCompile(`at\s(?P<function>\w+)\s\((?P<file>.*?):(?P<line>\d+).*?\)`)
				match := pattern.FindStringSubmatch(err.Error())
				Expect(len(match)).To(Equal(4))
				//FIXME(timorl): because of a throw bug in otto we cannot expect more meaningful errors
			})
		})

		Context("When a nil value is passed", func() {
			It("should be represented as null in the virtual machine", func() {
				goodExtension, err := schema.NewExtension(map[string]interface{}{
					"id": "good_extension",
					"code": `gohan_register_handler("test_event", function(context) {
						if (context.nulo === null) {
							context.respondo = "verdo"
						} else {
							context.respondo = "ne verdo"
						}
					});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				goodExtension.URL = "good_extension.js"

				extensions := []*schema.Extension{goodExtension}
				env := newEnvironment()
				env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

				context := map[string]interface{}{
					"id":   "test",
					"nulo": nil,
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("respondo", "verdo"))
			})
		})

		Context("When extension is running too long", func() {
			It("should be aborted in the middle of extension and cleaned up", func() {
				timeoutExtension, err := schema.NewExtension(map[string]interface{}{
					"id": "timeout_extension",
					"code": `
					var unclosedTx;
					gohan_register_handler("test_event", function(context) {
						unclosedTx = gohan_db_transaction();
						while(true) {
							// busy loop, but ok for this test
							var i = 1;
						};
					});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				timeoutExtension.URL = "timeout_extension.js"

				extensions := []*schema.Extension{timeoutExtension}
				env := newEnvironment()
				env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")
				cloned := env.Clone().(*otto.Environment)

				context := map[string]interface{}{
					"id": "test",
				}

				err = env.HandleEvent("test_event", context)
				Expect(err).To(MatchError(ContainSubstring("exceed timeout for extension execution")))
				txValue1, err := env.VM.Get("unclosedTx")
				Expect(err).ToNot(HaveOccurred())
				tx1, err := otto.GetTransaction(txValue1)
				Expect(err).ToNot(HaveOccurred())
				Expect(tx1.Closed()).To(BeTrue())

				err = cloned.HandleEvent("test_event", context)
				Expect(err).To(MatchError(ContainSubstring("exceed timeout for extension execution")))
				txValue2, err := cloned.VM.Get("unclosedTx")
				Expect(err).ToNot(HaveOccurred())
				tx2, err := otto.GetTransaction(txValue2)
				Expect(err).ToNot(HaveOccurred())
				Expect(tx1.Closed()).To(BeTrue())

				Expect(tx1).NotTo(BeIdenticalTo(tx2))

			})
		})

		Context("When extension is running too long after default time limit overridden", func() {
			It("should be aborted in the middle of extension with overridden time limit", func() {
				timeoutExtension, err := schema.NewExtension(map[string]interface{}{
					"id": "timeout_extension",
					"code": `gohan_register_handler("test_event", function(context) {
						while(true) {
							// busy loop, but ok for this test
							var i = 1;
						};
					});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				timeoutExtension.URL = "timeout_extension.js"

				extensions := []*schema.Extension{timeoutExtension}
				env := newEnvironment()
				env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

				// give the extension only 100 ms to execute
				env.SetEventTimeLimit("test_*", time.Duration(100)*time.Millisecond)

				context := map[string]interface{}{
					"id": "test",
				}

				timeStart := time.Now()
				err = env.HandleEvent("test_event", context)
				timeEnd := time.Now()
				timeDuration := timeEnd.Sub(timeStart)
				Expect(err).To(MatchError(ContainSubstring("exceed timeout for extension execution")))

				// the extension should have been aborted after about 100 ms
				// not even close to the default time limit which is 30 secs
				Expect(timeDuration).Should(BeNumerically("<", time.Millisecond*200))
			})

			It("should not be aborted in the middle of extension with overridden time limit if the override is for a different event", func() {
				timeoutExtension, err := schema.NewExtension(map[string]interface{}{
					"id": "timeout_extension",
					"code": `gohan_register_handler("test_event", function(context) {
						while(true) {
							// busy loop, but ok for this test
							var i = 1;
						};
					});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				timeoutExtension.URL = "timeout_extension.js"

				extensions := []*schema.Extension{timeoutExtension}
				env := newEnvironment()
				env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

				// give the extension only 100 ms to execute
				env.SetEventTimeLimit("unknown_*", time.Duration(100)*time.Millisecond)

				context := map[string]interface{}{
					"id": "test",
				}

				timeStart := time.Now()
				err = env.HandleEvent("test_event", context)
				timeEnd := time.Now()
				timeDuration := timeEnd.Sub(timeStart)
				Expect(err).To(MatchError(ContainSubstring("exceed timeout for extension execution")))

				// the extension must not be aborted after about 100 ms
				Expect(timeDuration).Should(BeNumerically(">", time.Millisecond*200))
			})
		})
	})

	Describe("Using gohan_http builtin", func() {
		Context("When the destination is reachable", func() {
			It("Should return the contents", func() {
				server := ghttp.NewServer()
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/contents"),
					ghttp.RespondWith(200, "HELLO"),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
						gohan_register_handler("test_event", function(context){
								context.resp = gohan_http('GET', '` + server.URL() + `/contents', {}, {});
						});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("status_code", "200")))
				Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("body", "HELLO")))
				server.Close()
			})
		})

		Context("When the destination is not reachable", func() {
			It("Should return the error", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
						gohan_register_handler("test_event", function(context){
							context.resp = gohan_http('GET', 'http://localhost:38000/contents', {}, {});
						});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("status", "err")))
				Expect(context).To(HaveKeyWithValue("resp", HaveKey("error")))
			})
		})

		Context("When the content type is not specified", func() {
			It("Should post the data as a JSON document", func() {
				server := ghttp.NewServer()
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/contents"),
					ghttp.RespondWith(200, "HELLO"),
					ghttp.VerifyJSON("{\"data\": \"posted_data\"}"),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
						gohan_register_handler("test_event", function(context){
							var d = { data: 'posted_data' }
							context.resp = gohan_http('POST', '` + server.URL() + `/contents', {}, d);
						});`,
					"path": ".*",
				})

				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("status_code", "200")))
				server.Close()
			})
		})

		Context("When the content type is text/plain", func() {
			It("Should post the data as plain text", func() {

				verifyPosted := func(expected string) http.HandlerFunc {
					return func(w http.ResponseWriter, req *http.Request) {
						body, _ := ioutil.ReadAll(req.Body)
						req.Body.Close()
						actual := strings.Trim(string(body), "\"")
						Ω(actual).Should(Equal(expected), "Post data Mismatch")
					}
				}

				postedString := "posted_string"
				server := ghttp.NewServer()
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/contents"),
					ghttp.RespondWith(200, "HELLO"),
					verifyPosted(postedString),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
						gohan_register_handler("test_event", function(context){
								var d = "` + postedString + `"
								context.resp = gohan_http('POST', '` + server.URL() + `/contents', {'Content-Type': 'text/plain' }, d);
						});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("status_code", "200")))
				server.Close()
			})
		})
	})

	Describe("Using gohan_raw_http builtin", func() {
		Context("When the destination is reachable", func() {
			It("Should return the contents", func() {
				server := ghttp.NewServer()
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/contents"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/json"}}),
					ghttp.VerifyBody([]byte("{\"input\":\"value\"}")),
					ghttp.RespondWith(200, []byte("{\"output\":\"value\"}")),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
						gohan_register_handler("test_event", function(context){
						    var headers = {
						        "content-type": "application/json"
						    };
						    var body = JSON.stringify({
						      input: "value"
						    });
						    context.response = gohan_raw_http('POST', '` + server.URL() + `/contents', headers, body);

						});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("response", HaveKeyWithValue("status_code", 200)))
				Expect(context).To(HaveKeyWithValue("response", HaveKeyWithValue("body", "{\"output\":\"value\"}")))
				server.Close()
			})

			It("Should not follow redirect", func() {
				server := ghttp.NewServer()
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/contents"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte("{\"input\":\"value\"}")),
					ghttp.RespondWith(302, []byte(""), http.Header{
						"Location": []string{server.URL() + "/redirect"},
					}),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `
						gohan_register_handler("test_event", function(context){
						    var headers = {
						        "content-type": "application/json"
						    };
						    var body = JSON.stringify({
						      input: "value"
						    });
						    context.response = gohan_raw_http('POST', '` + server.URL() + `/contents', headers, body);
						});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := newEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context).To(HaveKeyWithValue("response", HaveKeyWithValue("status_code", 302)))
				Expect(context).To(HaveKeyWithValue("response", HaveKeyWithValue("body", "")))
				server.Close()
			})
		})
	})

	Describe("Using gohan_config builtin", func() {
		It("Should return correct value", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
						gohan_register_handler("test_event", function(context){
							context.resp = gohan_config("database", "");
						});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			context := map[string]interface{}{}
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("connection", "test.db")))
			Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("type", "sqlite3")))
		})

		It("Should return correct value - key not found", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
						gohan_register_handler("test_event", function(context){
								context.resp = gohan_config("does not exist", false);
						});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			context := map[string]interface{}{}
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("resp", BeFalse()))
		})
	})

	Describe("Using gohan database manipulation builtins", func() {
		var (
			adminAuth     schema.Authorization
			auth          schema.Authorization
			context       middleware.Context
			schemaID      string
			path          string
			action        string
			currentSchema *schema.Schema
			extensions    []*schema.Extension
			env           extension.Environment
			events        map[string]string

			network1 map[string]interface{}
			network2 map[string]interface{}
			subnet1  map[string]interface{}
		)

		BeforeEach(func() {
			adminAuth = schema.NewAuthorization(adminTenantID, "admin", adminTokenID, []string{"admin"}, nil)
			auth = adminAuth

			context = middleware.Context{}

			events = map[string]string{}

			network1 = map[string]interface{}{
				"id":                "test1",
				"name":              "Rohan",
				"description":       "The proud horsemasters",
				"tenant_id":         adminTenantID,
				"providor_networks": map[string]interface{}{},
				"route_targets":     []interface{}{},
				"shared":            false,
			}
			network2 = map[string]interface{}{
				"id":                "test2",
				"name":              "Gondor",
				"description":       "Once glorious empire",
				"tenant_id":         adminTenantID,
				"providor_networks": map[string]interface{}{},
				"route_targets":     []interface{}{},
				"shared":            false,
			}
			subnet1 = map[string]interface{}{
				"id":        "test3",
				"name":      "Minas Tirith",
				"tenant_id": adminTenantID,
				"cidr":      "10.10.0.0/16",
			}
		})

		JustBeforeEach(func() {
			var ok bool
			currentSchema, ok = manager.Schema(schemaID)
			Expect(ok).To(BeTrue())

			path = currentSchema.GetPluralURL()

			policy, role := manager.PolicyValidate(action, path, auth)
			Expect(policy).NotTo(BeNil())
			context["policy"] = policy
			context["role"] = role
			context["tenant_id"] = auth.TenantID()
			context["tenant_name"] = auth.TenantName()
			context["auth_token"] = auth.AuthToken()
			context["catalog"] = auth.Catalog()
			context["auth"] = auth
			context["identity_service"] = &middleware.FakeIdentity{}

			env = newEnvironment()
			environmentManager.RegisterEnvironment(schemaID, env)
			extensions = []*schema.Extension{}
			for event, javascript := range events {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id":   event + "_extension",
					"code": `gohan_register_handler("` + event + `", function(context) {` + javascript + `});`,
					"path": path,
				})
				Expect(err).ToNot(HaveOccurred())
				extensions = append(extensions, extension)
			}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, path)).To(Succeed())
		})

		AfterEach(func() {
			tx, err := testDB.Begin()
			Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
			environmentManager.UnRegisterEnvironment(schemaID)
			defer tx.Close()
			for _, schema := range schema.GetManager().Schemas() {
				if whitelist[schema.ID] {
					continue
				}
				err = clearTable(tx, schema)
				Expect(err).ToNot(HaveOccurred(), "Failed to clear table.")
			}
			err = tx.Commit()
			Expect(err).ToNot(HaveOccurred(), "Failed to commit transaction.")
		})

		Describe("Using gohan_db_* builtins", func() {
			BeforeEach(func() {
				schemaID = "network"
				action = "read"
			})

			Context("When given a transaction", func() {
				It("Correctly handles CRUD operations", func() {
					tx, err := testDB.Begin()
					Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
					defer tx.Commit()

					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": `
						gohan_register_handler("test_event", function(context){
						  gohan_db_create(context.transaction,
						    'network', {
								'id':'test1',
								'name': 'name',
								'description': 'description',
								'providor_networks': {},
								'route_targets': [],
								'shared': false,
								'tenant_id': 'admin'});
						  context.network = gohan_db_fetch(context.transaction, 'network', 'test1', 'admin');
						  gohan_db_update(context.transaction,
						    'network', {'id':'test1', 'name': 'name_updated', 'tenant_id': 'admin'});
						  context.networks = gohan_db_list(context.transaction, 'network', {});
						  context.networks2 = gohan_db_list(context.transaction, 'network', {'shared': false});
						  gohan_db_delete(context.transaction, 'network', 'test1');
						  context.networks3 = gohan_db_list(context.transaction, 'network', {});
						});`,
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					extensions := []*schema.Extension{extension}
					env := newEnvironment()
					env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

					context := map[string]interface{}{
						"id":          "test",
						"transaction": tx,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())
					Expect(context["network"]).ToNot(BeNil())
					Expect(context["networks"]).ToNot(BeNil())
					Expect(context["networks2"]).ToNot(BeNil())
				})
			})

			Context("When given no transaction", func() {
				It("Correctly handles CRUD operations", func() {
					tx, err := testDB.Begin()
					Expect(err).ToNot(HaveOccurred(), "Failed to create transaction.")
					defer tx.Commit()

					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": `
						gohan_register_handler("test_event", function(context){
						  gohan_db_create(context.transaction,
						    'network', {
								'id':'test1',
								'name': 'name',
								'description': 'description',
								'providor_networks': {},
								'route_targets': [],
								'shared': false,
								'tenant_id': 'admin'});
						  context.network = gohan_db_fetch(context.transaction, 'network', 'test1', 'admin');
						  gohan_db_update(context.transaction,
						    'network', {'id':'test1', 'name': 'name_updated', 'tenant_id': 'admin'});
						  context.networks = gohan_db_list(context.transaction, 'network', {});
						  gohan_db_delete(context.transaction, 'network', 'test1');
						  context.networks2 = gohan_db_list(context.transaction, 'network', {});
						});`,
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					extensions := []*schema.Extension{extension}
					env := newEnvironment()
					env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

					context := map[string]interface{}{
						"id":          "test",
						"transaction": nil,
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())
					Expect(context["network"]).ToNot(BeNil())
					Expect(context["networks"]).ToNot(BeNil())
				})
			})
		})

		Describe("Using gohan_db_transaction", func() {
			BeforeEach(func() {
				schemaID = "network"
				action = "read"
			})

			Context("When given a transaction", func() {
				It("Correctly handles CRUD operations", func() {
					extension, err := schema.NewExtension(map[string]interface{}{
						"id": "test_extension",
						"code": `
						gohan_register_handler("test_event", function(context){
						  var tx = gohan_db_transaction();
						  gohan_db_create(tx,
						    'network', {
								'id':'test1',
								'name': 'name',
								'description': 'description',
								'providor_networks': {},
								'route_targets': [],
								'shared': false,
								'tenant_id': 'admin'});
						  context.network = gohan_db_fetch(tx, 'network', 'test1', 'admin');
						  gohan_db_update(tx,
						    'network', {'id':'test1', 'name': 'name_updated', 'tenant_id': 'admin'});
						  context.networks = gohan_db_list(tx, 'network', {});
						  gohan_db_delete(tx, 'network', 'test1');
						  context.networks2 = gohan_db_list(tx, 'network', {});
						  tx.Commit();
						});`,
						"path": ".*",
					})
					Expect(err).ToNot(HaveOccurred())
					extensions := []*schema.Extension{extension}
					env := newEnvironment()
					env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")

					context := map[string]interface{}{
						"id": "test",
					}
					Expect(env.HandleEvent("test_event", context)).To(Succeed())
					Expect(context["network"]).ToNot(BeNil())
					Expect(context["networks"]).ToNot(BeNil())
				})
			})
		})

		Describe("Using chaining builtins", func() {
			BeforeEach(func() {
				schemaID = "network"
			})

			Describe("Using gohan_model_list", func() {
				var (
					tx transaction.Transaction
				)

				BeforeEach(func() {
					resource, err := manager.LoadResource(schemaID, network1)
					Expect(err).NotTo(HaveOccurred())
					tx, err = testDB.Begin()
					Expect(err).NotTo(HaveOccurred())
					defer tx.Close()
					Expect(tx.Create(resource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())

					action = "read"
				})

				Describe("When invoked correctly", func() {
					Context("With transaction", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'network', {});`
						})

						It("Correctly lists elements", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(tx.Commit()).To(Succeed())
							Expect(context).To(HaveKeyWithValue("networks", ConsistOf(util.MatchAsJSON(network1))))
						})
					})

					Context("With a string filter", func() {
						BeforeEach(func() {
							events["test"] = `
								console.log(context)
								context.networks = gohan_model_list(context, 'network', {'id': 'test1'});`
						})

						It("Correctly lists elements", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context).To(HaveKeyWithValue("networks", ConsistOf(util.MatchAsJSON(network1))))
						})
					})

					Context("With an array filter", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'network', {'id': ['test1']});`
						})

						It("Correctly lists elements", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context).To(HaveKeyWithValue("networks", ConsistOf(util.MatchAsJSON(network1))))
						})
					})

					Context("With chained exception", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'network', {});`
							events["post_list_in_transaction"] = `
								throw new CustomException("Labori inteligente estas pli bona ol laboregi", 390);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context["exception"]).To(HaveKeyWithValue("message", ContainSubstring("Labori inteligente")))
							Expect(context["exception_message"]).To(ContainSubstring("Labori inteligente"))
						})
					})

					Context("With internal resource manipulation error", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'network', {});`
							events["post_list_in_transaction"] = `
								delete context.response;`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							err = env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("No response")))
						})
					})
				})

				Describe("When invoked incorrectly", func() {
					Context("With wrong number of arguments", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list();`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("arguments")))
						})
					})

					Context("With wrong schema ID", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'netwerk', {});`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("Unknown schema")))
						})
					})

					Context("With wrong filter", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'network', {'id': context.policy});`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("not a string")))
						})
					})

					Context("With wrong filter but array", func() {
						BeforeEach(func() {
							events["test"] = `
								context.networks = gohan_model_list(context, 'network', {'id': [context.policy]});`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("not a string")))
						})
					})
				})
			})

			Describe("Using gohan_model_fetch", func() {
				var (
					tx transaction.Transaction
				)

				BeforeEach(func() {
					resource, err := manager.LoadResource(schemaID, network1)
					Expect(err).NotTo(HaveOccurred())
					tx, err = testDB.Begin()
					Expect(err).NotTo(HaveOccurred())
					defer tx.Close()
					Expect(tx.Create(resource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())

					action = "read"
				})

				Describe("When invoked correctly", func() {
					Context("With transaction", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_fetch(context, 'network', 'test1', null);
								`
						})

						It("Correctly fetches the element", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(tx.Commit()).To(Succeed())
							By(fmt.Sprintf("%v", context))
							resultRaw, ok := context["network"]
							Expect(ok).To(BeTrue())
							_, ok = resultRaw.(map[string]interface{})
							Expect(ok).To(BeTrue())
						})
					})

					Context("Asking for a nonexistent resource", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_fetch(context, 'network', 'neEstas', null);`
						})

						It("Returns the not found error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context["exception"]).To(HaveKeyWithValue("problem", int(resources.NotFound)))
							Expect(context["exception_message"]).To(ContainSubstring("ResourceException"))
						})
					})

					Context("With chained exception", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_fetch(context, 'network', 'test1', null);`
							events["post_show_in_transaction"] = `
								throw new CustomException("Labori inteligente estas pli bona ol laboregi", 390);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context["exception"]).To(HaveKeyWithValue("message", ContainSubstring("Labori inteligente")))
							Expect(context["exception_message"]).To(ContainSubstring("Labori inteligente"))
							innerException := context["exception"].(map[string]interface{})["inner_exception"].(map[string]interface{})
							Expect(innerException).To(HaveKeyWithValue("code", int64(390)))
						})
					})

					Context("With internal resource manipulation error", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_fetch(context, 'network', 'test1', null);`
							events["post_show_in_transaction"] = `
								delete context.response;`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							err = env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("No response")))
						})
					})
				})

				Describe("When invoked incorrectly", func() {
					Context("With wrong number of arguments", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_fetch();`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("arguments")))
						})
					})

					Context("With wrong schema ID", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_fetch(context, 'netwerk', 'test1', null);`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("Unknown schema")))
						})
					})

				})
			})

			Describe("Using gohan_model_create", func() {
				var (
					tx transaction.Transaction
				)

				BeforeEach(func() {
					resource, err := manager.LoadResource(schemaID, network1)
					Expect(err).NotTo(HaveOccurred())
					tx, err = testDB.Begin()
					Expect(err).NotTo(HaveOccurred())
					defer tx.Close()
					Expect(tx.Create(resource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())

					action = "create"
				})

				Describe("When invoked correctly", func() {
					Context("With transaction", func() {
						BeforeEach(func() {
							script := `
							context.network = gohan_model_create(context, 'network', {'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v', 'route_targets': [], 'providor_networks': {}, 'shared': false});`
							events["test"] = fmt.Sprintf(script, "id", network2["id"], "name", network2["name"], "description", network2["description"], "tenant_id", network2["tenant_id"])
						})

						It("Correctly creates the element", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(tx.Commit()).To(Succeed())
							for key, value := range network2 {
								Expect(context).To(HaveKeyWithValue("network", HaveKeyWithValue(key, value)))
							}
						})
					})

					Context("With chained exception", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_create(context, 'network', {'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v', 'route_targets': [], 'providor_networks': {}, 'shared': false});`
							events["test"] = fmt.Sprintf(script, "id", network2["id"], "name", network2["name"], "description", network2["description"], "tenant_id", network2["tenant_id"])
							events["post_create_in_transaction"] = `
								throw new CustomException("Labori inteligente estas pli bona ol laboregi", 390);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context["exception"]).To(HaveKeyWithValue("message", ContainSubstring("Labori inteligente")))
							Expect(context["exception_message"]).To(ContainSubstring("Labori inteligente"))
							innerException := context["exception"].(map[string]interface{})["inner_exception"].(map[string]interface{})
							Expect(innerException).To(HaveKeyWithValue("code", int64(390)))
						})
					})

					Context("With internal resource manipulation error", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_create(context, 'network', {'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v', 'route_targets': [], 'providor_networks': {}, 'shared': false});`
							events["test"] = fmt.Sprintf(script, "id", network2["id"], "name", network2["name"], "description", network2["description"], "tenant_id", network2["tenant_id"])
							events["post_create_in_transaction"] = `
								delete context.response;`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx
							err = env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("No response")))
						})
					})
				})

				Describe("When invoked incorrectly", func() {
					Context("With wrong number of arguments", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_create();`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("arguments")))
						})
					})

					Context("With wrong schema ID", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_create(context, 'netwerk', {'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v'});`
							events["test"] = fmt.Sprintf(script, "id", network2["id"], "name", network2["name"], "description", network2["description"], "tenant_id", network2["tenant_id"])
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("Unknown schema")))
						})
					})

					Context("With wrong resource to create", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_create(context, 'network', 'Ne estas reto');`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("be of type 'Object'")))
						})
					})
				})
			})

			Describe("Using gohan_model_update", func() {
				var (
					tx transaction.Transaction
				)

				BeforeEach(func() {
					resource, err := manager.LoadResource(schemaID, network1)
					Expect(err).NotTo(HaveOccurred())
					tx, err = testDB.Begin()
					Expect(err).NotTo(HaveOccurred())
					defer tx.Close()
					Expect(tx.Create(resource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())

					action = "update"
				})

				Describe("When invoked correctly", func() {
					Context("With transaction", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_update(context, 'network', 'test1', {'%v': '%v'}, null);`
							events["test"] = fmt.Sprintf(script, "name", network2["name"])
						})

						It("Correctly updates the element", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(tx.Commit()).To(Succeed())
							Expect(context).To(HaveKeyWithValue("network", HaveKeyWithValue("name", network2["name"])))
						})
					})

					Context("With chained exception", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_update(context, 'network', 'test1', {'%v': '%v'}, null);`
							events["test"] = fmt.Sprintf(script, "name", network2["name"])
							events["post_update_in_transaction"] = `
								throw new CustomException("Labori inteligente estas pli bona ol laboregi", 390);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context["exception"]).To(HaveKeyWithValue("message", ContainSubstring("Labori inteligente")))
							Expect(context["exception_message"]).To(ContainSubstring("Labori inteligente"))
						})
					})

					Context("With internal resource manipulation error", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_update(context,  'network', 'test1', {'%v': '%v'}, null);`
							events["test"] = fmt.Sprintf(script, "name", network2["name"])
							events["post_update_in_transaction"] = `
								delete context.response;`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							err = env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("No response")))
						})
					})
				})

				Describe("When invoked incorrectly", func() {
					Context("With wrong number of arguments", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_update();`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("arguments")))
						})
					})

					Context("With wrong schema ID", func() {
						BeforeEach(func() {
							script := `
								context.network = gohan_model_update(context,  'netwerk', 'test1', {'%v': '%v'}, null);`
							events["test"] = fmt.Sprintf(script, "name", network2["name"])
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("Unknown schema")))
						})
					})

					Context("With wrong update data map", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_update(context, 'network', 'test1', 'Ne estas reto', null);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							err = env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("be of type 'Object'")))
						})
					})
				})
			})

			Describe("Using gohan_model_delete", func() {
				var (
					tx transaction.Transaction
				)

				BeforeEach(func() {
					resource, err := manager.LoadResource(schemaID, network1)
					Expect(err).NotTo(HaveOccurred())
					tx, err = testDB.Begin()
					Expect(err).NotTo(HaveOccurred())
					defer tx.Close()
					Expect(tx.Create(resource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())

					action = "delete"
				})

				Describe("When invoked correctly", func() {
					Context("With transaction", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_delete(context, 'network', 'test1');`
						})

						It("Correctly deletes the element", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(tx.Commit()).To(Succeed())
						})
					})

					Context("With chained exception", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_delete(context, 'network', 'test1');`
							events["post_delete_in_transaction"] = `
								throw new CustomException("Labori inteligente estas pli bona ol laboregi", 390);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							context["transaction"] = tx

							Expect(env.HandleEvent("test", context)).To(Succeed())
							Expect(context["exception"]).To(HaveKeyWithValue("message", ContainSubstring("Labori inteligente")))
							Expect(context["exception_message"]).To(ContainSubstring("Labori inteligente"))
						})
					})
				})

				Describe("When invoked incorrectly", func() {
					Context("With wrong number of arguments", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_delete();`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("arguments")))
						})
					})

					Context("With wrong schema ID", func() {
						BeforeEach(func() {
							events["test"] = `
								context.network = gohan_model_delete(context, 'netwerk', 'test1');`
						})

						It("Returns the proper error", func() {
							err := env.HandleEvent("test", context)
							Expect(err).To(MatchError(ContainSubstring("Unknown schema")))
						})
					})
				})
			})

			Describe("Actually chaining them", func() {
				var (
					tx                   transaction.Transaction
					createNetworkContext middleware.Context
					createSubnetContext  middleware.Context
					readSubnetContext    middleware.Context
					subnetEnv            extension.Environment
					subnetEvents         map[string]string
				)

				BeforeEach(func() {

					resource, err := manager.LoadResource(schemaID, network1)
					Expect(err).NotTo(HaveOccurred())
					tx, err = testDB.Begin()
					Expect(err).NotTo(HaveOccurred())
					defer tx.Close()
					Expect(tx.Create(resource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())

					action = "read"

					createNetworkContext = middleware.Context{}
					createSubnetContext = middleware.Context{}
					readSubnetContext = middleware.Context{}
					subnetEvents = map[string]string{}
				})

				JustBeforeEach(func() {
					curAction := "create"
					curSchemaID := "subnet"
					curSchema, ok := manager.Schema(curSchemaID)
					Expect(ok).To(BeTrue())

					curPath := curSchema.GetPluralURL()

					curPolicy, curRole := manager.PolicyValidate(curAction, curPath, auth)
					Expect(curPolicy).NotTo(BeNil())
					createSubnetContext["policy"] = curPolicy
					createSubnetContext["role"] = curRole
					createSubnetContext["tenant_id"] = auth.TenantID()
					createSubnetContext["tenant_name"] = auth.TenantName()
					createSubnetContext["auth_token"] = auth.AuthToken()
					createSubnetContext["catalog"] = auth.Catalog()
					createSubnetContext["auth"] = auth
					createSubnetContext["identity_service"] = &middleware.FakeIdentity{}

					curAction = "create"
					curPolicy, curRole = manager.PolicyValidate(curAction, curPath, auth)
					Expect(curPolicy).NotTo(BeNil())
					readSubnetContext["policy"] = curPolicy
					readSubnetContext["role"] = curRole
					readSubnetContext["tenant_id"] = auth.TenantID()
					readSubnetContext["tenant_name"] = auth.TenantName()
					readSubnetContext["auth_token"] = auth.AuthToken()
					readSubnetContext["catalog"] = auth.Catalog()
					readSubnetContext["auth"] = auth
					readSubnetContext["identity_service"] = &middleware.FakeIdentity{}

					subnetEnv = newEnvironment()
					environmentManager.RegisterEnvironment(curSchemaID, subnetEnv)
					curExtensions := []*schema.Extension{}
					for event, javascript := range subnetEvents {
						extension, err := schema.NewExtension(map[string]interface{}{
							"id":   event + "_extension",
							"code": `gohan_register_handler("` + event + `", function(context) {` + javascript + `});`,
							"path": curPath,
						})
						Expect(err).ToNot(HaveOccurred())
						curExtensions = append(curExtensions, extension)
					}
					Expect(subnetEnv.LoadExtensionsForPath(curExtensions, timeLimit, timeLimits, curPath)).To(Succeed())

					curAction = "create"
					curSchemaID = "network"
					curSchema, ok = manager.Schema(curSchemaID)
					Expect(ok).To(BeTrue())

					curPath = curSchema.GetPluralURL()

					curPolicy, curRole = manager.PolicyValidate(curAction, curPath, auth)
					Expect(curPolicy).NotTo(BeNil())
					createNetworkContext["policy"] = curPolicy
					createNetworkContext["role"] = curRole
					createNetworkContext["tenant_id"] = auth.TenantID()
					createNetworkContext["tenant_name"] = auth.TenantName()
					createNetworkContext["auth_token"] = auth.AuthToken()
					createNetworkContext["catalog"] = auth.Catalog()
					createNetworkContext["auth"] = auth
					createNetworkContext["identity_service"] = &middleware.FakeIdentity{}
				})

				Describe("When being chained", func() {
					Context("Without exceptions", func() {
						BeforeEach(func() {
							script := `
									context.network = gohan_model_create(context,
										'network', {'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v',
											'route_targets': [], 'providor_networks': {}, 'shared': false});`
							events["test"] = fmt.Sprintf(script, "id",
								network2["id"], "name", network2["name"],
								"description", network2["description"], "tenant_id", network2["tenant_id"])
							script = `
									console.log("model create");
									gohan_model_create(
										{
											transaction: context.transaction,
											policy: context.policy,
										},
										'subnet',
										{'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v',
										'network_id': context.response.network.id, 'description': "test"});`
							events["post_create_in_transaction"] = fmt.Sprintf(script, "id", subnet1["id"], "name", subnet1["name"], "tenant_id", subnet1["tenant_id"], "cidr", subnet1["cidr"])
							subnetEvents["test_subnet"] = `
							context.subnet = gohan_model_fetch(context, 'subnet', 'test3', null);
							`
						})

						It("Correctly handles chaining", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							createNetworkContext["transaction"] = tx
							By("Creating the network")
							Expect(env.HandleEvent("test", createNetworkContext)).To(Succeed())
							tx.Commit()
							tx.Close()
							By("network created")
							for key, value := range network2 {
								Expect(createNetworkContext).To(HaveKeyWithValue("network", HaveKeyWithValue(key, value)))
							}

							tx, err = testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							readSubnetContext["transaction"] = tx

							By("Also creating a default subnet for it")
							Expect(subnetEnv.HandleEvent("test_subnet", readSubnetContext)).To(Succeed())
							tx.Close()
							for key, value := range subnet1 {
								Expect(readSubnetContext).To(HaveKey("subnet"))
								Expect(readSubnetContext).To(HaveKeyWithValue("subnet", HaveKeyWithValue(key, value)))
							}
							Expect(readSubnetContext).To(HaveKeyWithValue("subnet", HaveKey("description")))
						})
					})

					Context("With an exception", func() {
						BeforeEach(func() {
							script := `
							context.network = gohan_model_create(context, 'network', {'%v': '%v', '%v': '%v', '%v': '%v', 'route_targets': [], 'providor_networks': {}, 'shared': false, 'description': ""});`

							events["test"] = fmt.Sprintf(script, "id", network2["id"], "name", network2["name"], "tenant_id", network2["tenant_id"])
							script = `
								gohan_model_create(context,
									'subnet', {'%v': '%v', '%v': '%v', '%v': '%v', '%v': '%v', 'network_id': context.response.id});`
							events["post_create_in_transaction"] = fmt.Sprintf(script, "id", subnet1["id"], "name", subnet1["name"], "tenant_id", subnet1["tenant_id"], "cidr", subnet1["cidr"])
							subnetEvents["pre_create_in_transaction"] = `
								throw new CustomException("Minas Tirith has fallen!", 390);`
						})

						It("Returns the proper error", func() {
							tx, err := testDB.Begin()
							Expect(err).NotTo(HaveOccurred())
							defer tx.Close()
							createNetworkContext["transaction"] = tx

							Expect(env.HandleEvent("test", createNetworkContext)).To(Succeed())
							Expect(createNetworkContext["exception"]).To(HaveKeyWithValue("message", ContainSubstring("Minas Tirith has fallen!")))
							Expect(createNetworkContext["exception_message"]).To(ContainSubstring("Minas Tirith has fallen!"))
						})
					})
				})
			})
		})
	})

	Describe("Using gohan_global", func() {
		var (
			loadingExtension       *schema.Extension
			loadingGlobalExtension *schema.Extension
		)

		BeforeEach(func() {
			loadingExtension, _ = schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					        function(context) {
							var abc = gohan_global("abc");
							context.resp = abc.test
						}
					);
					`,
				"path": ".*",
			})
			loadingGlobalExtension, _ = schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					        function(context) {
							var abc = gohan_process_global("abc");
							context.resp = abc.test
						}
					);
					`,
				"path": ".*",
			})
		})

		It("Should return same value for each environment when using process global", func() {
			savingExtension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					        function(context) {
							var abc = gohan_process_global("abc");
							abc.test = 123
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{savingExtension, loadingGlobalExtension}
			env1 := newEnvironment()
			env2 := newEnvironment()
			Expect(env1.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			Expect(env2.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			context := map[string]interface{}{}
			Expect(env1.HandleEvent("test_event", context)).To(Succeed())
			Expect(env2.HandleEvent("test_event", context)).To(Succeed())
			Expect(context["resp"]).To(Equal(int64(123)))
		})

		It("Should return different values for each environment when using gohan_global", func() {
			savingExtension1, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					        function(context) {
							var abc = gohan_global("abc");
							abc.test = 123
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			savingExtension2, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					        function(context) {
							var abc = gohan_global("abc");
							abc.test = 456
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			env1Extensions := []*schema.Extension{savingExtension1, loadingExtension}
			env2Extensions := []*schema.Extension{savingExtension2, loadingExtension}
			env1, env2 := newEnvironment(), newEnvironment()
			Expect(env1.LoadExtensionsForPath(env1Extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			Expect(env2.LoadExtensionsForPath(env2Extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			env1Context := map[string]interface{}{}
			env2Context := map[string]interface{}{}
			Expect(env1.HandleEvent("test_event", env1Context)).To(Succeed())
			Expect(env2.HandleEvent("test_event", env2Context)).To(Succeed())
			Expect(env1Context["resp"]).To(Equal(int64(123)))
			Expect(env2Context["resp"]).To(Equal(int64(456)))
		})

		It("Should return same value within cloned environment when using gohan_global", func() {
			savingExtension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					        function(context) {
							var abc = gohan_global("abc");
							abc.test = 123
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{savingExtension, loadingExtension}
			env1 := newEnvironment()
			env2 := env1.Clone()
			Expect(env1.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			Expect(env2.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			env1Context := map[string]interface{}{}
			env2Context := map[string]interface{}{}
			Expect(env1.HandleEvent("test_event", env1Context)).To(Succeed())
			Expect(env2.HandleEvent("test_event", env2Context)).To(Succeed())
			Expect(env1Context["resp"]).To(Equal(int64(123)))
			Expect(env2Context["resp"]).To(Equal(int64(123)))
		})
	})

	Describe("Using gohan_get_env builtin", func() {

		AfterEach(func() {
			os.Unsetenv("GOHAN_TEST_GET_ENV")
		})

		It("Should return proper environmental variable", func() {
			os.Setenv("GOHAN_TEST_GET_ENV", "123")
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler("test_event",
					 	function(context) {
					 		context.resp = gohan_get_env("GOHAN_TEST_GET_ENV", "");
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			context := map[string]interface{}{}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context["resp"]).To(Equal("123"))
		})

		It("Should return default value environmental variable is not set", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler("test_event",
					 	function(context) {
					 		context.resp = gohan_get_env("GOHAN_TEST_GET_ENV", "321");
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			context := map[string]interface{}{}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context["resp"]).To(Equal("321"))
		})
	})

	Describe("Using gohan_load_hook builtin", func() {
		It("Should use registered load hook when loading extension code", func() {
			hookExtension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					function loadHook(code, file) {
						return " \
						gohan_register_handler( \
							\"test_event\", \
						 	function(context) { \
								context.resp = 321; \
							} \
						); ";
					}
					gohan_register_handler(
						"reg_hook",
					 	function(context) {
					 		gohan_load_hook("loadHook");
						}
					);
					`,
				"path": "hook",
			})
			Expect(err).ToNot(HaveOccurred())
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler("test_event",
					 	function(context) {
					 		context.resp = 123;
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{hookExtension, extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "hook")).To(Succeed())
			context := map[string]interface{}{}
			Expect(env.HandleEvent("reg_hook", context)).To(Succeed())
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context["resp"]).To(Equal(int64(321)))
		})
	})

	Describe("Using gohan_sync_fetch builtin", func() {
		It("Should fetch sync", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					 	function(context) {
							context.resp = gohan_sync_fetch("/gohan_sync_fetch_test");
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			context := map[string]interface{}{}
			env.Sync.Delete("/gohan_sync_fetch_test", false)
			env.Sync.Update("/gohan_sync_fetch_test", "{}")
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("value", "{}")))
		})
	})
	Describe("Using gohan_sync_watch builtin", func() {
		It("Should timeout with no events", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					 	function(context) {
							context.resp = gohan_sync_watch("/gohan_sync_watch_test", 500, 0);
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			context := map[string]interface{}{}
			env.Sync.Delete("/gohan_sync_watch_test", false)
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("resp", HaveLen(0)))
		})

		It("Should timeout with one event", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					 	function(context) {
							context.resp = gohan_sync_watch("/gohan_sync_watch_test", 500, 0);
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			context := map[string]interface{}{}
			env.Sync.Delete("/gohan_sync_watch_test", false)
			go func() {
				time.Sleep(time.Duration(200) * time.Millisecond)
				env.Sync.Update("/gohan_sync_watch_test", "{}")
			}()
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("action", "set")))
			Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("key", "/gohan_sync_watch_test")))
			Expect(context).To(HaveKeyWithValue("resp", HaveKeyWithValue("data", map[string]interface{}{})))
		})
	})

	Describe("Using gohan_sync_delete builtin", func() {
		It("Should delete sync", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					 	function(context) {
							gohan_sync_delete("/gohan_sync_delete_test");
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			context := map[string]interface{}{}
			env.Sync.Delete("/gohan_sync_delete_test", false)
			env.Sync.Update("/gohan_sync_delete_test", "{}")
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			node, err := env.Sync.Fetch("/gohan_sync_delete_test")
			Expect(err).To(HaveOccurred())
			Expect(node).To(BeNil())
		})
	})

	Describe("Using gohan_sync_delete builtin with prefix", func() {
		It("Should delete sync", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "test_extension",
				"code": `
					gohan_register_handler(
						"test_event",
					 	function(context) {
							gohan_sync_delete("/gohan_sync_delete_test", true);
						}
					);
					`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env := newEnvironment()
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())
			context := map[string]interface{}{}
			env.Sync.Delete("/gohan_sync_delete_test", true)
			env.Sync.Update("/gohan_sync_delete_test/child1", "bla")
			env.Sync.Update("/gohan_sync_delete_test/child2", "bar")
			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			node, err := env.Sync.Fetch("/gohan_sync_delete_test")
			Expect(err).To(HaveOccurred())
			Expect(node).To(BeNil())
		})
	})
	var _ = Describe("Concurrency race", func() {
		var (
			env *otto.Environment
		)
		channel := make(chan string)
		Context("Given environment", func() {
			BeforeEach(func() {
				env = newEnvironment()
				env.SetUp()
				vm := env.VM
				builtins := map[string]interface{}{
					"test_consume": func(call ottopkg.FunctionCall) ottopkg.Value {
						result := <-channel
						ottoResult, _ := vm.ToValue(result)
						return ottoResult
					},
					"test_produce": func(call ottopkg.FunctionCall) ottopkg.Value {
						ottoProduct := otto.ConvertOttoToGo(call.Argument(0))
						product := otto.ConvertOttoToGo(ottoProduct).(string)
						channel <- product
						return ottopkg.NullValue()
					},
				}
				for name, object := range builtins {
					vm.Set(name, object)
				}

				Expect(env.Load("<race_test>", `
				var produce = function() {
					for (var i = 0; i < 10; i++) {
						console.log("producing:", i);
						test_produce(i.toString());
					}
				};

				var consume = function() {
					for (var i = 0; i < 10; i++) {
						var result = test_consume();
						console.log("consumed:", result);
					}
				}`)).To(BeNil())
				environmentManager.RegisterEnvironment("test_race", env)
			})

			It("Should work", func() {
				var consumerError = make(chan error)
				var producerError = make(chan error)

				go func() {
					testEnv, _ := environmentManager.GetEnvironment("test_race")
					ottoEnv := testEnv.(*otto.Environment)
					_, err := ottoEnv.VM.Call("consume", nil)
					consumerError <- err
				}()

				go func() {
					testEnv, _ := environmentManager.GetEnvironment("test_race")
					ottoEnv := testEnv.(*otto.Environment)
					_, err := ottoEnv.VM.Call("produce", nil)
					producerError <- err
				}()

				select {
				case err := <-consumerError:
					Expect(err).To(BeNil())
				case <-time.After(1 * time.Second):
					Fail("Timeout when waiting for consumer")
				}
				select {
				case err := <-producerError:
					Expect(err).To(BeNil())
				case <-time.After(1 * time.Second):
					Fail("Timeout when waiting for producer")
				}
			})
		})
	})
	var _ = Describe("Timeout", func() {
		Context("stops if execution time exceeds timelimit", func() {
			It("Should work", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "infinite_loop",
					"code": `
						gohan_register_handler("test_event", function(context){
							while(true){}
						});`,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := otto.NewEnvironment("otto_test", testDB, &middleware.FakeIdentity{}, testSync)
				Expect(env.LoadExtensionsForPath(extensions, time.Duration(100), timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).ToNot(Succeed())
			})
		})
	})

	Describe("IsEventHandled", func() {
		var (
			env     extension.Environment
			context = map[string]interface{}{"id": "test"}
		)

		createEnv := func(code string) {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id":   "no_handler",
				"code": code,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			env = otto.NewEnvironment("otto_test", testDB, &middleware.FakeIdentity{}, testSync)
			Expect(env.LoadExtensionsForPath(extensions, time.Duration(100), timeLimits, "test_path")).To(Succeed())
		}

		It("Should return false if no handler registered", func() {
			createEnv("")

			Expect(env.IsEventHandled("test_event", context)).To(BeFalse())
		})

		It("Should return true if a handler is registered", func() {
			code := `gohan_register_handler("test_event", function(context){});`
			createEnv(code)

			Expect(env.IsEventHandled("test_event", context)).To(BeTrue())
		})
	})
})

var _ = Describe("Using gohan_file builtin", func() {
	var (
		context map[string]interface{}
		env     *otto.Environment

		timeLimit  time.Duration
		timeLimits []*schema.PathEventTimeLimit
	)

	BeforeEach(func() {
		context = map[string]interface{}{
			"id": "test",
		}
		env = newEnvironment()

		timeLimit = time.Duration(10) * time.Second
		timeLimits = []*schema.PathEventTimeLimit{}
	})

	Context("List files", func() {
		It("Should work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "list_files",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.list = gohan_file_list('./');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKey("list"))
			Expect(context["list"]).ToNot(BeNil())
		})
		It("Shouldn't work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "list_files",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.list = gohan_file_list('./doesnt_exist');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).ToNot(Succeed())
			Expect(context).ToNot(HaveKey("list"))
		})
	})
	Context("Read file", func() {
		It("Should work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.list = gohan_file_read('./test.db');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKey("list"))
			Expect(context["list"]).ToNot(BeNil())
		})
		It("Shouldn't work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.list = gohan_file_read('./doesnt_exist');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).ToNot(Succeed())
			Expect(context).ToNot(HaveKey("list"))
		})
	})
	Context("Read file with CD to the JavaScript file location", func() {
		It("Should work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file_cd",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.list = gohan_file_read_cd('./test.db');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKey("list"))
			Expect(context["list"]).ToNot(BeNil())
		})
		It("Shouldn't work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file_cd",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.list = gohan_file_read_cd('./doesnt_exist');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).ToNot(Succeed())
			Expect(context).ToNot(HaveKey("list"))
		})
	})
	Context("Is dir", func() {
		It("Is dir", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.dir = gohan_file_dir('./');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("dir", true))
		})
		It("Isn't dir", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.dir= gohan_file_dir('./test.db');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).To(Succeed())
			Expect(context).To(HaveKeyWithValue("dir", false))
		})
		It("Shouldn't work", func() {
			extension, err := schema.NewExtension(map[string]interface{}{
				"id": "read_file",
				"code": `
					gohan_register_handler("test_event", function(context){
						context.dir = gohan_file_dir('./doesnt_exist');
					});`,
				"path": ".*",
			})
			Expect(err).ToNot(HaveOccurred())
			extensions := []*schema.Extension{extension}
			Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

			Expect(env.HandleEvent("test_event", context)).ToNot(Succeed())
			Expect(context).ToNot(HaveKey("dir"))
		})
	})
})
