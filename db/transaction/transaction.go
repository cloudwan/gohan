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
	"errors"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx"
)

// ErrResourceNotFound is error message for missing resource
var ErrResourceNotFound = errors.New("resource not found")

//Type represents transaction types
type Type string

const (
	//ReadUncommitted is transaction type for READ UNCOMMITTED
	//You don't need to use this for most case
	ReadUncommitted Type = "READ UNCOMMITTED"
	//ReadCommited is transaction type for READ COMMITTED
	//You don't need to use this for most case
	ReadCommited Type = "READ COMMITTED"
	//RepeatableRead is transaction type for REPEATABLE READ
	//This is default value for read request
	RepeatableRead Type = "REPEATABLE READ"
	//Serializable is transaction type for Serializable
	Serializable Type = "SERIALIZABLE"
)

// TxOptions represents transaction options
type TxOptions struct {
	IsolationLevel Type
}

//Filter represents db filter
type Filter map[string]interface{}

//ResourceState represents the state of a resource
type ResourceState struct {
	ConfigVersion int64
	StateVersion  int64
	Error         string
	State         string
	Monitoring    string
}

//ViewOptions specifies additional options.
type ViewOptions struct {
	// Details specifies if all the underlying structures should be
	// returned.
	Details bool
	// Fields limits list output to only showing selected fields.
	Fields []string
}

//Transaction is common interface for handling transaction
type Transaction interface {
	Create(*schema.Resource) error
	Update(*schema.Resource) error
	StateUpdate(*schema.Resource, *ResourceState) error
	Delete(*schema.Schema, interface{}) error
	Fetch(*schema.Schema, Filter, *ViewOptions) (*schema.Resource, error)
	LockFetch(*schema.Schema, Filter, schema.LockPolicy, *ViewOptions) (*schema.Resource, error)
	StateFetch(*schema.Schema, Filter) (ResourceState, error)
	List(*schema.Schema, Filter, *ViewOptions, *pagination.Paginator) ([]*schema.Resource, uint64, error)
	LockList(*schema.Schema, Filter, *ViewOptions, *pagination.Paginator, schema.LockPolicy) ([]*schema.Resource, uint64, error)
	RawTransaction() *sqlx.Tx
	Query(*schema.Schema, string, []interface{}) (list []*schema.Resource, err error)
	Commit() error
	Exec(query string, args ...interface{}) error
	Close() error
	Closed() bool
	GetIsolationLevel() Type
}

// GetIsolationLevel returns isolation level for an action
func GetIsolationLevel(s *schema.Schema, action string) Type {
	level, ok := s.IsolationLevel[action]
	if !ok {
		switch action {
		case "read":
			return RepeatableRead
		default:
			return Serializable
		}
	}
	levelStr := level.(string)
	return Type(levelStr)
}

//IDFilter create filter for specific ID
func IDFilter(ID interface{}) Filter {
	return Filter{"id": ID}
}
