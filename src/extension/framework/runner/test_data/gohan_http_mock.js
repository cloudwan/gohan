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


var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testUnexpectedCall() {
  result = gohan_http("GET", "http://www.xyz.com", {}, {});
}

function testSingleReturnSingleCall() {
  var mock_response = "OK";
  gohan_http.Expect("GET", "http://www.xyz.com", {}, {}).Return(mock_response);
  var response = gohan_http("GET", "http://www.xyz.com", {}, {});
  if (response !== mock_response) {
    Fail();
  }
}

function testSingleReturnMultipleCalls() {
  var mock_response = "OK";
  gohan_http.Expect("GET", "http://www.xyz.com", {}, {}).Return(mock_response);
  var response = gohan_http("GET", "http://www.xyz.com", {}, {});
  if (response !== mock_response) {
    Fail();
  }

  var unexpected_response = gohan_http("GET", "http://www.xyz.com", {}, {});
}

function testWrongArgumentsCall() {
  var mock_response = "OK";
  gohan_http.Expect("POST", "http://www.abc.com", {"a": "a"}, {}).Return(mock_response);
  var response = gohan_http("GET", "http://www.xyz.com", {}, {});
  if (response !== mock_response) {
    Fail();
  }
}
