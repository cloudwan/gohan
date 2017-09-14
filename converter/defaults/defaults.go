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

package defaults

// PlainDefaults is an interface for default values
type PlainDefaults interface {
	Write() string
}

// CreatePlainDefaults is a PlainDefaults factory
func CreatePlainDefaults(value interface{}) PlainDefaults {
	switch value.(type) {
	case string:
		return &StringDefault{value: value.(string)}
	case int:
		return &IntDefault{value: value.(int)}
	case bool:
		return &BoolDefault{value: value.(bool)}
	case float64:
		return &FloatDefault{value: value.(float64)}
	}
	return nil
}
