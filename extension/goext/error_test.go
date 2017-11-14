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
	"net/http"
	"runtime"
	"testing"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goext/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGoext(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Goext suite")
}

var _ = Describe("Error", func() {
	Context("Error stacking should work", func() {
		var (
			errSingle *goext.Error
			errDouble *goext.Error
		)
		BeforeEach(func() {
			errSingle = goext.NewError(http.StatusInternalServerError, fmt.Errorf("it does not work"))
			errSingle.Origin = "first.go:100"

			errDouble = goext.NewError(http.StatusServiceUnavailable, goext.NewError(http.StatusUseProxy, fmt.Errorf("nothing here")))
			errDouble.Origin = "one.go:1000"
			errDouble.Err.(*goext.Error).Origin = "two.go:2000"
		})

		It("Should stack errors", func() {
			Expect(errSingle).To(Not(BeNil()))
			Expect(errSingle.Status).To(Equal(http.StatusInternalServerError))
			Expect(errSingle.Err).To(Equal(fmt.Errorf("it does not work")))

			Expect(errDouble).To(Not(BeNil()))
			Expect(errDouble.Status).To(Equal(http.StatusServiceUnavailable))
			Expect(errDouble.Err.(*goext.Error).Status).To(Equal(http.StatusUseProxy))
			Expect(errDouble.Err.(*goext.Error).Err).To(Equal(fmt.Errorf("nothing here")))
		})

		It("Should return top stack error by default", func() {
			Expect(fmt.Sprintf("%s", errSingle)).To(Equal("HTTP 500 (Internal Server Error): it does not work"))
			Expect(fmt.Sprintf("%s", errDouble)).To(Equal("HTTP 503 (Service Unavailable): nothing here"))
		})

		It("Should return full error stack on request", func() {
			Expect(fmt.Sprintf("%s", errSingle.ErrorStack())).To(Equal("HTTP 500 (Internal Server Error) at first.go:100: it does not work"))
			Expect(fmt.Sprintf("%s", errDouble.ErrorStack())).To(Equal(
				`HTTP 503 (Service Unavailable) at one.go:1000 from
  <- HTTP 305 (Use Proxy) at two.go:2000: nothing here`))
		})

		It("Should capture correct stack", func() {
			error := goext.NewErrorInternalServerError(fmt.Errorf("test error"))
			_, _, line, _ := runtime.Caller(0)
			Expect(error.Origin).To(HaveSuffix(fmt.Sprintf("github.com/cloudwan/gohan/extension/goext/error_test.go:%d", line-1)))
		})

		It("ErrorMatcher should match errors", func() {
			someError := goext.NewErrorInternalServerError(fmt.Errorf("some internal error"))
			otherError := goext.NewErrorInternalServerError(fmt.Errorf("other internal error"))
			errorMatcher := goext_test.MatchError(someError)
			Expect(errorMatcher.Match(someError)).To(BeTrue())
			Expect(errorMatcher.Match(otherError)).To(BeFalse())
			Expect(errorMatcher.FailureMessage(otherError)).To(Equal(fmt.Sprintf("Expected\n\t%#v\nto be\n\t%#v", otherError, someError)))
			Expect(errorMatcher.NegatedFailureMessage(otherError)).To(Equal(fmt.Sprintf("Expected\n\t%#v\nnot to be\n\t%#v", otherError, someError)))
			Expect(someError).To(errorMatcher)
		})
	})
})
