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

import "strings"

//Split splits value using separator
func Split(value, sep string) (interface{}, error) {
	stringResult := strings.Split(value, sep)
	result := make([]interface{}, len(stringResult))
	for i, val := range stringResult {
		result[i] = val
	}
	return result, nil
}

//Join joins list using separator
func Join(value []interface{}, sep string) (interface{}, error) {
	input := make([]string, len(value))
	for i, val := range value {
		input[i] = val.(string)
	}
	result := strings.Join(input, sep)
	return result, nil
}
