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

import "strings"

const new = "common"

// CommonMark is a mark that changes names to common
type CommonMark struct {
	used  bool
	begin int
	end   int
}

// Change implementation
func (commonMark *CommonMark) Change(string *string) bool {
	if strings.HasPrefix((*string)[commonMark.begin:], new) {
		return false
	}
	result := (*string)[:commonMark.begin] + new
	if commonMark.used {
		result += (*string)[commonMark.end:]
	} else {
		commonMark.used = true
		commonMark.end = len(*string)
	}
	*string = result
	return true
}

// Update implementation
func (commonMark *CommonMark) Update(mark Mark) {
	difference := mark.lengthDifference()
	commonMark.begin += difference
	commonMark.end += difference
}

// lengthDifference implementation
func (commonMark *CommonMark) lengthDifference() int {
	if commonMark.used {
		return len(new) - commonMark.end + commonMark.begin
	}
	return 0
}
