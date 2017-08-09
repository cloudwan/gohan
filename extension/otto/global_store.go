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

package otto

import "sync"

// GlobalStore for values living longer that a single extension environment
type GlobalStore struct {
	objects map[string]map[string]interface{}
	mtx     sync.Mutex
}

// NewGlobalStore ...
func NewGlobalStore() *GlobalStore {
	return &GlobalStore{
		objects: map[string]map[string]interface{}{},
	}
}

// Get a value registered for name,
// or, if there is none, register an empty object.
func (store *GlobalStore) Get(name string) map[string]interface{} {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	if result, ok := store.objects[name]; ok {
		return result
	}

	store.objects[name] = map[string]interface{}{}
	return store.objects[name]
}
