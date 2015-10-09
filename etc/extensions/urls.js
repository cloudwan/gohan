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


function substitutePrefix(namespacePrefix, schema) {
  schema.prefix = gohan_schema_url(schema.id);
}

gohan_register_handler("post_list", function(context) {
  for (var i = 0; i < context.response.schemas.length; ++i) {
    substitutePrefix(context.namespace_prefix, context.response.schemas[i]);
  }
});

gohan_register_handler("post_show", function(context) {
  substitutePrefix(context.namespace_prefix, context.response);
});

