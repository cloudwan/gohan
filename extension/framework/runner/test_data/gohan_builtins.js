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

function testGohanBuiltins() {
  var transaction = MockTransaction();
  var network1 = {
      "id": "abc",
      "name": "net",
      "tenant_id": "tenant",
      "shared": false,
      "admin_state_up": false
  };
  var network2 = {
      "id": "def",
      "name": "nonet",
      "tenant_id": "tenant",
      "shared": false,
      "admin_state_up": false
  };

  var resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) != JSON.stringify([])) {
    Fail("Invalid resources - expected an empty array");
  }

  var resp = gohan_db_create(transaction, "network", network1);
  if (!_.isUndefined(resp.error)) {
    Fail("Failed to create a resource: %s", JSON.stringify(resp.error));
  }
  resp = gohan_db_create(transaction, "network", network2);
  if (!_.isUndefined(resp.error)) {
    Fail("Failed to create a resource: %s", JSON.stringify(resp.error));
  }

  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1, network2])) {
    Fail("Invalid resources - expected %s, received %s",
        JSON.stringify([network1, network2]), JSON.stringify(resources));
  }

  network1.admin_state_up = true;
  gohan_db_update(transaction, "network", network1);
  var resource = gohan_db_fetch(transaction, "network", "abc", "tenant");

  if (JSON.stringify(resource) !== JSON.stringify(network1)) {
    FAIL("Invalid resource - expected %s, received %s",
        JSON.stringify(network1), JSON.stringify(resource));
  }

  gohan_db_delete(transaction, "network", network2.id);
  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1])) {
    Fail("Invalid resources - expected %s, received %s",
        JSON.stringify([network1]), JSON.stringify(resources));
  }

  gohan_db_delete(transaction, "network", network1.id);
  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) != JSON.stringify([])) {
    Fail("Invalid resources - expected an empty array");
  }
}
