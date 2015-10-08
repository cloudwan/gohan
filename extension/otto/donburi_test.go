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
	"os"
	"testing"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	. "github.com/onsi/gomega"
)

var (
	dataStore db.DB
)

const (
	dbtype = "yaml"
	conn   = "./test_extension.yaml"
)

func beforeEach() {
	os.Remove(conn)
	dataStore, _ = db.ConnectDB(dbtype, conn)
}

func afterEach() {
	os.Remove(conn)
}

func TestDonburiFlows(t *testing.T) {
	RegisterTestingT(t)
	beforeEach()
	defer afterEach()
	donburi, err := schema.NewExtension(map[string]interface{}{
		"id":   "donburi",
		"url":  "file://../../etc/extensions/donburi.js",
		"path": ".*",
	})
	if err != nil {
		t.Error(err)
	}

	extension, err := schema.NewExtension(map[string]interface{}{
		"id": "test_donburi",
		"code": `tasks:
  - eval: "1 + 1"
    register: result
  - eval: "true"
    register: when_is_working
    when: "result == 2"
  - block:
    - vars:
        list2 : [4, 5, 6]
    - eval: "result += item"
      with_items:
       - 1
       - 2
       - 3
    when: when_is_working
  - vars:
      message: "hello"
  - vars:
      retry_count: 0
      rescue_executed: false
      always_executed: false
      template_test: "message: {{.message}}"
  - eval: "result += item"
    with_items: "list2"
  - eval: "context[item.key] = item.value"
    with_dict:
      alice: 18
      bob: 21
  - block:
    - sleep: 10
    - eval: retry_count += 1
    - eval: throw 'error'
    rescue:
    - eval: "rescue_executed = true"
    always:
    - eval: "always_executed = true"
    retry: 3
`,
		"path":      ".*",
		"code_type": "donburi",
	})
	if err != nil {
		t.Error(err)
	}

	context := map[string]interface{}{}

	extensions := []*schema.Extension{donburi, extension}
	env := NewEnvironment(dataStore, &middleware.FakeIdentity{})
	err = env.LoadExtensionsForPath(extensions, "test_path")
	if err != nil {
		t.Error(err)
	}
	err = env.HandleEvent("post_create", context)
	if err != nil {
		t.Error(err)
	}
	Expect(context["result"]).To(Equal(float64(23)))
	Expect(context["when_is_working"]).To(Equal(true))
	Expect(context["rescue_executed"]).To(Equal(true))
	Expect(context["always_executed"]).To(Equal(true))
	Expect(context["template_test"]).To(Equal("message: hello"))
	Expect(context["alice"]).To(Equal(int(18)))
	Expect(context["retry_count"]).To(Equal(float64(3)))
}

func TestDonburiExec(t *testing.T) {
	RegisterTestingT(t)
	beforeEach()
	defer afterEach()
	donburi, err := schema.NewExtension(map[string]interface{}{
		"id":   "donburi",
		"url":  "file://../../etc/extensions/donburi.js",
		"path": ".*",
	})
	if err != nil {
		t.Error(err)
	}

	extension, err := schema.NewExtension(map[string]interface{}{
		"id": "test_donburi",
		"code": `tasks:
    - command:
       name: echo
       args:
         - test
      register: result
    - command:
       name: no_command
       args: []
      register: result2
`,
		"path":      ".*",
		"code_type": "donburi",
	})
	if err != nil {
		t.Error(err)
	}

	context := map[string]interface{}{}

	extensions := []*schema.Extension{donburi, extension}
	env := NewEnvironment(dataStore, &middleware.FakeIdentity{})
	err = env.LoadExtensionsForPath(extensions, "test_path")
	if err != nil {
		t.Error(err)
	}
	err = env.HandleEvent("post_create", context)
	if err != nil {
		t.Error(err)
	}
	output := context["result"].(map[string]string)
	Expect(output["status"]).To(Equal("success"))
	Expect(output["output"]).To(Equal("test\n"))
	output2 := context["result2"].(map[string]string)
	Expect(output2["status"]).To(Equal("error"))
}

