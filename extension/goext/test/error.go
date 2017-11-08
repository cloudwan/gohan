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

package goext_test

import (
	"fmt"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/onsi/gomega/types"
)

// MatchErrorMatcher matches goext.Error
//
// example:
// Expect(myFunction(myParams)).To(goext.MatchError(goext.NewErrorInternalServerError(myModule.ErrMyError)))
type MatchErrorMatcher struct {
	Expected *goext.Error
}

func (matcher *MatchErrorMatcher) Match(actual interface{}) (bool, error) {
	error, ok := actual.(*goext.Error)
	if !ok {
		return false, fmt.Errorf("ErrorMatcher matcher expects an Error")
	}
	return error.Err.Error() == matcher.Expected.Err.Error() && error.Status == matcher.Expected.Status, nil
}

func (matcher *MatchErrorMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto be\n\t%#v", actual, matcher.Expected)
}

func (matcher *MatchErrorMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to be\n\t%#v", actual, matcher.Expected)
}

func MatchError(expected *goext.Error) types.GomegaMatcher {
	return &MatchErrorMatcher{
		Expected: expected,
	}
}
