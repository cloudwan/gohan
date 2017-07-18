// Copyright (C) 2016  Juniper Networks, Inc.
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

package inproc

// Handler is type for go based callback
type Handler func(event string, context map[string]interface{}) error

var handlers = map[string]Handler{}

// RegisterHandler register go call back
func RegisterHandler(name string, handler Handler) {
	handlers[name] = handler
}

// GetCallback returns registered go callback
func GetCallback(name string) Handler {
	callback, ok := handlers[name]
	if !ok {
		return nil
	}
	return callback
}
