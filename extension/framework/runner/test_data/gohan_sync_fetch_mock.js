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

var SCHEMA_INCLUDES = [];
var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testUnexpectedCall() {
    var result = gohan_sync_fetch("unexpectedKey");
}

function testLastThrow() {
    var mock_response1 = 1;

    gohan_sync_fetch.Expect().Return(1);
    gohan_sync_fetch.Expect().Return(null);
    gohan_sync_fetch.Expect().Throw("TimeOutException");
    var response1 = gohan_sync_fetch();
    if (response1 !== mock_response1) {
        Fail();
    }
    var mock_response2 = null;
    var response2 = gohan_sync_fetch();
    if (response2 !== mock_response2) {
        Fail();
    }

    try {
        gohan_sync_fetch()
    } catch(error) {
        if (error instanceof TimeOutException) {
            return;
        }
    }
    Fail()
}

function testWrongThrow() {
    gohan_sync_fetch.Expect().Throw("FakeException");
    try {
        gohan_sync_fetch()
    } catch(error) {
        if (error instanceof TimeOutException) {
            return;
        }
    }
    Fail()
}