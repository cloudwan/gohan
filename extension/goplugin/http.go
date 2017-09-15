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

package goplugin

import (
	"context"
	"io/ioutil"
	net_http "net/http"
	"strings"
	"time"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/otto"
)

// HTTP is an implementation of IHTTP
type HTTP struct{}

// Request performs http request
func (http *HTTP) Request(method, rawURL string, headers map[string]interface{}, postData interface{}, opaque bool, timeout int) (*goext.Response, error) {
	log.Debug("gohan_http  [%s] %s %s %t %d", method, headers, rawURL, opaque, timeout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()
	code, header, body, error := otto.GohanHTTP(ctx, method, rawURL, headers, postData, opaque)
	return &goext.Response{Code: code, Header: convertHeader(header), Body: body}, error
}

// RequestRaw performs raw http request
func (http *HTTP) RequestRaw(method, rawURL string, headers map[string]string, rawData string) (*goext.Response, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// prepare request
	req, err := net_http.NewRequest(method, rawURL, strings.NewReader(rawData))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	// set headers
	for header, value := range headers {
		req.Header.Set(header, value)
	}

	var resp *net_http.Response

	// run query
	done := make(chan struct{})
	go func() {
		resp, err = net_http.DefaultTransport.RoundTrip(req)
		close(done)
	}()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &goext.Response{Code: resp.StatusCode, Status: resp.Status, Header: convertHeader(resp.Header), Body: string(body)}, nil
}

func convertHeader(header net_http.Header) goext.Header {
	ret := make(map[string][]string, len(header))
	for k, v := range header {
		ret[k] = v
	}
	return ret
}
