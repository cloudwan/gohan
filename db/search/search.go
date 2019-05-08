// Copyright (C) 2019 NTT Innovation Institute, Inc.
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

package search

import (
	"strings"
)

type Search struct {
	Value string
}

func NewSearchField(value string) Search {
	searchValue := strings.Builder{}
	searchValue.WriteString("%")
	searchValue.WriteString(escapeSpecialChars(value))
	searchValue.WriteString("%")
	return Search{Value: searchValue.String()}
}

func escapeSpecialChars(value string) string{
	value = strings.Replace(value, "\\", "\\\\", -1)
	value = strings.Replace(value, "_", "\\_", -1)
	value = strings.Replace(value, "%", "\\%", -1)
	return value
}
