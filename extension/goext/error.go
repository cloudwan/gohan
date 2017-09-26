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

import "fmt"

// ErrorCode identifies an error code
type ErrorCode int

// Error represents an error
type Error struct {
	Code ErrorCode
	Err  error
}

// NewError returns a new error
func NewError(code ErrorCode, err error) Error {
	return Error{
		Code: code,
		Err:  err,
	}
}

// Error returns error message
func (e Error) Error() string {
	return fmt.Sprintf("%s (HTTP code: %d)", e.Err, e.Code)
}

var (
	// ErrorBadRequest indicates bad request error
	ErrorBadRequest = ErrorCode(400)

	// ErrorConflict indicates conflict error
	ErrorConflict = ErrorCode(409)

	// ErrorNotFound indicates not found error
	ErrorNotFound = ErrorCode(404)

	// ErrorInternalServerError indicates internal server error
	ErrorInternalServerError = ErrorCode(500)

	// ErrorNotImplemented indicates not implemented error
	ErrorNotImplemented = ErrorCode(501)
)

