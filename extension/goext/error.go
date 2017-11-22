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

package goext

import (
	"fmt"
	"net/http"
	"runtime"
)

// Error represents an error code with related HTTP status
type Error struct {
	Err    error
	Status int
	Origin string
}

// NewError returns a new error
func NewError(status int, err error) *Error {
	return &Error{
		Err:    err,
		Status: status,
		Origin: captureOrigin(),
	}
}

func captureOrigin() string {
	const SkipStackFrames = 3
	// 0: captureOrigin()
	// 1: NewError(...)
	// 2: NewErrorXXX(...)
	// 3: some_extension.go:123
	_, file, line, ok := runtime.Caller(SkipStackFrames)
	if !ok {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// Root returns root error that is at the bottom of the error stack; first error that is not *goext.Error
func (e Error) Root() error {
	switch e.Err.(type) {
	case *Error:
		return e.Err.(*Error).Root()
	default:
		return e.Err
	}
}

// Error returns formatted error message
func (e Error) Error() string {
	return fmt.Sprintf("HTTP %d (%s): %s", e.Status, http.StatusText(e.Status), e.Root())
}

// ErrorStack returns formatted full error stack message
func (e Error) ErrorStack() string {
	switch e.Err.(type) {
	case *Error:
		return fmt.Sprintf("HTTP %d (%s) at %s from", e.Status, http.StatusText(e.Status), e.Origin) + walkErrorStack(e.Err.(*Error))
	default:
		return fmt.Sprintf("HTTP %d (%s) at %s: %s", e.Status, http.StatusText(e.Status), e.Origin, e.Err)
	}
}

func walkErrorStack(e *Error) string {
	if e == nil {
		return ""
	}

	switch e.Err.(type) {
	case *Error:
		return fmt.Sprintf("\n  <- HTTP %d (%s) at %s from", e.Status, http.StatusText(e.Status), e.Origin) + walkErrorStack(e.Err.(*Error))
	default:
		return fmt.Sprintf("\n  <- HTTP %d (%s) at %s: %s", e.Status, http.StatusText(e.Status), e.Origin, e.Err)
	}
}

// 1xx
func NewErrorContinue(err error) *Error { return NewError(http.StatusContinue, err) }
func NewErrorSwitchingProtocols(err error) *Error {
	return NewError(http.StatusSwitchingProtocols, err)
}
func NewErrorProcessing(err error) *Error { return NewError(http.StatusProcessing, err) }

// 2xx
func NewErrorOK(err error) *Error       { return NewError(http.StatusOK, err) }
func NewErrorCreated(err error) *Error  { return NewError(http.StatusCreated, err) }
func NewErrorAccepted(err error) *Error { return NewError(http.StatusAccepted, err) }
func NewErrorNonAuthoritativeInfo(err error) *Error {
	return NewError(http.StatusNonAuthoritativeInfo, err)
}
func NewErrorNoContent(err error) *Error       { return NewError(http.StatusNoContent, err) }
func NewErrorResetContent(err error) *Error    { return NewError(http.StatusResetContent, err) }
func NewErrorPartialContent(err error) *Error  { return NewError(http.StatusPartialContent, err) }
func NewErrorMultiStatus(err error) *Error     { return NewError(http.StatusMultiStatus, err) }
func NewErrorAlreadyReported(err error) *Error { return NewError(http.StatusAlreadyReported, err) }
func NewErrorIMUsed(err error) *Error          { return NewError(http.StatusIMUsed, err) }

// 3xx
func NewErrorMultipleChoices(err error) *Error   { return NewError(http.StatusMultipleChoices, err) }
func NewErrorMovedPermanently(err error) *Error  { return NewError(http.StatusMovedPermanently, err) }
func NewErrorFound(err error) *Error             { return NewError(http.StatusFound, err) }
func NewErrorSeeOther(err error) *Error          { return NewError(http.StatusSeeOther, err) }
func NewErrorNotModified(err error) *Error       { return NewError(http.StatusNotModified, err) }
func NewErrorUseProxy(err error) *Error          { return NewError(http.StatusUseProxy, err) }
func NewErrorTemporaryRedirect(err error) *Error { return NewError(http.StatusTemporaryRedirect, err) }
func NewErrorPermanentRedirect(err error) *Error { return NewError(http.StatusPermanentRedirect, err) }

// 4xx
func NewErrorBadRequest(err error) *Error        { return NewError(http.StatusBadRequest, err) }
func NewErrorUnauthorized(err error) *Error      { return NewError(http.StatusUnauthorized, err) }
func NewErrorPaymentRequired(err error) *Error   { return NewError(http.StatusPaymentRequired, err) }
func NewErrorForbidden(err error) *Error         { return NewError(http.StatusForbidden, err) }
func NewErrorNotFound(err error) *Error          { return NewError(http.StatusNotFound, err) }
func NewErrorMethodNotAllowed(err error) *Error  { return NewError(http.StatusMethodNotAllowed, err) }
func NewErrorNotAcceptable(err error) *Error     { return NewError(http.StatusNotAcceptable, err) }
func NewErrorProxyAuthRequired(err error) *Error { return NewError(http.StatusProxyAuthRequired, err) }
func NewErrorRequestTimeout(err error) *Error    { return NewError(http.StatusRequestTimeout, err) }
func NewErrorConflict(err error) *Error          { return NewError(http.StatusConflict, err) }
func NewErrorGone(err error) *Error              { return NewError(http.StatusGone, err) }
func NewErrorLengthRequired(err error) *Error    { return NewError(http.StatusLengthRequired, err) }
func NewErrorPreconditionFailed(err error) *Error {
	return NewError(http.StatusPreconditionFailed, err)
}
func NewErrorRequestEntityTooLarge(err error) *Error {
	return NewError(http.StatusRequestEntityTooLarge, err)
}
func NewErrorRequestURITooLong(err error) *Error { return NewError(http.StatusRequestURITooLong, err) }
func NewErrorUnsupportedMediaType(err error) *Error {
	return NewError(http.StatusUnsupportedMediaType, err)
}
func NewErrorRequestedRangeNotSatisfiable(err error) *Error {
	return NewError(http.StatusRequestedRangeNotSatisfiable, err)
}
func NewErrorExpectationFailed(err error) *Error { return NewError(http.StatusExpectationFailed, err) }
func NewErrorTeapot(err error) *Error            { return NewError(http.StatusTeapot, err) }
func NewErrorUnprocessableEntity(err error) *Error {
	return NewError(http.StatusUnprocessableEntity, err)
}
func NewErrorLocked(err error) *Error           { return NewError(http.StatusLocked, err) }
func NewErrorFailedDependency(err error) *Error { return NewError(http.StatusFailedDependency, err) }
func NewErrorUpgradeRequired(err error) *Error  { return NewError(http.StatusUpgradeRequired, err) }
func NewErrorPreconditionRequired(err error) *Error {
	return NewError(http.StatusPreconditionRequired, err)
}
func NewErrorTooManyRequests(err error) *Error { return NewError(http.StatusTooManyRequests, err) }
func NewErrorRequestHeaderFieldsTooLarge(err error) *Error {
	return NewError(http.StatusRequestHeaderFieldsTooLarge, err)
}
func NewErrorUnavailableForLegalReasons(err error) *Error {
	return NewError(http.StatusUnavailableForLegalReasons, err)
}

// 5xx
func NewErrorInternalServerError(err error) *Error {
	return NewError(http.StatusInternalServerError, err)
}
func NewErrorNotImplemented(err error) *Error { return NewError(http.StatusNotImplemented, err) }
func NewErrorBadGateway(err error) *Error     { return NewError(http.StatusBadGateway, err) }
func NewErrorServiceUnavailable(err error) *Error {
	return NewError(http.StatusServiceUnavailable, err)
}
func NewErrorGatewayTimeout(err error) *Error { return NewError(http.StatusGatewayTimeout, err) }
func NewErrorHTTPVersionNotSupported(err error) *Error {
	return NewError(http.StatusHTTPVersionNotSupported, err)
}
func NewErrorVariantAlsoNegotiates(err error) *Error {
	return NewError(http.StatusVariantAlsoNegotiates, err)
}
func NewErrorInsufficientStorage(err error) *Error {
	return NewError(http.StatusInsufficientStorage, err)
}
func NewErrorLoopDetected(err error) *Error { return NewError(http.StatusLoopDetected, err) }
func NewErrorNotExtended(err error) *Error  { return NewError(http.StatusNotExtended, err) }
func NewErrorNetworkAuthenticationRequired(err error) *Error {
	return NewError(http.StatusNetworkAuthenticationRequired, err)
}
