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

package gohanscript_test

import (
	"time"

	"github.com/cloudwan/gohan/extension/gohanscript"
	//Import gohan script lib
	"github.com/cloudwan/gohan/extension"
	_ "github.com/cloudwan/gohan/extension/gohanscript/autogen"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Gohanscript extension manager", func() {
	var (
		timeLimit  time.Duration
		timeLimits []*schema.PathEventTimeLimit
	)

	BeforeEach(func() {
		timeLimit = time.Duration(10) * time.Second
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
		Context("When extension uses code property", func() {
			It("should run the extension code", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "test_extension",
					"code": `tasks:
                             - vars:
                                 person: John
                     `,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := gohanscript.NewEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["person"]).ToNot(BeNil())
			})
		})
		Context("When extension URL uses file:// protocol", func() {
			It("should read the file and run the extension", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id":   "test_extension",
					"url":  "file://./examples/variable.yaml",
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())

				extensions := []*schema.Extension{extension}
				env := gohanscript.NewEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["person"]).ToNot(BeNil())
			})
		})

		Context("When extension URL uses http:// protocol", func() {
			It("should download and run the extension", func() {
				server := ghttp.NewServer()
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/extension.yaml"),
					ghttp.RespondWith(200, `tasks:
                         - vars:
                             person: John
                         `),
				))

				extension, err := schema.NewExtension(map[string]interface{}{
					"id":   "test_extension",
					"url":  server.URL() + "/extension.yaml",
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := gohanscript.NewEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, "test_path")).To(Succeed())

				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).To(Succeed())
				Expect(context["person"]).ToNot(BeNil())
				server.Close()
			})
		})
	})

	var _ = Describe("Timeout", func() {
		Context("stops if execution time exceeds timeLimit", func() {
			It("Should work", func() {
				extension, err := schema.NewExtension(map[string]interface{}{
					"id": "infinite_loop",
					"code": `tasks:
                             - minigo: |
                                 for  {

                                 }
                             `,
					"path": ".*",
				})
				Expect(err).ToNot(HaveOccurred())
				extensions := []*schema.Extension{extension}
				env := gohanscript.NewEnvironment()
				Expect(env.LoadExtensionsForPath(extensions, time.Duration(100), timeLimits, "test_path")).To(Succeed())
				context := map[string]interface{}{
					"id": "test",
				}
				Expect(env.HandleEvent("test_event", context)).ToNot(Succeed())
			})
		})
	})
})
