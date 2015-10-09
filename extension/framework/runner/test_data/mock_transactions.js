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

var network1 = {
  "id": "abc",
  "name": "net",
  "tenant_id": "tenant",
  "shared": false,
  "admin_state_up": false
};

function testMockTransactions() {
  var transaction = MockTransaction();
  var resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) != JSON.stringify([])) {
    Fail("Invalid resources - expected an empty array");
  }

  resp = gohan_db_create(transaction, "network", network1);

  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1])) {
    Fail("Invalid resources in first transaction - expected %s, received %s",
        JSON.stringify([network1]), JSON.stringify(resources));
  }

  CommitMockTransaction();

  transaction = MockTransaction();

  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1])) {
    Fail("Invalid resources in second transaction - expected %s, received %s",
        JSON.stringify([network1]), JSON.stringify(resources));
  }
}
