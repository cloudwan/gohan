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
	"reflect"
	"runtime"
	"testing"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goext/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func TestGoext(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Goext suite")
}

var _ = Describe("Error", func() {
	Context("Error stacking should work", func() {
		var (
			errSingle      *goext.Error
			errDouble      *goext.Error
			errDefaultCtor *goext.Error
		)
		BeforeEach(func() {
			errSingle = goext.NewError(http.StatusInternalServerError, errors.New("it does not work"))
			errSingle.Origin = "first.go:100"

			errDouble = goext.NewError(http.StatusServiceUnavailable, goext.NewError(http.StatusUseProxy, errors.New("nothing here")))
			errDouble.Origin = "one.go:1000"
			errDouble.Err.(*goext.Error).Origin = "two.go:2000"

			errDefaultCtor = &goext.Error{}
		})

		It("Should stack errors", func() {
			Expect(errSingle).To(Not(BeNil()))
			Expect(errSingle.Status).To(Equal(http.StatusInternalServerError))
			Expect(errSingle.Err.Error()).To(Equal("it does not work"))

			Expect(errDouble).To(Not(BeNil()))
			Expect(errDouble.Status).To(Equal(http.StatusServiceUnavailable))
			Expect(errDouble.Err.(*goext.Error).Status).To(Equal(http.StatusUseProxy))
			Expect(errDouble.Err.(*goext.Error).Err.Error()).To(Equal("nothing here"))
		})

		It("Shouldn't panic when inner err is nil", func() {
			Expect(func() { errDefaultCtor.ErrorStack() }).To(Not(Panic()))
		})

		It("Should return top stack error by default", func() {
			Expect(fmt.Sprintf("%s", errSingle)).To(Equal("HTTP 500 (Internal Server Error): it does not work"))
			Expect(fmt.Sprintf("%s", errDouble)).To(Equal("HTTP 503 (Service Unavailable): nothing here"))
		})

		It("Should return full error stack on request", func() {
			Expect(fmt.Sprintf("%s", errSingle.ErrorStack())).To(ContainSubstring("HTTP 500 (Internal Server Error) at first.go:100: it does not work"))
			Expect(fmt.Sprintf("%s", errDouble.ErrorStack())).To(ContainSubstring(
				`HTTP 503 (Service Unavailable) at one.go:1000 from
  <- HTTP 305 (Use Proxy) at two.go:2000: nothing here`))
		})

		It("Should capture correct stack", func() {
			error := goext.NewErrorInternalServerError(errors.New("test error"))
			_, _, line, _ := runtime.Caller(0)
			Expect(error.Origin).To(HaveSuffix(fmt.Sprintf("github.com/cloudwan/gohan/extension/goext/error_test.go:%d", line-1)))
		})

		It("Should capture full stack for wrapped errors", func() {
			var innerLine int
			innerFunc := func() *goext.Error {
				_, _, line, _ := runtime.Caller(0)
				innerLine = line + 2
				return goext.NewErrorBadGateway(fmt.Errorf("test error"))
			}

			var middleLine int
			middleFunc := func() error {
				_, _, line, _ := runtime.Caller(0)
				middleLine = line + 2
				return innerFunc()
			}

			var outerLine int
			outerFunc := func() *goext.Error {
				_, _, line, _ := runtime.Caller(0)
				outerLine = line + 2
				return goext.NewErrorBadRequest(middleFunc())
			}

			_, _, line, _ := runtime.Caller(0)
			callLine := line + 2
			err := outerFunc()
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", innerLine)))
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", middleLine)))
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", outerLine)))
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", callLine)))

			Expect(err.ErrorStack()).To(ContainSubstring("Bad Gateway"))
			Expect(err.ErrorStack()).To(ContainSubstring("Bad Request"))
		})

		It("Should detect dependent library changes", func() {
			// if this test fails it means that the implementation of github.com/pkg/errors has changed.
			// you probably should adapt the reflection code in NewError.
			errType := reflect.TypeOf(errors.New("test error")).String()
			Expect(errType).To(Equal("*errors.fundamental"))

			errType = reflect.TypeOf(errors.Errorf("test %s", "error")).String()
			Expect(errType).To(Equal("*errors.fundamental"))
		})

		It("Should capture full stack for built-in errors", func() {
			var innerLine int
			innerFunc := func() error {
				_, _, line, _ := runtime.Caller(0)
				innerLine = line + 2
				return errors.New("test error")
			}

			var outerLine int
			outerFunc := func() *goext.Error {
				_, _, line, _ := runtime.Caller(0)
				outerLine = line + 2
				return goext.NewErrorBadRequest(innerFunc())
			}

			_, _, line, _ := runtime.Caller(0)
			callLine := line + 2
			err := outerFunc()
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", innerLine)))
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", outerLine)))
			Expect(err.ErrorStack()).To(ContainSubstring(fmt.Sprintf("error_test.go:%d", callLine)))

			Expect(err.ErrorStack()).To(ContainSubstring("Bad Request"))
		})

		It("ErrorMatcher should match errors", func() {
			someError := goext.NewErrorInternalServerError(errors.New("some internal error"))
			otherError := goext.NewErrorInternalServerError(errors.New("other internal error"))
			errorMatcher := goext_test.MatchError(someError)
			Expect(errorMatcher.Match(someError)).To(BeTrue())
			Expect(errorMatcher.Match(otherError)).To(BeFalse())
			Expect(errorMatcher.FailureMessage(otherError)).To(Equal(fmt.Sprintf("Expected\n\t%#v\nto be\n\t%#v", otherError, someError)))
			Expect(errorMatcher.NegatedFailureMessage(otherError)).To(Equal(fmt.Sprintf("Expected\n\t%#v\nnot to be\n\t%#v", otherError, someError)))
			Expect(someError).To(errorMatcher)
		})
	})
})
