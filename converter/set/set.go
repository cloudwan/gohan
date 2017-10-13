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

package set

import (
	"fmt"
	"sort"
)

// Set is type for a set of elements identified by their names
type Set map[string]Element

// New is a constructor for an empty set
func New() Set {
	return make(map[string]Element)
}

func (set Set) safeContains(element Element) bool {
	return set.Contains(element) && set[element.Name()] != element
}

// Empty checks if set is nil or empty
func (set Set) Empty() bool {
	return set == nil || len(set) == 0
}

// Size returns number of elements in a set
func (set Set) Size() int {
	if set == nil {
		return 0
	}
	return len(set)
}

// Any returns an arbitrary element of a set
func (set Set) Any() Element {
	for _, value := range set {
		return value
	}
	return nil
}

// Contains checks if set contains an element with given name
func (set Set) Contains(element Element) bool {
	if set == nil {
		return false
	}
	_, ok := set[element.Name()]
	return ok
}

// Delete deletes an element with a given name from a set
func (set Set) Delete(element Element) {
	if set != nil {
		delete(set, element.Name())
	}
}

// Insert inserts an element with a given name to a set
func (set Set) Insert(element Element) {
	if set != nil {
		set[element.Name()] = element
	}
}

// InsertAll inserts all elements from an other set to a given set
func (set Set) InsertAll(other Set) {
	if !other.Empty() {
		for _, value := range other {
			set.Insert(value)
		}
	}
}

// SafeInsert inserts element to a set
// If the same element was already in the set nothing is done
// If element with the same name already was in the set an error is returned
func (set Set) SafeInsert(element Element) error {
	if set.safeContains(element) {
		return fmt.Errorf(
			"the element with the name %s already in the set",
			element.Name(),
		)
	}
	set.Insert(element)
	return nil
}

// SafeInsertAll inserts all elements from an other set to a given set
// If at least one of the elements in the given set has the same name
// as of one the elements of the given set and is different from it
// an error is returned an the given set remains unchanged
func (set Set) SafeInsertAll(other Set) error {
	for _, value := range other {
		if set.safeContains(value) {
			return fmt.Errorf(
				"the element with the name %s already in the set",
				value.Name(),
			)
		}
	}
	set.InsertAll(other)
	return nil
}

// ToArray converts a set to an array sorted by elements names
func (set Set) ToArray() []Element {
	if set == nil {
		return nil
	}
	result := make([]Element, len(set))
	i := 0
	for _, value := range set {
		result[i] = value
		i++
	}
	sort.Sort(byName(result))
	return result
}
