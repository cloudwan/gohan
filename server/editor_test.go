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

package server

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/cloudwan/gohan/schema"
)

func TestGetSchema(t *testing.T) {
	manager := schema.GetManager()
	defer schema.ClearManager()

	if err := manager.LoadSchemaFromFile("embed://etc/schema/gohan.json"); err != nil {
		t.Error(err)
	}
	if err := manager.LoadSchemaFromFile("../tests/test_schema_member.yaml"); err != nil {
		t.Error(err)
	}

	a := schema.NewAuthorizationBuilder().
		WithTenant(schema.Tenant{ID: "member", Name: "member"}).
		WithRoleIDs("Member").
		BuildScopedToTenant()
	s, ok := manager.Schema("member_resource")
	if !ok {
		t.Error("Could not find schema")
	}

	r := TrimmedResource(s, a)

	if !resourceMatches(r, "test-fixtures/editor_member.json") {
		t.Fatal("Unexpected resource")
	}
}

func resourceMatches(r *schema.Resource, path string) bool {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var expected map[string]interface{}
	if err = json.NewDecoder(f).Decode(&expected); err != nil {
		panic(err)
	}

	var actual map[string]interface{}
	tmp, err := r.JSONString()
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal([]byte(tmp), &actual); err != nil {
		panic(err)
	}

	return reflect.DeepEqual(expected, actual)
}
