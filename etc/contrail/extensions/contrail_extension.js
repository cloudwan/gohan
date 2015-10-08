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


gohan_register_handler("pre_list", function(context){
  var plural = context.schema.Plural
  var schema_id = context.schema.ID
  var token = context.auth.Access.Token.Id
  var contrail_response = gohan_http(
    "GET", CONTRAIL_URL + plural + "?detail=true",
    {"X-Auth-Token": token}, null)

  var contrail_data = JSON.parse(contrail_response.body)
  var response = []
  _.each(contrail_data[plural], function(data){
    data[schema_id].id = data[schema_id].uuid
    response.push(data[schema_id])
  });

  context.response = {}
  context.response[plural] = response
});


gohan_register_handler("pre_create", function(context){
  var plural = context.schema.Plural
  var schema_id = context.schema.ID
  var token = context.auth.Access.Token.Id
  console.log(context.resource)
  var request = {}
  request[schema_id] = context.resource
  var contrail_response = gohan_http("POST", CONTRAIL_URL + plural,
                                     {"X-Auth-Token": token,
                                      "Content-Type": "application/json"
                                     },
                                     request)

  console.log(contrail_response.body)
  var contrail_data = JSON.parse(contrail_response.body)
  context.response = contrail_data
});


gohan_register_handler("pre_delete", function(context){
  var plural = context.schema.Plural
  var schema_id = context.schema.ID
  var id = context.id
  var token = context.auth.Access.Token.Id
  var contrail_response = gohan_http(
    "DELETE",
    CONTRAIL_URL + schema_id + "/" + id,
    {"X-Auth-Token": token},
    null)
  context.response = "deleted"
});


gohan_register_handler("pre_update", function(context){
  var schema_id = context.schema.ID
  var id = context.id
  var token = context.auth.Access.Token.Id
  var request = {}
  request[schema_id] = context.resource
  var contrail_response = gohan_http("PUT", CONTRAIL_URL + schema_id + "/" + id,
                                     {"X-Auth-Token": token,
                                      "Content-Type": "application/json"
                                     },
                                     request)
  var contrail_data = JSON.parse(contrail_response.body)
  context.response = contrail_data});


gohan_register_handler("pre_show", function(context){
  var schema_id = context.schema.ID;
  var id = context.id;
  var token = context.auth.Access.Token.Id;
  var contrail_response = gohan_http(
    "GET",
    CONTRAIL_URL + schema_id + "/" + id,
    {"X-Auth-Token": token}, null);
  var contrail_data = JSON.parse(contrail_response.body);
  context.response = contrail_data});
