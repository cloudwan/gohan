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

// Header represents HTTP header
type Header map[string][]string

// Response represents HTTP response
type Response struct {
	Code   int
	Status string
	Header Header
	Body   string
}

// IHTTP is an interface to http in Gohan
type IHTTP interface {
	Request(method, rawURL string, headers map[string]interface{}, postData interface{}, opaque bool, timeout int) (*Response, error)
	RequestRaw(method, rawURL string, headers map[string]string, rawData string) (*Response, error)
}
