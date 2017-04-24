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

(function jsPlugin() {
    gohan_log_error("(PP) Initializing JS ext");

    var schema = gohan.getSchema("pingpongJS");

    var gohan_register_handler = this.gohan_register_handler || this.gohan_regist_handler;

    schema.registerEventHandler("ping", function(context, resource) {
        gohan_log_error("(PP) JS: Handle ping on schema pingpongJS, Trigger pong");
        this.gohan_trigger_event("pong", context);
    });

    gohan_register_handler("ping", function(context) {
        gohan_log_error("(PP) JS: Handle global ping");
    })

    gohan_register_handler("pong", function(context) {
        gohan_log_error("(PP) JS: Handle pong");
    });
})();
