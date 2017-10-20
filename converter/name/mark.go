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

package name

// Mark is a type used to change names
type Mark interface {
	// Change should change string
	// args:
	//   1. *string - string; a string to be changed
	// return:
	//   true iff. string was changed
	Change(*string) bool

	// Update should update mark based on other mark
	// args:
	//   1. Mark - mark; a mark on which update is based
	Update(Mark)

	// lengthDifference should return a difference of
	// lengths between string before and after the change
	// return:
	//   difference of lengths
	lengthDifference() int
}

// CreateMark is a factory for Mark interface
func CreateMark(prefix string) Mark {
	return &CommonMark{
		used:  false,
		begin: len(prefix),
	}
}
