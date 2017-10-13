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

// Element is an interface for set elements
type Element interface {
	Name() string
}

type byName []Element

func (array byName) Len() int {
	return len(array)
}

func (array byName) Swap(i, j int) {
	array[i], array[j] = array[j], array[i]
}

func (array byName) Less(i, j int) bool {
	return array[i].Name() < array[j].Name()
}
