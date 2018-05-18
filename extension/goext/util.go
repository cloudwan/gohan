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

type IUtil interface {
	// NewUUID create a new unique ID
	NewUUID() string
	// GetTransaction returns transaction from given context
	GetTransaction(context Context) (ITransaction, bool)

	// ResourceFromMapForType converts mapped representation to structure representation of the resource for given type
	ResourceFromMapForType(context map[string]interface{}, rawResource interface{}) (Resource, error)

	// ResourceToMap converts structure representation of the resource to mapped representation
	ResourceToMap(resource interface{}) map[string]interface{}
}
