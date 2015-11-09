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

package pagination

import (
	"net/url"
	"testing"

	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/gomega"
)

func TestNewPaginator(t *testing.T) {
	RegisterTestingT(t)
	_, err := NewPaginator(nil, "name", "desc", 100, 200)
	Expect(err).ToNot(HaveOccurred())
}

func TestNewPaginatorDefaults(t *testing.T) {
	RegisterTestingT(t)
	pg, err := NewPaginator(nil, "", "", 0, 0)
	Expect(err).ToNot(HaveOccurred())
	Expect(pg.Key).To(Equal(defaultSortKey))
	Expect(pg.Order).To(Equal(ASC))
}

func TestUnknownSortOrder(t *testing.T) {
	RegisterTestingT(t)
	pg, err := NewPaginator(nil, "", "bad", 0, 0)
	Expect(err).To(HaveOccurred())
	Expect(pg).To(BeNil())
}

func TestFromURLQuery(t *testing.T) {
	RegisterTestingT(t)
	values := url.Values{
		"limit":      []string{"123"},
		"offset":     []string{"456"},
		"sort_key":   []string{"asd"},
		"sort_order": []string{"asc"},
	}
	pg, err := FromURLQuery(nil, values)
	Expect(err).ToNot(HaveOccurred())
	expected := &Paginator{
		Key:    "asd",
		Order:  "asc",
		Limit:  123,
		Offset: 456,
	}
	Expect(pg).To(Equal(expected))
}

func TestFromURLQueryErrors(t *testing.T) {
	RegisterTestingT(t)
	s := schema.NewSchema("foo", "foos", "Foo", "", "foo")
	s.Properties = append(s.Properties, schema.NewProperty("prop", "", "", "string", "", "", "", "", false, true, map[string]interface{}{}, "default"))

	values := url.Values{
		"limit":      []string{"a123"},
		"offset":     []string{"456"},
		"sort_key":   []string{"foo"},
		"sort_order": []string{"asc"},
	}
	pg, err := FromURLQuery(s, values)
	Expect(err).To(HaveOccurred(), "Got %v", pg)

	values.Set("limit", "123")
	values.Set("offset", "-1")
	pg, err = FromURLQuery(s, values)
	Expect(err).To(HaveOccurred(), "Got %v", pg)

	values.Set("offset", "123")
	values.Set("sort_key", "bad_key")
	pg, err = FromURLQuery(s, values)
	Expect(err).To(HaveOccurred(), "Got %v", pg)
}
