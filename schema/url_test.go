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

package schema

import (
	"net/url"
	"testing"
)

func TestIsURLRelative(t *testing.T) {
	t.Parallel()

	data := []struct {
		Rawurl   string
		Relative bool
	}{
		{"foobar", true},
		{"./foobar", true},
		{"/foobar", false},

		{"file://foobar", true},
		{"file://./foobar", true},
		{"file://../foobar", true},
		{"file://../../foobar", true},
		{"file:///foobar", false},

		{"http://foo.bar", false},
	}

	for _, d := range data {
		u, err := url.Parse(d.Rawurl)
		if err != nil {
			panic("Wrong URL")
		}
		if isURLRelative(u) != d.Relative {
			t.Error("Mismatch on", d)
		}
	}
}

func TestFixRelativeURL(t *testing.T) {
	t.Parallel()

	const dir = "/home/test/me"

	data := []struct {
		Rawurl   string
		Expected string
	}{
		{"./foobar", "file:///home/test/me/foobar"},
		{"/foobar", "file:///foobar"},
		{"file://foobar", "file:///home/test/me/foobar"},
		{"file://./foobar", "file:///home/test/me/foobar"},
		{"file://../foobar", "file:///home/test/foobar"},
		{"file://../../foobar", "file:///home/foobar"},
		{"file:///home/foobar", "file:///home/foobar"},
	}

	for _, d := range data {
		actual, err := fixRelativeURL(d.Rawurl, dir)
		if err != nil {
			t.Fatal(err)
		}
		if actual != d.Expected {
			t.Error("Mismatch on", d, actual)
		}
	}
}
