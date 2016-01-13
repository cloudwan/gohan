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

//Append add element to the list and return new list.
func Append(list []interface{}, value interface{}) []interface{} {
	return append(list, value)
}

//Contains checks if a value is in a list.
func Contains(list []interface{}, value interface{}) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

//Size checks length of a list
func Size(list []interface{}) int {
	return len(list)
}

//Shift retrives first element and left
func Shift(list []interface{}) (interface{}, []interface{}) {
	return list[0], list[1:]
}

//Unshift pushes an item on the beginnings
func Unshift(list []interface{}, value interface{}) []interface{} {
	return append([]interface{}{value}, list...)
}

//Copy copies a list
func Copy(list []interface{}) []interface{} {
	return append([]interface{}(nil), list...)
}

//Delete deletes an item from list
func Delete(list []interface{}, index int) []interface{} {
	return append(list[:index], list[index+1:]...)
}

//First return the first element
func First(list []interface{}) interface{} {
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

//Last return the last element
func Last(list []interface{}) interface{} {
	if len(list) == 0 {
		return nil
	}
	return list[len(list)-1]
}
