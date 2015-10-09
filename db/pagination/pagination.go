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
	"fmt"
	"net/url"
	"strconv"

	"github.com/cloudwan/gohan/schema"
)

const (
	//ASC ascending order
	ASC = "asc"
	//DESC descending order
	DESC = "desc"

	defaultSortKey   = "id"
	defaultSortOrder = ASC
)

//Paginator stores pagination data
type Paginator struct {
	Key    string
	Order  string
	Limit  uint64
	Offset uint64
}

//NewPaginator create Paginator
func NewPaginator(s *schema.Schema, key, order string, limit, offset uint64) (*Paginator, error) {
	if key == "" {
		key = defaultSortKey
	}
	if order == "" {
		order = defaultSortOrder
	}
	if order != ASC && order != DESC {
		return nil, fmt.Errorf("Unknown sort order %s", order)
	}
	if s != nil {
		found := false
		for _, p := range s.Properties {
			if p.ID == key {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Schema %s has no property %s which can used as sorting key", s.ID, key)
		}
	}
	return &Paginator{
		Key:    key,
		Order:  order,
		Limit:  limit,
		Offset: offset,
	}, nil
}

//FromURLQuery create Paginator from Query params
func FromURLQuery(s *schema.Schema, values url.Values) (pg *Paginator, err error) {
	sortKey := values.Get("sort_key")
	sortOrder := values.Get("sort_order")
	var limit uint64
	var offset uint64

	if l := values.Get("limit"); l != "" {
		limit, err = strconv.ParseUint(l, 10, 64)
	}
	if err != nil {
		return
	}

	if o := values.Get("offset"); o != "" {
		offset, err = strconv.ParseUint(o, 10, 64)
	}
	if err != nil {
		return
	}
	return NewPaginator(s, sortKey, sortOrder, limit, offset)
}
