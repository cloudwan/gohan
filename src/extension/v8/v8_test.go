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

// +build v8
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

package v8

import (
	_ "github.com/onsi/ginkgo"
	_ "github.com/onsi/gomega"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

type ExampleObject struct {
	count float64
}

func (object *ExampleObject) Method(context map[string]interface{}, count float64) float64 {
	log.Printf("called %s", count)
	object.count += count
	log.Println(object.count)
	return object.count
}

func (object *ExampleObject) Method2(context map[string]interface{}) {
	log.Printf("called %s", context["fp"])
}

func TestV8(t *testing.T) {
	env := NewEnvironment()

	example := &ExampleObject{count: 0}
	env.rpc.RegistObject("example", example)

	testJSCode := `
var example = {
  "count": 0,
  "cast": function(contextID, num){
    this.count += num;
  },
  "call": function(contextID, num){
    this.count += num;
    v8rpc.cast("example", "Method", [contextID, 1]);
    v8rpc.call("example", "Method", [contextID, 2], function(reply){
      example.count += reply;
    });
    return this.count;
  },
};
v8rpc.register_object("example", example);


	`
	err := env.Load("test_js_code", testJSCode)
	if err != nil {
		log.Fatal("Failed to load %s", err)
	}
	context := map[string]interface{}{}
	contextID := "context_id"
	env.rpc.contexts[contextID] = context
	defer delete(env.rpc.contexts, contextID)
	env.rpc.Cast("example", "cast", []interface{}{contextID, 4})
	env.rpc.Call("example", "call", []interface{}{contextID, 8}, func(reply interface{}) {
		log.Printf("%v", reply)
	})

	reply := env.rpc.BlockCall("example", "call", []interface{}{contextID, 16})
	if reply.(float64) != 37 {
		t.Log(reply)
		t.Fail()
	}
	if example.count != 6 {
		t.Fail()
	}
}

func Test_Object(t *testing.T) {
	env := NewEnvironment()

	example := &ExampleObject{count: 0}
	env.rpc.RegistObject("example", example)

	testJSCode := `
var example = {
  "call": function(contextID){
    v8rpc.cast("example", "Method2", [contextID]);
    return contextID
  },
};
v8rpc.register_object("example", example);
	`
	err := env.Load("test_js_code", testJSCode)
	if err != nil {
		log.Fatal("Failed to load %s", err)
	}
	fp, _ := os.Open("./test.txt")
	context := map[string]interface{}{}
	context["fp"] = fp
	contextID := "context_id"
	env.rpc.contexts[contextID] = context
	defer delete(env.rpc.contexts, contextID)
	env.rpc.Call("example", "call", []interface{}{contextID}, func(reply interface{}) {
		log.Printf("%v", reply)
	})
}
