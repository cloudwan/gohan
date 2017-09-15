// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package goext

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

// ResourceState represents the state of a resource
type ResourceState struct {
	ConfigVersion int64
	StateVersion  int64
	Error         string
	State         string
	Monitoring    string
}

// ListOptions specifies additional list related options.
type ListOptions struct {
	// Details specifies if all the underlying structures should be
	// returned.
	Details bool
	// Fields limits list output to only showing selected fields.
	Fields []string
}

// ITransaction is common interface for handling transaction
type ITransaction interface {
	Create(schema ISchema, resource map[string]interface{}) error
	Update(schema ISchema, resource map[string]interface{}) error
	StateUpdate(schema ISchema, resource map[string]interface{}, state *ResourceState) error
	Delete(schema ISchema, resourceID interface{}) error
	Fetch(schema ISchema, filter Filter) (map[string]interface{}, error)
	LockFetch(schema ISchema, filter Filter, lockPolicy LockPolicy) (map[string]interface{}, error)
	StateFetch(schema ISchema, filter Filter) (ResourceState, error)
	List(schema ISchema, filter Filter, listOptions *ListOptions, paginator *Paginator) ([]map[string]interface{}, uint64, error)
	LockList(schema ISchema, filter Filter, listOptions *ListOptions, paginator *Paginator, lockPolicy LockPolicy) ([]map[string]interface{}, uint64, error)
	RawTransaction() interface{} // *sqlx.Tx
	Query(schema ISchema, query string, args []interface{}) (list []map[string]interface{}, err error)
	Commit() error
	Exec(query string, args ...interface{}) error
	Close() error
	Closed() bool
	GetIsolationLevel() Type
}
