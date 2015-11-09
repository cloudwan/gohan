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

package transaction

import (
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx"
)

//ResourceState represents the state of a resource
type ResourceState struct {
	ConfigVersion int64
	StateVersion  int64
	Error         string
	State         string
	Monitoring    string
}

//Transaction is common interface for handing transaction
type Transaction interface {
	Create(*schema.Resource) error
	Update(*schema.Resource) error
	StateUpdate(*schema.Resource, *ResourceState) error
	Delete(*schema.Schema, interface{}) error
	Fetch(*schema.Schema, interface{}, []string) (*schema.Resource, error)
	StateFetch(*schema.Schema, interface{}, []string) (ResourceState, error)
	List(*schema.Schema, map[string]interface{}, *pagination.Paginator) ([]*schema.Resource, uint64, error)
	RawTransaction() *sqlx.Tx
	Query(*schema.Schema, string, []interface{}) (list []*schema.Resource, err error)
	Commit() error
	Close() error
	Closed() bool
}
