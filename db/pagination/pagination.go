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
	"math"
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

type OptionPaginator func(*Paginator) error

//NewPaginator create Paginator
func NewPaginator(options ...OptionPaginator) (*Paginator, error) {
	pg := &Paginator{
		Key:    "",
		Order:  "",
		Limit:  math.MaxUint64,
		Offset: 0,
	}

	for _, op := range options {
		err := op(pg)
		if err != nil {
			return nil, err
		}
	}
	return pg, nil
}

func OptionKey(s *schema.Schema, key string) OptionPaginator {
	return func(pg *Paginator) error {
		pg.Key = key

		if s != nil && pg.Key != "" {
			found := false
			for _, p := range s.Properties {
				if p.ID == pg.Key {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Schema %s has no property %s which can used as sorting key", s.ID, pg.Key)
			}
		}

		return nil
	}
}

func OptionOrder(order string) OptionPaginator {
	return func(pg *Paginator) error {
		pg.Order = order

		if pg.Order != "" && pg.Order != ASC && pg.Order != DESC {
			return fmt.Errorf("Unknown sort order %s", pg.Order)
		}

		return nil
	}
}

func OptionLimit(limit uint64) OptionPaginator {
	return func(pg *Paginator) error {
		pg.Limit = limit
		return nil
	}
}

func OptionOffset(offset uint64) OptionPaginator {
	return func(pg *Paginator) error {
		pg.Offset = offset
		return nil
	}
}

//FromURLQuery create Paginator from Query params
func FromURLQuery(s *schema.Schema, values url.Values) (pg *Paginator, err error) {
	var sortKey string
	var sortOrder string
	var limit uint64
	var offset uint64

	if sortKey = values.Get("sort_key"); sortKey == "" {
		sortKey = defaultSortKey
	}
	if sortOrder = values.Get("sort_order"); sortOrder == "" {
		sortOrder = defaultSortOrder
	}

	if l := values.Get("limit"); l != "" {
		limit, err = strconv.ParseUint(l, 10, 64)
		if err != nil {
			return
		}
		if limit < 0 {
			return nil, fmt.Errorf("Request contains invalid limit value %d", limit)
		}
	} else {
		limit = math.MaxUint64
	}

	if o := values.Get("offset"); o != "" {
		offset, err = strconv.ParseUint(o, 10, 64)
		if err != nil {
			return
		}
	}

	return NewPaginator(OptionKey(s, sortKey), OptionOrder(sortOrder), OptionLimit(limit), OptionOffset(offset))
}
