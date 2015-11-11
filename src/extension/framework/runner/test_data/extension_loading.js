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

function testExtension1Loaded() {
  from_extension1();
}

function testExtension2NotLoaded() {
  if (typeof from_extension2 !== "undefined") {
    Fail("Extension 2 should not be loaded!");
  }
}
