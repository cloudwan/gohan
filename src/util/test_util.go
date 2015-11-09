// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package util

import (
	"encoding/json"
	"fmt"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

//MatchAsJSON checks whether the provided arguments are equivalent after being marshalled as JSONs
func MatchAsJSON(expected interface{}) types.GomegaMatcher {
	return &theJSONMatcher{
		expected: expected,
	}
}

type theJSONMatcher struct {
	expected interface{}
}

func (matcher *theJSONMatcher) Match(actual interface{}) (bool, error) {
	expectedJSON, err := json.Marshal(matcher.expected)
	if err != nil {
		return false, fmt.Errorf("Could not marshal expected: %v", err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return false, fmt.Errorf("Could not marshal actual: %v", err)
	}
	return gomega.MatchJSON(expectedJSON).Match(actualJSON)
}

func (matcher *theJSONMatcher) FailureMessage(actual interface{}) string {
	return format.Message(actual, "to match as JSON", matcher.expected)
}

func (matcher *theJSONMatcher) NegatedFailureMessage(actual interface{}) string {
	return format.Message(actual, "not to match as JSON", matcher.expected)
}
