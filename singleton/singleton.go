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

package singleton

import (
	"sync"
)

// Scope specifies bean scope, default scope is ScopeSingleton.
type Scope uint8

const (
	// ScopeSingleton is default Scope, with this setting there is a single
	// bean instance shared by all goroutines.
	ScopeSingleton Scope = iota
	// ScopeGLSSingleton is goroutine local storage singleton, with this setting
	// bean instance is shared by a goroutine tree, child goroutines
	// share bean instance with parent.
	ScopeGLSSingleton
)

var (
	c  cache
	mu sync.Mutex
)

func init() {
	SetScope(ScopeSingleton)
}

// SetScope changes singletons scope, see scopes documentation.
func SetScope(s Scope) {
	switch s {
	case ScopeSingleton:
		c = make(mapCache)
	case ScopeGLSSingleton:
		c = glsCache{}
	}
}

// Get returns singleton for a given key, value does not exist if will be
// created by factory.
func Get(key string, factory func() interface{}) interface{} {
	mu.Lock()
	defer mu.Unlock()

	v := c.Get(key)
	if v == nil {
		v = factory()
		c.Set(key, v)
	}
	return v
}

// Clear removes singleton for a given key.
func Clear(key string) {
	mu.Lock()
	defer mu.Unlock()
	c.Set(key, nil)
}
