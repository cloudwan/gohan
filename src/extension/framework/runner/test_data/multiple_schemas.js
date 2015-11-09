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


var SCHEMAS = ["./schema.yaml", "./other_schema.yaml"];
var PATH = "/v1.0/networks";

var network = {
  "id": "4e8e5957-649f-477b-9e5b-f1f75b21c03c",
  "name": "Network 1",
  "tenant_id": "9bacb3c5d39d41a79512987f338cf177",
  "admin_state_up": false,
  "shared": false
};

var test_object = {
  "id": "4e8e5957-649f-477b-9e5b-f1f75b21c03c",
  "name": "Test Object 1",
  "tenant_id": "9bacb3c5d39d41a79512987f338cf177"
};

function testBothSchemasLoaded() {
  var result = gohan_db_create(MockTransaction(), "network", network);
  if (!_.isEqual(network, result)) {
    Fail("Failed to create an object using the first schema file.");
  }

  result = gohan_db_create(MockTransaction(), "test_object", test_object);
  if (!_.isEqual(test_object, result)) {
    Fail("Failed to create an object using the second schema file.");
  }
}
