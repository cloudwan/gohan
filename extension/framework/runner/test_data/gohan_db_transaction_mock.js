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

var SCHEMA_INCLUDES = [];
var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testUnexpectedCall() {
  var tx = gohan_db_transaction();
}

function testSingleReturnSingleCall() {
  var mock_transaction = "mock_transaction";
  gohan_db_transaction.Expect().Return(mock_transaction);
  var tx = gohan_db_transaction();
  if (tx !== mock_transaction) {
    Fail();
  }
}

function testSingleReturnMultipleCalls() {
  var mock_transaction = "mock_transaction";
  gohan_db_transaction.Expect().Return(mock_transaction);
  var tx = gohan_db_transaction();
  if (tx !== mock_transaction) {
    Fail();
  }

  var unexpected_tx = gohan_db_transaction();
}

function testWrongArgumentsCall() {
  var mock_transaction = "mock_transaction";
  gohan_db_transaction.Expect().Return(mock_transaction);
  var tx = gohan_db_transaction("something extra");
  if (tx !== mock_transaction) {
    Fail();
  }
}
