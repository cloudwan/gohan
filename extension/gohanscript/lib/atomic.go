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

package lib

import (
	"github.com/cloudwan/gohan/util"
	"github.com/streamrail/concurrent-map"
)

//MakeMap makes thread safe map
func MakeMap() cmap.ConcurrentMap {
	return cmap.New()
}

//MapSet set value to map
func MapSet(m cmap.ConcurrentMap, key string, value interface{}) {
	m.Set(key, value)
}

//MapGet set value to map
func MapGet(m cmap.ConcurrentMap, key string) interface{} {
	val, _ := m.Get(key)
	return val
}

//MapHas key or not
func MapHas(m cmap.ConcurrentMap, key string) bool {
	return m.Has(key)
}

//MapRemove removes key from map
func MapRemove(m cmap.ConcurrentMap, key string) {
	m.Remove(key)
}

//MakeCounter makes thread safe counter
func MakeCounter(value int) *util.Counter {
	return util.NewCounter(int64(value))
}

//CounterAdd makes thread safe counter
func CounterAdd(counter *util.Counter, value int) {
	counter.Add(int64(value))
}

//CounterValue makes thread safe counter
func CounterValue(counter *util.Counter) int {
	return int(counter.Value())
}
