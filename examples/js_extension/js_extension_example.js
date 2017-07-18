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
    gohan_log_info("Example log from goext extension");

    var schemas = gohan_schemas();
    gohan_log_info("Number of schemas: " + schemas.length);

    _.each(schemas, function(s) {
        gohan_log_debug("Found schema: " + s.ID);
    });

    var todoSchema = null;

    _.each(schemas, function(s) {
        if (s.ID == "todo") {
            todoSchema = s;
            return;
        }
    });

    if (todoSchema == null) {
        gohan_log_error("schema todo not found");
		return;
    }

    // list
	var tx = gohan_db_transaction();

    var startList = new Date().getTime();

    var todos = gohan_db_list(tx, "todo", {});

    var stopList = new Date().getTime();
    gohan_log_warning("List time: " + (stopList - startList) + " [ms]");

    gohan_log_info("Found " + todos.length + " TODO resources");

})();
