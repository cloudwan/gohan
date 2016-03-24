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

package gohanscript

import (
	"testing"
)

func TestParseDict(t *testing.T) {
	expected := map[string]interface{}{
		"msg": "debug",
		"var": "score.math",
	}
	l := newLexer("msg=\"debug\" var = score.math")
	actual := l.parseDict()
	if l.err != nil {
		t.Error(l.err)
	}
	for key, expected := range expected {
		if actual[key] != expected {
			t.Errorf("error key:%s expected:%v actual:%v", key, expected, actual[key])
		}
	}
	errorKeys := []string{
		"name",
		"\"name",
	}
	for _, testKey := range errorKeys {
		l := newLexer(testKey)
		value := l.parseDict()
		if l.err == nil {
			t.Errorf("error expected for key:%s", testKey)
		}
		if value != nil {
			t.Errorf("value expected nil for key:%s", testKey)
		}
	}
}