func TestDonburiInjectionAttack(t *testing.T) {
	RegisterTestingT(t)
	beforeEach()
	defer afterEach()
	donburi, err := schema.NewExtension(map[string]interface{}{
		"id":   "donburi",
		"url":  "file://../../etc/extensions/donburi.js",
		"path": ".*",
	})
	if err != nil {
		t.Error(err)
	}

	extension, err := schema.NewExtension(map[string]interface{}{
		"id": "test_donburi",
		"code": `tasks:
    - vars:
        executed: true
      when: code
    - eval: "{{.code}}"
    - block:
        - eval: "throw error"
          rescue:
            - eval "{{.code}}"
          always:
            - eval "{{.code}}"
          when: code
`,
		"path":      ".*",
		"code_type": "donburi",
	})
	if err != nil {
		t.Error(err)
	}

	context := map[string]interface{}{
		"attacked": false,
		"code":     "context['attacked'] = true",
	}

	extensions := []*schema.Extension{donburi, extension}
	env := NewEnvironment(dataStore, &middleware.FakeIdentity{})
	err = env.LoadExtensionsForPath(extensions, "test_path")
	if err != nil {
		t.Error(err)
	}
	err = env.HandleEvent("post_create", context)
	if err != nil {
		t.Error(err)
	}
	Expect(context["attacked"]).To(Equal(false))
}

func TestDonburiResources(t *testing.T) {
	RegisterTestingT(t)
	beforeEach()
	defer afterEach()
	donburi, err := schema.NewExtension(map[string]interface{}{
		"id":   "donburi",
		"url":  "file://../../etc/extensions/donburi.js",
		"path": ".*",
	})
	if err != nil {
		t.Error(err)
	}

	extension, err := schema.NewExtension(map[string]interface{}{
		"id": "test_donburi",
		"code": `tasks:
    - vars:
        failed: false
    - resources:
      - vars:
          dependent: true
      - eval: "if(dependent){failed = true}"
        when: event_type == "pre_delete"
      - eval: "if(!dependent){failed = true}"
        when: event_type == "post_create"
      - vars:
          dependent: false
`,
		"path":      ".*",
		"code_type": "donburi",
	})
	if err != nil {
		t.Error(err)
	}

	context := map[string]interface{}{}

	extensions := []*schema.Extension{donburi, extension}
	env := NewEnvironment(dataStore, &middleware.FakeIdentity{})
	err = env.LoadExtensionsForPath(extensions, "test_path")
	if err != nil {
		t.Error(err)
	}
	err = env.HandleEvent("post_create", context)
	if err != nil {
		t.Error(err)
	}
	Expect(context["failed"]).To(Equal(false))
	err = env.HandleEvent("pre_delete", context)
	if err != nil {
		t.Error(err)
	}
	Expect(context["failed"]).To(Equal(false))
}

func BenchmarkDonburi(b *testing.B) {
	beforeEach()
	defer afterEach()
	donburi, _ := schema.NewExtension(map[string]interface{}{
		"id":   "donburi",
		"url":  "file://../../etc/extensions/donburi.js",
		"path": ".*",
	})
	extension, _ := schema.NewExtension(map[string]interface{}{
		"id": "test_donburi",
		"code": `tasks:
  - vars:
      message: "hello"
  - vars:
      template_test: "message: {{.message}}"
`,
		"path":      ".*",
		"code_type": "donburi",
	})

	context := map[string]interface{}{}

	extensions := []*schema.Extension{donburi, extension}
	env := NewEnvironment(dataStore, &middleware.FakeIdentity{})
	err := env.LoadExtensionsForPath(extensions, "test_path")
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env.HandleEvent("post_create", context)
	}
}
