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

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/rackspace/gophercloud"
	"github.com/twinj/uuid"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
)

var _ = Describe("CLI functions", func() {
	var (
		server           *ghttp.Server
		provider         *gophercloud.ProviderClient
		gohanEndpointURL string
		err              error
		netSchema        *schema.Schema
	)

	BeforeEach(func() {
		logOutput = ioutil.Discard
		server = ghttp.NewServer()
		os.Setenv("OS_USERNAME", "admin")
		os.Setenv("OS_PASSWORD", "password")
		os.Setenv("OS_TENANT_NAME", "admin")
		os.Setenv("OS_AUTH_URL", server.URL()+"/v2.0/")
		os.Setenv("GOHAN_SERVICE_NAME", "gohan")
		os.Setenv("GOHAN_REGION", "RegionOne")
		os.Setenv("GOHAN_SCHEMA_URL", "/gohan/v0.1/schemas")
		os.Setenv("GOHAN_CACHE_SCHEMAS", "false")
		os.Setenv("GOHAN_CACHE_TIMEOUT", "1ns")
	})

	AfterEach(func() {
		os.Unsetenv("OS_USERNAME")
		os.Unsetenv("OS_PASSWORD")
		os.Unsetenv("OS_TENANT_NAME")
		os.Unsetenv("OS_AUTH_URL")
		os.Unsetenv("GOHAN_SERVICE_NAME")
		os.Unsetenv("GOHAN_REGION")
		os.Unsetenv("GOHAN_SCHEMA_URL")
		os.Unsetenv("GOHAN_ENDPOINT_URL")
		server.Close()
	})

	Describe("GohanClientCLIOpts constructor", func() {
		It("Should create GohanClientCLIOpts successfully - without GOHAN_ENDPOINT_URL", func() {
			opts, err := NewOptsFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(opts).ToNot(BeNil())
		})

		It("Should create GohanClientCLIOpts successfully - without GOHAN_ENDPOINT_URL", func() {
			os.Unsetenv("GOHAN_SERVICE_NAME")
			os.Unsetenv("GOHAN_REGION")
			os.Setenv("GOHAN_ENDPOINT_URL", "127.0.0.1")
			opts, err := NewOptsFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(opts).ToNot(BeNil())
		})

		It("Should show error - GOHAN_SERVICE_NAME not set", func() {
			os.Unsetenv("GOHAN_SERVICE_NAME")
			opts, err := NewOptsFromEnv()
			Expect(opts).To(BeNil())
			Expect(err).To(MatchError("Environment variable GOHAN_SERVICE_NAME needs to be set"))
		})

		It("Should show error - GOHAN_REGION not set", func() {
			os.Unsetenv("GOHAN_REGION")
			opts, err := NewOptsFromEnv()
			Expect(opts).To(BeNil())
			Expect(err).To(MatchError("Environment variable GOHAN_REGION needs to be set"))
		})

		It("Should show error - GOHAN_SCHEMA_URL not set", func() {
			os.Unsetenv("GOHAN_SCHEMA_URL")
			opts, err := NewOptsFromEnv()
			Expect(opts).To(BeNil())
			Expect(err).To(MatchError("Environment variable GOHAN_SCHEMA_URL needs to be set"))
		})

		It("Should show error - error parsing GOHAN_CACHE_SCHEMAS", func() {
			os.Setenv("GOHAN_CACHE_SCHEMAS", "potato")
			opts, err := NewOptsFromEnv()
			Expect(opts).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Should show error - error parsing GOHAN_CACHE_TIMEOUT", func() {
			os.Setenv("GOHAN_CACHE_TIMEOUT", "this is not time")
			opts, err := NewOptsFromEnv()
			Expect(opts).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GohanClientCLI constructor", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v2.0/tokens"),
					ghttp.RespondWithJSONEncoded(200, getAuthResponse(server.URL())),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/gohan/v0.1/schemas"),
					ghttp.RespondWithJSONEncoded(200, getSchemasResponse()),
				),
			)
		})

		It("Should create GohanClientCLI instance successfully", func() {
			opts, _ := NewOptsFromEnv()
			gohanClientCLI, err := NewGohanClientCLI(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(gohanClientCLI).ToNot(BeNil())
		})

		It("Should show error - authentication failed", func() {
			server.SetHandler(0, ghttp.RespondWith(401, nil))
			opts, _ := NewOptsFromEnv()
			gohanClientCLI, err := NewGohanClientCLI(opts)
			Expect(gohanClientCLI).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Should show error - getting endpoint url failed", func() {
			os.Setenv("GOHAN_SERVICE_NAME", "wrong")
			opts, _ := NewOptsFromEnv()
			gohanClientCLI, err := NewGohanClientCLI(opts)
			Expect(gohanClientCLI).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Should show error - getting schemas failed", func() {
			server.SetHandler(1, ghttp.RespondWith(404, nil))
			opts, _ := NewOptsFromEnv()
			gohanClientCLI, err := NewGohanClientCLI(opts)
			Expect(gohanClientCLI).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Should show error - getting cached schemas failed", func() {
			os.Setenv("GOHAN_CACHE_SCHEMAS", "true")
			os.Setenv("GOHAN_CACHE_PATH", "wrong")
			server.SetHandler(1, ghttp.RespondWith(404, nil))
			opts, _ := NewOptsFromEnv()
			gohanClientCLI, err := NewGohanClientCLI(opts)
			Expect(gohanClientCLI).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Should show error - setting cached schemas failed", func() {
			file, _ := ioutil.TempDir(os.TempDir(), "gohan_test")
			os.Setenv("GOHAN_CACHE_SCHEMAS", "true")
			os.Setenv("GOHAN_CACHE_PATH", file)
			opts, _ := NewOptsFromEnv()
			gohanClientCLI, err := NewGohanClientCLI(opts)
			Expect(gohanClientCLI).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Keystone interaction", func() {
		Describe("Authentication", func() {
			It("Should authenticate successfully", func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v2.0/tokens"),
					ghttp.VerifyJSONRepresenting(getAuthRequest()),
					ghttp.RespondWithJSONEncoded(200, getAuthResponse(server.URL())),
				))
				provider, err = getProviderClient()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.IdentityBase).To(Equal(server.URL() + "/"))
				Expect(provider.IdentityEndpoint).To(Equal(server.URL() + "/v2.0/"))
				Expect(provider.TokenID).To(Equal("admin_token"))
			})

			It("Should show error - OS_AUTH_URL not set", func() {
				os.Unsetenv("OS_AUTH_URL")
				provider, err = getProviderClient()
				Expect(provider).To(BeNil())
				Expect(err).To(MatchError("Environment variable OS_AUTH_URL needs to be set."))
			})

			It("Should show error - OS_USERNAME not set", func() {
				os.Unsetenv("OS_USERNAME")
				provider, err = getProviderClient()
				Expect(provider).To(BeNil())
				Expect(err).To(MatchError("Environment variable OS_USERNAME, OS_USERID, or OS_TOKEN needs to be set."))
			})

			It("Should show error - OS_PASSWORD not set", func() {
				os.Unsetenv("OS_PASSWORD")
				provider, err = getProviderClient()
				Expect(provider).To(BeNil())
				Expect(err).To(MatchError("Environment variable OS_PASSWORD or OS_TOKEN needs to be set."))
			})

			It("Should show error - domain not specified", func() {
				server.AppendHandlers(ghttp.RespondWith(400, "You must provide exactly one of DomainID or DomainName to authenticate by Username"))
				provider, err = getProviderClient()
				Expect(provider).To(BeNil())
				Expect(err).To(MatchError("Environment variable OS_DOMAIN_ID or OS_DOMAIN_NAME needs to be set"))
			})
		})

		Describe("Getting endpoint URL", func() {
			var gohanClientCLI GohanClientCLI

			BeforeEach(func() {
				server.AppendHandlers(ghttp.RespondWithJSONEncoded(200, getAuthResponse(server.URL())))
				provider, _ = getProviderClient()
			})

			It("Should get Gohan endpoint URL successfully", func() {
				opts, _ := NewOptsFromEnv()
				gohanClientCLI = GohanClientCLI{opts: opts}
				gohanEndpointURL, err = gohanClientCLI.getGohanEndpointURL(provider)
				Expect(err).ToNot(HaveOccurred())
				Expect(gohanEndpointURL).To(Equal(server.URL()))
			})

			It("Should show error - no endpoint", func() {
				os.Setenv("GOHAN_SERVICE_NAME", "wrongGohanServiceName")
				opts, _ := NewOptsFromEnv()
				gohanClientCLI = GohanClientCLI{opts: opts}
				gohanEndpointURL, err = gohanClientCLI.getGohanEndpointURL(provider)
				Expect(gohanEndpointURL).To(Equal(""))
				Expect(err).To(MatchError("No suitable endpoint could be found in the service catalog."))
			})
		})
	})

	Describe("Gohan interaction", func() {
		var gohanClientCLI GohanClientCLI

		BeforeEach(func() {
			server.AppendHandlers(ghttp.RespondWithJSONEncoded(200, getAuthResponse(server.URL())))
			provider, _ = getProviderClient()
			opts := GohanClientCLIOpts{
				gohanSchemaURL:   "/gohan/v0.1/schemas",
				gohanEndpointURL: server.URL(),
				outputFormat:     "json",
				logLevel:         l.CRITICAL,
			}
			gohanClientCLI = GohanClientCLI{
				provider: provider,
				opts:     &opts,
			}
		})

		Describe("Displaying commands info", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/gohan/v0.1/schemas"),
						ghttp.RespondWithJSONEncoded(200, getActionSchemasResponse()),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2.0/actions/do_a"),
						ghttp.RespondWithJSONEncoded(200, doA()),
					),
				)
			})

			It("Should show custom commands in schemas info", func() {
				result, err := gohanClientCLI.ExecuteCommand("action", []string{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(`
  action
  -----------------------------------
  Description: Action

  Properties:
    - a [ number ]: <no value> <no value>
    - b [ boolean ]: <no value> <no value>
    - c [ string ]: <no value> <no value>
    - d [ object ]: <no value> <no value>

  Commands:

  - List all action resources

    gohan client action list

  - Show a action resources

    gohan client action show [ID]

  - Create a action resources

    gohan client action create \
      --a [ number ] \
      --b [ boolean ] \
      --c [ string ] \
      --d [ object ] \


  - Update action resources

    gohan client action set \
      --a [number ] \
      --b [boolean ] \
      --c [string ] \
      --d [object ] \
      [ID]

  - Delete action resources

    gohan client action delete [ID]

  Custom commands:

  - do_a

    gohan client action do_a

  - do_b

    gohan client action do_b [ID]

  - do_c

    gohan client action do_c [Input]
      Input type: [ number ]

  - do_d

    gohan client action do_d [Input] [ID]
      Input type: [ object ]
      Input properties:
        --a_in [ number ]
        --b_in [ string ]
        --c_in [ boolean ]
`))
			})

			It("Should show action parameters when inspecting action", func() {
				gohanClientCLI.schemas, err = gohanClientCLI.getSchemas()
				Expect(err).ToNot(HaveOccurred())
				gohanClientCLI.commands = gohanClientCLI.getCommands()
				result, err := gohanClientCLI.ExecuteCommand("action do_d", []string{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(`
  action
  -----------------------------------

  Command: do_d
  gohan client action do_d [Input] [ID]
    Input type: [ object ]
    Input properties:
      --a_in [ number ]
      --b_in [ string ]
      --c_in [ boolean ]
`))
			})

			It("Should execute action with no parameters", func() {
				gohanClientCLI.schemas, err = gohanClientCLI.getSchemas()
				Expect(err).ToNot(HaveOccurred())
				gohanClientCLI.commands = gohanClientCLI.getCommands()
				result, err := gohanClientCLI.ExecuteCommand("action do_a", []string{})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("\"" + doA() + "\""))
			})
		})

		Describe("Reading schemas", func() {
			It("Should get Gohan schemas successfully", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/gohan/v0.1/schemas"),
						ghttp.RespondWithJSONEncoded(200, getSchemasResponse()),
					),
				)
				schemas, err := gohanClientCLI.getSchemas()
				Expect(err).ToNot(HaveOccurred())
				Expect(schemas).ToNot(BeNil())
				compareSchemas(schemas, getSchemas())
			})

			It("Should show error - Could not retrieve schemas - 404", func() {
				server.AppendHandlers(
					ghttp.RespondWith(404, nil),
				)
				schemas, err := gohanClientCLI.getSchemas()
				Expect(schemas).To(BeNil())
				message := fmt.Sprintf("Expected HTTP response code [200] when accessing "+
					"[GET %v/gohan/v0.1/schemas], but got 404 instead\n", server.URL())
				Expect(err).To(MatchError(message))
			})

			It("Should show error - Could not retrieve schemas - wrong response JSON", func() {
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, map[string]interface{}{}),
				)
				schemas, err := gohanClientCLI.getSchemas()
				Expect(schemas).To(BeNil())
				Expect(err).To(MatchError("No 'schemas' key in response JSON"))
			})

			It("Should show error - Could not retrieve schemas - could not parse schema", func() {
				wrongSchemas := map[string]interface{}{
					"schemas": []map[string]interface{}{
						map[string]interface{}{
							"wrong": "key",
						},
					},
				}
				server.AppendHandlers(
					ghttp.RespondWithJSONEncoded(200, wrongSchemas),
				)
				schemas, err := gohanClientCLI.getSchemas()
				Expect(schemas).To(BeNil())
				Expect(err).To(MatchError(ContainSubstring("Could not parse schemas")))
			})
		})

		Describe("Caching", func() {

			Describe("Get cache", func() {
				var file *os.File

				BeforeEach(func() {
					file, _ = ioutil.TempFile(os.TempDir(), "gohan_test")
				})

				AfterEach(func() {
					file.Close()
				})

				It("Should get cached schemas successfully", func() {
					cache := map[string]interface{}{
						"Expire":  "2999-08-13T15:37:25.234607353+02:00",
						"Schemas": getSchemas(),
					}
					cacheString, _ := json.Marshal(cache)
					file.WriteString(fmt.Sprintf("%s", cacheString))

					gohanClientCLI.opts.cachePath = file.Name()
					schemas, err := gohanClientCLI.getCachedSchemas()
					Expect(err).ToNot(HaveOccurred())
					Expect(schemas).To(HaveLen(2))
					Expect(schemas[0].ID).To(Equal("castle"))
					Expect(schemas[1].ID).To(Equal("tower"))
				})

				It("Should bypass cache - error reading file", func() {
					server.AppendHandlers(ghttp.RespondWithJSONEncoded(200, getSchemasResponse()))
					file.Chmod(os.ModeDir)
					gohanClientCLI.opts.cachePath = file.Name()
					schemas, err := gohanClientCLI.getCachedSchemas()
					Expect(err).ToNot(HaveOccurred())
					Expect(schemas).To(HaveLen(2))
					Expect(schemas[0].ID).To(Equal("castle"))
					Expect(schemas[1].ID).To(Equal("tower"))
				})

				It("Should bypass cache - error unmarshalling cache", func() {
					server.AppendHandlers(ghttp.RespondWithJSONEncoded(200, getSchemasResponse()))
					gohanClientCLI.opts.cachePath = file.Name()
					schemas, err := gohanClientCLI.getCachedSchemas()
					Expect(err).ToNot(HaveOccurred())
					Expect(schemas).To(HaveLen(2))
					Expect(schemas[0].ID).To(Equal("castle"))
					Expect(schemas[1].ID).To(Equal("tower"))
				})

				It("Should bypass cache - cache expired", func() {
					cache := map[string]interface{}{
						"Expire":  "1942-08-13T15:37:25.234607353+02:00",
						"Schemas": getSchemas(),
					}
					cacheString, _ := json.Marshal(cache)
					file.WriteString(fmt.Sprintf("%s", cacheString))

					server.AppendHandlers(ghttp.RespondWithJSONEncoded(200, getSchemasResponse()))
					gohanClientCLI.opts.cachePath = file.Name()
					schemas, err := gohanClientCLI.getCachedSchemas()
					Expect(err).ToNot(HaveOccurred())
					Expect(schemas).To(HaveLen(2))
					Expect(schemas[0].ID).To(Equal("castle"))
					Expect(schemas[1].ID).To(Equal("tower"))
				})
			})

			Describe("Set cache", func() {
				BeforeEach(func() {
					gohanClientCLI.schemas = getSchemas()
					gohanClientCLI.opts.cachePath = "/tmp/gohan_test_" + uuid.NewV4().String()
				})

				It("Should set cache successfully", func() {
					err = gohanClientCLI.setCachedSchemas()
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.cachePath).To(BeARegularFile())
					file, err := ioutil.ReadFile(gohanClientCLI.opts.cachePath)
					Expect(err).ToNot(HaveOccurred())
					cache := Cache{}
					err = json.Unmarshal(file, &cache)
					Expect(err).ToNot(HaveOccurred())
					Expect(cache.Schemas).To(HaveLen(2))
					Expect(cache.Schemas[0].ID).To(Equal("castle"))
					Expect(cache.Schemas[1].ID).To(Equal("tower"))
				})

				It("Should show error - could not write file", func() {
					name, _ := ioutil.TempDir(os.TempDir(), "gohan_test")
					gohanClientCLI.opts.cachePath = name
					err = gohanClientCLI.setCachedSchemas()
					Expect(err).To(MatchError(ContainSubstring("is a directory")))
				})
			})
		})

		Describe("Commands", func() {
			var towerSchema *schema.Schema
			var castleSchema *schema.Schema
			const customCommandsTotal int = 2

			BeforeEach(func() {
				towerSchema, _ = schema.NewSchemaFromObj(getTowerSchema())
				castleSchema, _ = schema.NewSchemaFromObj(getCastleSchema())
			})

			It("Should get all commands", func() {
				gohanClientCLI.schemas = []*schema.Schema{
					towerSchema,
					castleSchema,
				}
				commands := gohanClientCLI.getCommands()
				Expect(commands).To(HaveLen(10 + customCommandsTotal))
				Expect(commands[0].Name).To(Equal("tower list"))
				Expect(commands[1].Name).To(Equal("tower show"))
				Expect(commands[2].Name).To(Equal("tower create"))
				Expect(commands[3].Name).To(Equal("tower set"))
				Expect(commands[4].Name).To(Equal("tower delete"))

				Expect(commands[5].Name).To(Equal("castle list"))
				Expect(commands[6].Name).To(Equal("castle show"))
				Expect(commands[7].Name).To(Equal("castle create"))
				Expect(commands[8].Name).To(Equal("castle set"))
				Expect(commands[9].Name).To(Equal("castle delete"))
				sort.Sort(ByName(commands[10 : 10+customCommandsTotal]))
				Expect(commands[10].Name).To(Equal("castle close_gates"))
				Expect(commands[11].Name).To(Equal("castle open_gates"))
			})

			Describe("Execute command", func() {
				BeforeEach(func() {
					gohanClientCLI.commands = []gohanCommand{}
				})

				It("Should execute command successfully", func() {
					command := gohanCommand{
						Name: "command",
						Action: func(args []string) (string, error) {
							return "ok", nil
						},
					}
					gohanClientCLI.commands = append(gohanClientCLI.commands, command)
					result, err := gohanClientCLI.ExecuteCommand("command", []string{})
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(Equal("ok"))
				})

				It("Should show sub commands - command not found", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/gohan/v0.1/schemas"),
							ghttp.RespondWithJSONEncoded(200, getSchemasResponse()),
						),
					)
					result, err := gohanClientCLI.ExecuteCommand("command", []string{})
					Expect(result).To(ContainSubstring("Command not found"))
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should show error - error executing command", func() {
					command := gohanCommand{
						Name: "command",
						Action: func(args []string) (string, error) {
							return "", fmt.Errorf("It is not ok :(")
						},
					}
					gohanClientCLI.commands = append(gohanClientCLI.commands, command)
					result, err := gohanClientCLI.ExecuteCommand("command", []string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("It is not ok :("))
				})
			})

			Describe("Parse arguments", func() {
				var (
					s            *schema.Schema
					args         []string
					argsExpected map[string]interface{}
				)

				BeforeEach(func() {
					s, _ = schema.NewSchemaFromObj(getChamberSchema())
					args = []string{}
					argsExpected = map[string]interface{}{}
				})

				It("Should parse arguments successfully", func() {
					args = append(args, "--name", "Chamber of secrets")
					argsExpected["name"] = "Chamber of secrets"
					argsMap, err := getArgsAsMap(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - integer casting", func() {
					args = append(args, "--windows", "6")
					argsExpected["windows"] = int64(6)
					argsMap, err := getArgsAsMap(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - boolean casting", func() {
					args = append(args, "--isPrincessIn", "true")
					argsExpected["isPrincessIn"] = true
					argsMap, err := getArgsAsMap(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - array casting", func() {
					args = append(args, "--chest", `["money", "hat", "cat"]`)
					argsExpected["chest"] = []interface{}{"money", "hat", "cat"}
					argsMap, err := getArgsAsMap(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - object casting", func() {
					args = append(args, "--weapon", `{"name": "Butter knife"}`)
					argsExpected["weapon"] = map[string]interface{}{
						"name": "Butter knife",
					}
					argsMap, err := getArgsAsMap(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(BeEquivalentTo(argsExpected))
				})

				It("Should parse arguments successfully - nil parsing", func() {
					args = append(args, "--name", "<null>")
					argsExpected["name"] = nil
					argsMap, err := getArgsAsMap(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(BeEquivalentTo(argsExpected))
				})

				It("Should parse arguments successfully - handle output format table", func() {
					args = append(args, "--output-format", "table")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.outputFormat).To(Equal("table"))
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - handle output format json", func() {
					args = append(args, "--output-format", "json")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.outputFormat).To(Equal("json"))
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - handle verbosity level 0", func() {
					args = append(args, "--verbosity", "0")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.logLevel).To(Equal(l.WARNING))
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - handle verbosity level 1", func() {
					args = append(args, "--verbosity", "1")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.logLevel).To(Equal(l.NOTICE))
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - handle verbosity level 2", func() {
					args = append(args, "--verbosity", "2")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.logLevel).To(Equal(l.INFO))
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - handle verbosity level 3", func() {
					args = append(args, "--verbosity", "3")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(gohanClientCLI.opts.logLevel).To(Equal(l.DEBUG))
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should parse arguments successfully - handle fields", func() {
					args = append(args, "--fields", "field1,field2")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).ToNot(HaveOccurred())
					Expect(argsMap).To(Equal(argsExpected))
				})

				It("Should show error - wrong arguments format", func() {
					args = append(args, "--windows")
					argsMap, err := getArgsAsMap(args, s)
					Expect(argsMap).To(BeNil())
					Expect(err).To(MatchError("Parameters should be in [--param-name value]... format"))
				})

				It("Should show error - error casting integer", func() {
					args = append(args, "--windows", "six")
					argsMap, err := getArgsAsMap(args, s)
					Expect(argsMap).To(BeNil())
					Expect(err).To(MatchError("Error parsing parameter 'windows': strconv.ParseInt: parsing \"six\": invalid syntax"))
				})

				It("Should show error - error casting bool", func() {
					args = append(args, "--isPrincessIn", "yes")
					argsMap, err := getArgsAsMap(args, s)
					Expect(argsMap).To(BeNil())
					Expect(err).To(MatchError("Error parsing parameter 'isPrincessIn': strconv.ParseBool: parsing \"yes\": invalid syntax"))
				})

				It("Should show error - error casting array", func() {
					args = append(args, "--chest", `["asdf",]`)
					argsMap, err := getArgsAsMap(args, s)
					Expect(argsMap).To(BeNil())
					Expect(err).To(MatchError("Error parsing parameter 'chest': invalid character ']' looking for beginning of value"))
				})

				It("Should show error - error casting object", func() {
					args = append(args, "--weapon", "asdf")
					argsMap, err := getArgsAsMap(args, s)
					Expect(argsMap).To(BeNil())
					Expect(err).To(MatchError("Error parsing parameter 'weapon': invalid character 'a' looking for beginning of value"))
				})

				It("Should show error - incorrect output format", func() {
					args = append(args, "--output-format", "xml")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).To(MatchError("Incorrect output format. Available formats: [table json]"))
					Expect(argsMap).To(BeNil())
				})

				It("Should show error - incorrect verbosity level", func() {
					args = append(args, "--verbosity", "42")
					argsMap, err := gohanClientCLI.handleArguments(args, s)
					Expect(err).To(MatchError("Incorrect verbosity level. Available level range 0 3"))
					Expect(argsMap).To(BeNil())
				})

				Describe("Relation arguments", func() {
					var argsInput map[string]interface{}

					BeforeEach(func() {
						argsInput = map[string]interface{}{}
						gohanClientCLI.schemas = getSchemas()
					})

					It("Should parse parent argument successfully", func() {
						server.AppendHandlers(ghttp.RespondWith(200, nil))
						argsInput["castle"] = "Malbork"
						argsOutput, err := gohanClientCLI.handleRelationArguments(towerSchema, argsInput)
						Expect(err).ToNot(HaveOccurred())
						argsExpected["castle_id"] = "Malbork"
						Expect(argsOutput).To(Equal(argsExpected))
					})

					It("Should show error - parent schema not found", func() {
						gohanClientCLI.schemas = nil
						argsInput["castle"] = "Malbork"
						argsOutput, err := gohanClientCLI.handleRelationArguments(towerSchema, argsInput)
						Expect(argsOutput).To(BeNil())
						Expect(err).To(MatchError("Schema with ID 'castle' not found"))
					})

					It("Should parse related resource argument successfully", func() {
						server.AppendHandlers(ghttp.RespondWith(200, nil))
						argsInput["sister"] = "MinasMorgul"
						argsOutput, err := gohanClientCLI.handleRelationArguments(towerSchema, argsInput)
						Expect(err).ToNot(HaveOccurred())
						argsExpected["sister_id"] = "MinasMorgul"
						Expect(argsOutput).To(Equal(argsExpected))
					})

					It("Should show error - related schema not found", func() {
						gohanClientCLI.schemas = nil
						argsInput["sister"] = "MinasMorgul"
						argsOutput, err := gohanClientCLI.handleRelationArguments(towerSchema, argsInput)
						Expect(argsOutput).To(BeNil())
						Expect(err).To(MatchError("Schema with ID 'tower' not found"))
					})
				})
			})

			Describe("Output Format", func() {
				Describe("Table output format", func() {
					BeforeEach(func() {
						manager := schema.GetManager()
						schemaPath := "../../tests/test_schema.json"
						Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
						netSchema, _ = manager.Schema("net")
						gohanClientCLI.opts.outputFormat = outputFormatTable
					})

					It("Should format single resource successfully", func() {

						rawResult := map[string]interface{}{
							"resource": map[string]interface{}{
								"cidr":  "cidr",
								"mac":   "mac",
								"id":    "test",
								"port":  "port",
								"regex": "regex",
							},
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal(
							`+----------+-------+
| PROPERTY | VALUE |
+----------+-------+
| CIDR     | cidr  |
| MAC      | mac   |
| UUID     | test  |
| port     | port  |
| regex    | regex |
+----------+-------+
`))
					})

					It("Should format single resource with filtered columns successfully", func() {
						rawResult := map[string]interface{}{
							"resource": map[string]interface{}{
								"cidr":  "cidr",
								"mac":   "mac",
								"id":    nil,
								"port":  nil,
								"regex": nil,
							},
						}
						gohanClientCLI.opts.fields = []string{"cidr", "mac"}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal(
							`+----------+-------+
| PROPERTY | VALUE |
+----------+-------+
| CIDR     | cidr  |
| MAC      | mac   |
+----------+-------+
`))
					})

					It("Should format multiple resources successfully", func() {
						rawResult := map[string]interface{}{
							"resources": []interface{}{
								map[string]interface{}{
									"cidr":  "cidr1",
									"mac":   "mac1",
									"id":    "test1",
									"port":  "port1",
									"regex": "regex1",
								},
								map[string]interface{}{
									"cidr":  "cidr2",
									"mac":   "mac2",
									"id":    "test2",
									"port":  "port2",
									"regex": "regex2",
								},
								map[string]interface{}{
									"cidr":  "cidr3",
									"mac":   "mac3",
									"id":    "test3",
									"port":  "port3",
									"regex": "regex3",
								},
							},
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal(
							`+-------+------+-------+-------+--------+
| CIDR  | MAC  | UUID  | PORT  | REGEX  |
+-------+------+-------+-------+--------+
| cidr1 | mac1 | test1 | port1 | regex1 |
| cidr2 | mac2 | test2 | port2 | regex2 |
| cidr3 | mac3 | test3 | port3 | regex3 |
+-------+------+-------+-------+--------+
`))
					})

					It("Should format multiple resources with missing values successfully", func() {
						rawResult := map[string]interface{}{
							"resources": []interface{}{
								map[string]interface{}{
									"cidr":  "cidr1",
									"mac":   "mac1",
									"id":    "test1",
									"port":  "port1",
									"regex": "regex1",
								},
								map[string]interface{}{
									"cidr":  "",
									"mac":   "mac2",
									"id":    "",
									"port":  "port2",
									"regex": "",
								},
								map[string]interface{}{
									"cidr":  "cidr3",
									"mac":   "",
									"id":    "test3",
									"port":  "",
									"regex": "regex3",
								},
							},
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal(
							`+-------+------+-------+-------+--------+
| CIDR  | MAC  | UUID  | PORT  | REGEX  |
+-------+------+-------+-------+--------+
| cidr1 | mac1 | test1 | port1 | regex1 |
|       | mac2 |       | port2 |        |
| cidr3 |      | test3 |       | regex3 |
+-------+------+-------+-------+--------+
`))
					})

					It("Should format multiple resources with filtered columns successfully", func() {
						rawResult := map[string]interface{}{
							"resources": []interface{}{
								map[string]interface{}{
									"cidr":  "cidr1",
									"mac":   "mac1",
									"id":    nil,
									"port":  nil,
									"regex": nil,
								},
								map[string]interface{}{
									"cidr":  "cidr2",
									"mac":   "mac2",
									"id":    nil,
									"port":  nil,
									"regex": nil,
								},
								map[string]interface{}{
									"cidr":  "cidr3",
									"mac":   "mac3",
									"id":    nil,
									"port":  nil,
									"regex": nil,
								},
							},
						}
						gohanClientCLI.opts.fields = []string{"cidr", "mac"}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal(
							`+-------+------+
| CIDR  | MAC  |
+-------+------+
| cidr1 | mac1 |
| cidr2 | mac2 |
| cidr3 | mac3 |
+-------+------+
`))
					})

					It("Should format empty output successfully", func() {
						result := gohanClientCLI.formatOutput(netSchema, nil)
						Expect(result).To(Equal(""))
					})

					It("Should format empty list successfully", func() {
						rawResult := map[string]interface{}{
							"resources": []interface{}{},
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal(""))
					})

					It("Should format wrong resource successfully", func() {
						rawResult := map[string]interface{}{
							"resource": "Wrong resource",
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal("Wrong resource"))
					})

					It("Should format string error successfully", func() {
						rawResult := map[string]interface{}{
							"error": "Simple string error",
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(Equal("Simple string error"))
					})

					It("Should format map error successfully", func() {
						rawResult := map[string]interface{}{
							"error": map[string]interface{}{
								"name":    "Error name",
								"message": "Error message",
							},
						}
						result := gohanClientCLI.formatOutput(netSchema, rawResult)
						Expect(result).To(ContainSubstring("name:Error name"))
						Expect(result).To(ContainSubstring("message:Error message"))
					})
				})
			})
			Describe("List command", func() {
				var listCommand gohanCommand

				BeforeEach(func() {
					listCommand = gohanClientCLI.getListCommand(towerSchema)
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/v2.0/towers"),
							ghttp.RespondWithJSONEncoded(200, getTowerListResponse()),
						),
					)
				})

				It("Should create 'list' command with proper name", func() {
					Expect(listCommand.Name).To(Equal("tower list"))
				})

				It("Should list resources successfully", func() {
					result, err := listCommand.Action([]string{})
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(MatchJSON(getTowerListJSONResponse()))
				})
				It("Should add 2 fields to gohanClient", func() {
					_, err := listCommand.Action([]string{"--fields", "id,name"})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(gohanClientCLI.opts.fields)).To(Equal(2))
				})

				It("Should show error - error parsing arguments", func() {
					result, err := listCommand.Action([]string{"--isMain", "yes"})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError(HavePrefix("Error parsing parameter")))
				})

				It("Should show error - unexpected response", func() {
					server.SetHandler(1, ghttp.RespondWith(500, nil))
					result, err := listCommand.Action([]string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Unexpected response: 500 Internal Server Error"))
				})
			})

			Describe("Get Command", func() {
				var getCommand gohanCommand

				BeforeEach(func() {
					getCommand = gohanClientCLI.getGetCommand(towerSchema)
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/v2.0/towers/"+icyTowerID),
							ghttp.RespondWithJSONEncoded(200, getIcyTower()),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/v2.0/towers/"+icyTowerID),
							ghttp.RespondWithJSONEncoded(200, getIcyTower()),
						),
					)
				})

				It("Should create 'get' command with proper name", func() {
					Expect(getCommand.Name).To(Equal("tower show"))
				})

				It("Should get resource successfully", func() {
					result, err := getCommand.Action([]string{icyTowerID})
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(getIcyTower()))
				})

				It("Should get resource by name successfully", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					handler := server.GetHandler(2)
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(200, getIcyTowerListResponse()))
					server.AppendHandlers(handler)
					result, err := getCommand.Action([]string{icyTowerName})
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(getIcyTower()))
				})

				It("Should show error - wrong number of arguments", func() {
					result, err := getCommand.Action([]string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Wrong number of arguments"))
				})

				It("Should show error - error parsing arguments", func() {
					result, err := getCommand.Action([]string{"--isMain", "yes", icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError(HavePrefix("Error parsing parameter")))
				})

				It("Should show error - resource not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					server.SetHandler(2, ghttp.RespondWith(404, nil))
					result, err := getCommand.Action([]string{"wrongId"})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})

				It("Should show error - unexpected response", func() {
					server.SetHandler(1, ghttp.RespondWith(200, getTowerListJSONResponse()))
					server.SetHandler(2, ghttp.RespondWith(500, nil))
					result, err := getCommand.Action([]string{icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Unexpected response: 500 Internal Server Error"))
				})
			})

			Describe("Post Command", func() {
				var postCommand gohanCommand

				BeforeEach(func() {
					postCommand = gohanClientCLI.getPostCommand(towerSchema)
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/v2.0/towers"),
							ghttp.RespondWithJSONEncoded(201, getIcyTower()),
						),
					)
				})

				It("Should create 'post' command with proper name", func() {
					Expect(postCommand.Name).To(Equal("tower create"))
				})

				It("Should create resource successfully", func() {
					args := []string{
						"--name", "Icy Tower",
					}
					result, err := postCommand.Action(args)
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(getIcyTower()))
				})

				It("Should handle bad request successfully", func() {
					response := map[string]interface{}{
						"bad": "request",
					}
					server.SetHandler(1, ghttp.RespondWithJSONEncoded(400, response))
					result, err := postCommand.Action([]string{})
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(response))
				})

				It("Should show error - error parsing arguments", func() {
					result, err := postCommand.Action([]string{"--isMain", "yes"})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError(HavePrefix("Error parsing parameter")))
				})

				It("Should show error - unexpected response", func() {
					server.SetHandler(1, ghttp.RespondWith(500, nil))
					result, err := postCommand.Action([]string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Unexpected response: 500 Internal Server Error"))
				})

				It("Should show error - parent schema not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					result, err := postCommand.Action([]string{"--castle", "Malbork"})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Schema with ID 'castle' not found"))
				})
			})

			Describe("Put Command", func() {
				var putCommand gohanCommand

				BeforeEach(func() {
					putCommand = gohanClientCLI.getPutCommand(towerSchema)
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/v2.0/towers/"+icyTowerID),
							ghttp.RespondWithJSONEncoded(200, getIcyTower()),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/v2.0/towers/"+icyTowerID),
							ghttp.RespondWithJSONEncoded(202, getIcyTower()),
						),
					)
				})

				It("Should create 'put' command with proper name", func() {
					Expect(putCommand.Name).To(Equal("tower set"))
				})

				It("Should update resource successfully", func() {
					args := []string{
						"--name", "Icy Tower",
						icyTowerID,
					}
					result, err := putCommand.Action(args)
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(getIcyTower()))
				})

				It("Should update resource by name successfully", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					handler := server.GetHandler(2)
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(200, getIcyTowerListResponse()))
					server.AppendHandlers(handler)
					args := []string{
						"--name", "Icy Tower",
						icyTowerName,
					}
					result, err := putCommand.Action(args)
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(getIcyTower()))
				})

				It("Should handle bad request successfully", func() {
					response := map[string]interface{}{
						"bad": "request",
					}
					server.SetHandler(1, ghttp.RespondWithJSONEncoded(200, getTowerListJSONResponse()))
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(400, response))
					result, err := putCommand.Action([]string{icyTowerID})
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(response))
				})

				It("Should show error - wrong number of arguments", func() {
					result, err := putCommand.Action([]string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Wrong number of arguments"))
				})

				It("Should show error - error parsing arguments", func() {
					result, err := putCommand.Action([]string{"--isMain", "yes", icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError(HavePrefix("Error parsing parameter")))
				})

				It("Should show error - resource not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					server.SetHandler(2, ghttp.RespondWith(404, nil))
					result, err := putCommand.Action([]string{"wrongId"})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})

				It("Should show error - unexpected response", func() {
					server.SetHandler(1, ghttp.RespondWithJSONEncoded(200, getTowerListJSONResponse()))
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(500, nil))
					result, err := putCommand.Action([]string{icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Unexpected response: 500 Internal Server Error"))
				})

				It("Should show error - parent schema not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					result, err := putCommand.Action([]string{"--castle", "Malbork", icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Schema with ID 'castle' not found"))
				})
			})

			Describe("Delete Command", func() {
				var deleteCommand gohanCommand

				BeforeEach(func() {
					deleteCommand = gohanClientCLI.getDeleteCommand(towerSchema)
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/v2.0/towers/"+icyTowerID),
							ghttp.RespondWithJSONEncoded(200, getIcyTower()),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("DELETE", "/v2.0/towers/"+icyTowerID),
							ghttp.RespondWith(204, nil),
						),
					)
				})

				It("Should create 'delete' command with proper name", func() {
					Expect(deleteCommand.Name).To(Equal("tower delete"))
				})

				It("Should delete resource successfully", func() {
					args := []string{
						icyTowerID,
					}
					result, err := deleteCommand.Action(args)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(Equal(""))
				})

				It("Should delete resource by name successfully", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					handler := server.GetHandler(2)
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(200, getIcyTowerListResponse()))
					server.AppendHandlers(handler)
					args := []string{
						icyTowerName,
					}
					result, err := deleteCommand.Action(args)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(Equal(""))
				})

				It("Should show error - wrong number of arguments", func() {
					result, err := deleteCommand.Action([]string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Wrong number of arguments"))
				})

				It("Should show error - resource not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					server.SetHandler(2, ghttp.RespondWith(404, nil))
					result, err := deleteCommand.Action([]string{"wrongId"})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})

				It("Should show error - error parsing arguments", func() {
					result, err := deleteCommand.Action([]string{"--isMain", "yes", icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError(HavePrefix("Error parsing parameter")))
				})

				It("Should show error - unexpected response", func() {
					server.SetHandler(1, ghttp.RespondWithJSONEncoded(200, getTowerListJSONResponse()))
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(500, nil))
					result, err := deleteCommand.Action([]string{icyTowerID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Unexpected response: 500 Internal Server Error"))
				})
			})

			Describe("Custom commands", func() {
				var customCommands []gohanCommand
				const openGatesInput string = `{ "gate_id": 42 }`

				BeforeEach(func() {
					customCommands = gohanClientCLI.getCustomCommands(castleSchema)
					sort.Sort(ByName(customCommands))
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest(
								"GET",
								"/v2.0/castles/"+castleID,
							),
							ghttp.RespondWithJSONEncoded(
								200,
								map[string]interface{}{
									"id":   castleID,
									"name": "Camelot",
								},
							),
						),
					)
				})

				It("Should create proper number of custom commands with proper names", func() {
					Expect(len(customCommands)).To(Equal(customCommandsTotal))
					Expect(customCommands[0].Name).To(Equal("castle close_gates"))
					Expect(customCommands[1].Name).To(Equal("castle open_gates"))
				})

				It("Should 'open gates' successfully", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest(
								"GET",
								"/v2.0/castles/"+castleID+"/open_gates",
							),
							ghttp.RespondWithJSONEncoded(200, openGates()),
						),
					)
					Expect(customCommands[1].Name).To(Equal("castle open_gates"))
					result, err := customCommands[1].Action([]string{
						`{ "gate_id": 42 }`,
						castleID,
					})
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(openGates()))
				})

				It("Should 'close gates' successfully", func() {
					server.SetHandler(1,
						ghttp.CombineHandlers(
							ghttp.VerifyRequest(
								"GET",
								"/v2.0/castles/close_all_gates",
							),
							ghttp.RespondWithJSONEncoded(200, closeGates()),
						),
					)
					Expect(customCommands[0].Name).To(Equal("castle close_gates"))
					result, err := customCommands[0].Action([]string{})
					Expect(err).ToNot(HaveOccurred())
					var resultJSON interface{}
					json.Unmarshal([]byte(result), &resultJSON)
					Expect(resultJSON).To(Equal(closeGates()))
				})

				It("Should show error - wrong number of arguments", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest(
								"GET",
								"/v2.0/castles/"+castleID+"/open_gates",
							),
							ghttp.RespondWithJSONEncoded(200, openGates()),
						),
					)
					result, err := customCommands[1].Action([]string{castleID})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Wrong number of arguments"))
				})

				It("Should show error - error parsing arguments", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest(
								"GET",
								"/v2.0/castles/"+castleID+"/open_gates",
							),
							ghttp.RespondWithJSONEncoded(200, closeGates()),
						),
					)
					result, err := customCommands[1].Action([]string{
						"--opt", "yes",
						openGatesInput,
						castleID,
					})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError(HavePrefix("Error parsing parameter")))
				})

				It("Should show error - resource not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					server.AppendHandlers(ghttp.RespondWith(404, nil))
					result, err := customCommands[1].Action([]string{
						openGatesInput,
						"wrongID",
					})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})

				It("Should show error - unexpected response", func() {
					server.SetHandler(1, ghttp.RespondWith(500, nil))
					result, err := customCommands[0].Action([]string{})
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Unexpected response: 500 Internal Server Error"))
				})
				// TODO more?
			})
		})

		Describe("Name -> ID mapping", func() {
			var towerSchema *schema.Schema

			BeforeEach(func() {
				towerSchema, _ = schema.NewSchemaFromObj(getTowerSchema())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2.0/towers/"+icyTowerID),
						ghttp.RespondWithJSONEncoded(200, getIcyTower()),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2.0/towers", "name=Icy+Tower"),
						ghttp.RespondWithJSONEncoded(200, getIcyTowerListResponse()),
					),
				)
			})

			It("Should map successfully - ID given", func() {
				result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(icyTowerID))
			})

			It("Should map successfully - name given", func() {
				server.SetHandler(1,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2.0/towers/Icy+Tower"),
						ghttp.RespondWith(404, nil),
					),
				)
				result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerName)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(icyTowerID))
			})

			It("Should show error - query by name unsuccessful", func() {
				server.SetHandler(1, ghttp.RespondWith(404, nil))
				server.SetHandler(2, ghttp.RespondWith(404, nil))
				result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerName)
				Expect(result).To(Equal(""))
				Expect(err).To(MatchError("Resource not found"))
			})

			It("Should show error - multiple resource with given name", func() {
				server.SetHandler(1, ghttp.RespondWith(404, nil))
				server.SetHandler(2,
					ghttp.RespondWithJSONEncoded(200, map[string]interface{}{
						"towers": []interface{}{
							getIcyTower(),
							getIcyTower(),
						},
					}),
				)
				result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerName)
				Expect(result).To(Equal(""))
				Expect(err).To(MatchError("Multiple towers with name 'Icy Tower' found"))
			})

			Describe("Resource not found", func() {
				BeforeEach(func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
				})

				It("Should show error - response is not a map", func() {
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(200, ""))
					result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerName)
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})

				It("Should show error - response does not contain 'plural' key", func() {
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(200, map[string]interface{}{
						"NotPlural": "value",
					}))
					result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerName)
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})

				It("Should show error - response contains empty list", func() {
					server.SetHandler(2, ghttp.RespondWithJSONEncoded(200, map[string]interface{}{
						"towers": []interface{}{},
					}))
					result, err := gohanClientCLI.getResourceID(towerSchema, icyTowerName)
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Resource not found"))
				})
			})

			Describe("Schema by ID", func() {
				It("Should get resource ID successfully", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					gohanClientCLI.schemas = getSchemas()
					result, err := gohanClientCLI.getResourceIDForSchemaID("tower", icyTowerName)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(Equal(icyTowerID))
				})

				It("Should show error - schema not found", func() {
					server.SetHandler(1, ghttp.RespondWith(404, nil))
					result, err := gohanClientCLI.getResourceIDForSchemaID("tower", icyTowerName)
					Expect(result).To(Equal(""))
					Expect(err).To(MatchError("Schema with ID 'tower' not found"))
				})
			})
		})
	})
})
