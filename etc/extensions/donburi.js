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

var donburi = {
  plugin: {},
  no_template_keywords: ['eval', 'block',
                         'resources',
                         'netconf_close', 'netconf_exec',
                         'ssh_close', 'ssh_exec'],
  register_plugin: function(name, plugin){
    this.plugin[name] = plugin
  },
  apply_template: function(context, data){
    var self = this;
    if(_.isString(data)){
      return gohan_template(data, context);
    }
    if(_.isArray(data)){
      var result = [];
      _.each(data, function(sub_data){
        result.push(self.apply_template(context, sub_data));
      });
      return result;
    }
    if(_.isObject(data)){
      var result = {};
      _.each(data, function(sub_data, key){
        result[key] = self.apply_template(context, sub_data);
      });
      return result;
    }
    return data
  }
  run_task: function(event_type, context, task){
    var self = this;
    if(!_.isUndefined(task.when)){
      with(context){
        try{
          if(eval(task.when) == false){
            if(!_.isUndefined(task["else"])){
              try{
                self.run_tasks(event_type, context, task["else"])
              }catch(err){
                context.error = err;
              }
            }
            return;
          }
        }catch(err){
          context.error = err;
          return;
        }
      }
    }
    var result;
    var retry_count = task.retry;
    if(_.isUndefined(retry_count)){
      retry_count = 1;
    }
    for(var retry = 0; retry < retry_count; retry++){
      try{
        _.each(_.keys(task), function(key){
          var plugin = self.plugin[key];
          if(_.isUndefined(plugin)){
            return;
          }
          var with_items = task.with_items;
          var with_dict = task.with_dict;
          if(_.isUndefined(with_items) && _.isUndefined(with_dict)){
            result = self.run_plugin(plugin, event_type, context, key, task[key]);
          }else{
            if(_.isString(with_items)){
              with(context){
                with_items = eval(with_items);
              }
            }
            if(_.isArray(with_items)){
              result = [];
              _.each(with_items, function(item){
                context.item = item;
                var sub_result = self.run_plugin(plugin, event_type, context, key, task[key]);
                result.push(sub_result);
              });
            }
            if(_.isString(with_dict)){
              with(context){
                with_dict = eval(with_dict);
              }
            }
            if(_.isObject(with_dict)){
              result = {};
              _.each(with_dict, function(value, item_key){
                context.item = {key: item_key, value:value};
                var sub_result = self.run_plugin(plugin, event_type, context, key, task[key]);
                result[key] = sub_result;
              });
            }
          }
        });
        if(!_.isUndefined(task.register)){
            context[task.register] = result;
        }
      }catch(err){
        context.error = err;
        if(!_.isUndefined(task.rescue)){
          try{
            self.run_tasks(event_type, context, task.rescue)
          }catch(err){
            context.error = err;
          }
        }else{
          throw err;
        }
      }
      if(!_.isUndefined(task.always)){
        try{
          self.run_tasks(event_type, context, task.always)
        }catch(err){
          context.error = err;
        }
      }
    }
  },
  run_plugin: function(plugin, event_type, context, key, value){
    var self = this;
    if(self.no_template_keywords.indexOf(key) == -1){
      value = self.apply_template(context, value);
    }
    return plugin(event_type, context, value);
  }
  run_tasks: function(event_type, context, tasks){
    var self = this;
    _.each(tasks, function(task){
      self.run_task(event_type, context, task);
    });
  },
  run: function(code){
    var self = this;
    var events = ["post_create",
                  "post_update",
                  "pre_delete",
                  "notification"]
    _.each(events, function(event_type){
      gohan_register_handler(event_type, function(context){
        try{
          self.run_tasks(event_type, context, code.tasks);
        }catch(err){
          context.error = err;
        }
      });
      var event_type_in_transaction = event_type + "_in_transaction"
      gohan_register_handler(event_type_in_transaction, function(context){
        try{
          self.run_tasks(event_type_in_transaction, context, code.db_tasks);
        }catch(err){
          context.error = err;
        }
      });
    });
  }
}

donburi.register_plugin("debug", function(event_type, context, value){
  console.log(value);
});

donburi.register_plugin("sleep", function(event_type, context, value){
  gohan_sleep(value);
});

donburi.register_plugin("block", function(event_type, context, value){
  donburi.run_tasks(event_type, context, value);
});

donburi.register_plugin("netconf_open", function(event_type, context, value){
	return gohan_netconf_open(value.host, value.username)
});

donburi.register_plugin("netconf_exec", function(event_type, context, value){
    var command = donburi.apply_template(context, value.command);
	with(context){
		return gohan_netconf_exec(eval(value.connection), command)
	}
});

donburi.register_plugin("netconf_close", function(event_type, context, value){
	with(context){
		gohan_netconf_close(eval(value))
	}
});

donburi.register_plugin("ssh_open", function(event_type, context, value){
	return gohan_ssh_open(value.host, value.username)
});

donburi.register_plugin("ssh_exec", function(event_type, context, value){
    var command = donburi.apply_template(context, value.command);
	with(context){
		return gohan_ssh_exec(eval(value.connection), command)
	}
});

donburi.register_plugin("ssh_close", function(event_type, context, value){
	with(context){
		gohan_ssh_close(eval(value))
	}
});

donburi.register_plugin("resources", function(event_type, context, value){
  if(event_type == "pre_delete" || event_type == "pre_delete_in_transaction"){
    value.reverse();
  }
  donburi.run_tasks(event_type, context, value);
});

donburi.register_plugin("command", function(event_type, context, value){
  return gohan_exec(value.name, value.args);
});

donburi.register_plugin("eval", function(event_type, context, value){
  with(context){
    return eval(value);
  }
});

donburi.register_plugin("vars", function(event_type, context, value){
  _.each(value, function(value, key){
    context[key] = value;
  });
});

donburi.register_plugin("list", function(event_type, context, value){
  var transaction = context.transaction;
  return gohan_db_list(transaction, value.schema, value.tenant_id);
});

donburi.register_plugin("fetch", function(event_type, context, value){
  var transaction = context.transaction;
  var value = gohan_db_fetch(transaction, value.schema, value.id, value.tenant_id);
  return value
});

donburi.register_plugin("resource", function(event_type, context, value){
  var transaction = context.transaction;
  if(event_type == "post_create_in_transaction" || event_type == "post_create"){
    if(_.isUndefined(value.id)){
      value.id = gohan_uuid();
    }
    value.properties.id = value.id;
    var result = gohan_db_create(transaction, value.schema, value.properties);
    return result;
  }
  var id = value.id;
  if(event_type == "post_update_in_transaction" || event_type == "post_update"){
    return gohan_db_update(transaction, value.schema, value.properties);
  }
  if(event_type == "pre_delete_in_transaction" || event_type == "pre_delete"){
    return gohan_db_delete(transaction, value.schema, id);
  }
  return
});

donburi.register_plugin("update", function(event_type, context, value){
  var transaction = context.transaction;
  var resource = gohan_db_fetch(transaction, value.schema, value.properties.id, "");
  if(_.isUndefined(resource)){
    return
  }
  _.each(value.properties, function(value, key){
    resource[key] = value;
  });
  return gohan_db_update(transaction, value.schema, resource);
});

donburi.register_plugin("delete", function(event_type, context, value){
  var transaction = context.transaction;
  return gohan_db_delete(transaction, value.schema, value.id);
});

donburi.register_plugin("contrail", function(event_type, context, value){
    var schema_id = value.schema;
    if(_.isUndefined(contrail_donburi[schema_id])){
      return contrail_donburi["generic"](event_type, context, value);
    }
    return contrail_donburi[schema_id](event_type, context, value);
});

donburi.register_plugin("heat", function(event_type, context, value){
    return heat_donburi["stack"](event_type, context, value);
});

var contrail_donburi = {
  api_url: function(schema_id, id){
    if(_.isUndefined(id)){
      return CONTRAIL_URL + schema_id + "s";
    }
    return CONTRAIL_URL + schema_id + "/" + id;
  },
  create_request: function(schema_id, token, data){
    var self = this;
    var request = {};
    request[schema_id] = data;
    var response = gohan_http("POST", self.api_url(schema_id),
                      {"X-Auth-Token": token,
                       "Content-Type": "application/json"
                      },
                      request)
    var result = {}
    result.status_code = parseInt(response.status_code);
    result.body = response.body;
    if(result.status_code == 200){
      result["data"] = JSON.parse(response.body);
    }
    return result;
  },
  update_request: function(schema_id, token, id, data){
    var self = this;
    var request = {};
    request[schema_id] = data;
    var response = gohan_http("PUT", self.api_url(schema_id, id),
                      {"X-Auth-Token": token,
                       "Content-Type": "application/json"
                      },
                      request)
    var result = {}
    result.status_code = parseInt(response.status_code);
    result.body = response.body;
    if(result.status_code == 200){
      result["data"] = JSON.parse(response.body);
    }
    return result;
  },
  delete_request: function(schema_id, token, id){
    var self = this;
    var response = gohan_http(
      "DELETE",
      self.api_url(schema_id, id),
      {"X-Auth-Token": token}, null);
    var result = {}
    result.status_code = parseInt(response.status_code);
    result.body = response.body;
    return result;
  },
  get_request: function(schema_id, token, id){
    var self = this;
    var response = gohan_http(
      "GET",
      self.api_url(schema_id, id),
      {"X-Auth-Token": token}, null);
    var result = {}
    result.status_code = response.status_code;
    if(response.status_code == "200"){
      result["data"] = JSON.parse(response.body);
    }
    return result;
  },
  generic: function(event_type, context, value){
    var self = this;
    var transaction = context.transaction;
    var schema_id = value.schema;
    var token = context.auth_token;
    var data = value.properties;

    if(event_type == "post_create"){
      return self.create_request(schema_id, token, data);
    }

    var id = value.id;
    if(_.isUndefined(id) || id === ""){
      return;
    }
    if(event_type == "post_update"){
      var update_data = {};
      _.each(value.allow_update, function(key){
        update_data[key] = data[key];
      });
      if(update_data == {}){
          return;
      }
      return self.update_request(schema_id, token, id, update_data);
    }else if(event_type == "pre_delete"){
      return self.delete_request(schema_id, token, id);
    }

    return;
  },
  "virtual-network-subnet": function (event_type, context, value){
    var self = this;
    var token = context.auth_token;
    var network_id = value.network_id;
    var data = value.properties;
    var network_response = self.get_request("virtual-network", token, network_id);
    if( _.isUndefined(network_response.data)){
      return
    }
    var subnet_strings = data.subnet_cidr.split("/");
    data.subnet = {
      ip_prefix: subnet_strings[0],
      ip_prefix_len: parseInt(subnet_strings[1])
    }
    var network = network_response.data["virtual-network"];
    var network_ipam_refs = network["network_ipam_refs"];
    if(_.isUndefined(network_ipam_refs)){
      network_ipam_refs = [
        {
          "attr": {"ipam_subnets": []},
          "to": ["default-domain", "default-project", "default-network-ipam"]
        }];
     network["network_ipam_refs"] = network_ipam_refs;
    }
    var ipam_subnets = network_ipam_refs[0]["attr"]["ipam_subnets"];
    if(event_type == "post_create"){
      ipam_subnets.push(data);
    }
    if(event_type == "post_update"){
      _.each(ipam_subnets, function(subnet){
        if( subnet.subnet_uuid != data.subnet_uuid ){
          return
        }
        _.each(data, function(key){
          subnet[key] = data[key];
        });
      });
    }
    if(event_type == "pre_delete"){
      var new_ipam_subnets = []
      _.each(ipam_subnets, function(subnet){
        if( subnet.subnet_uuid != data.subnet_uuid ){
          new_ipam_subnets.push(subnet);
        }
      });
      network_ipam_refs[0]["attr"]["ipam_subnets"] = new_ipam_subnets;
    }
    var result = self.update_request("virtual-network", token, network_id, network);
    return result;
  }
};

//TODO(nati) This is experimental.
//We need to use output on keystone context

var heat_donburi = {
  api_url: function(url, schema_id, id){
    var base_url = url + "/" + schema_id + "s";
    if(_.isUndefined(id)){
      return base_url;
    }
    return base_url + "/" + id;
  },
  create_request: function(url, schema_id, token, data){
    var self = this;
    var response = gohan_http("POST", self.api_url(url, schema_id),
                      {"X-Auth-Token": token,
                       "Content-Type": "application/json"
                      },
                      data)
    var result = {}
    result.status_code = parseInt(response.status_code);
    result.body = response.body;
    if(result.status_code == 201){
      result["data"] = JSON.parse(response.body);
    }
    return result;
  },
  update_request: function(url, schema_id, token, id, data){
    var self = this;
    var request = {};
    request[schema_id] = data;
    var response = gohan_http("PUT", self.api_url(url, schema_id, id),
                      {"X-Auth-Token": token,
                       "Content-Type": "application/json"
                      },
                      request)
    var result = {}
    result.status_code = parseInt(response.status_code);
    if(result.status_code == 200){
      result["data"] = JSON.parse(response.body);
    }
    return result;
  },
  delete_request: function(url, schema_id, token, id){
    var self = this;
    var response = gohan_http(
      "DELETE",
      self.api_url(url, schema_id, id),
      {"X-Auth-Token": token}, null);
    var result = {}
    result.status_code = parseInt(response.status_code);
    return result;
  },
  get_request: function(url, schema_id, token, id){
    var self = this;
    var response = gohan_http(
      "GET",
      self.api_url(schema_id, id),
      {"X-Auth-Token": token}, null);
    var result = {}
    result.status_code = parseInt(response.status_code);
    if(response.status_code == 200){
      result["data"] = JSON.parse(response.body);
    }
    return result;
  },
  get_endpoint: function(context, endpoint_interface, type){
    var url = "";
    _.each(context.catalog, function(catalog){
      if(catalog.Type === type){
        _.each(catalog.Endpoints, function(endpoint){
          if(endpoint.Interface === endpoint_interface){
             url = endpoint.URL.replace("%(tenant_id)s", context.tenant);
          }
        });
      }
    })
    return url;
  },
  stack: function(event_type, context, value){
    var self = this;
    var transaction = context.transaction;
    var schema_id = "stack";
    var tenant_id = context.tenant;
    var token = context.auth_token;
    var stack_name = value.stack_name;
    var url = self.get_endpoint(context, "public", "orchestration");
    var data = {
      stack_name: stack_name,
      template: value.template,
    };

    if(event_type == "post_create"){
      var response = self.create_request(url, schema_id, token, data);
      return response;
    }

    var id = stack_name + "/" + value.id;
    if(_.isUndefined(id) || id === ""){
      return;
    }
    if(event_type == "post_update"){
      var update_data = {};
      _.each(value.allow_update, function(key){
        update_data[key] = data[key];
      });
      if(update_data == {}){
          return;
      }
      return self.update_request(url, schema_id, token, id, update_data);
    }else if(event_type == "pre_delete"){
      return self.delete_request(url, schema_id, token, id);
    }

    return;
  },
};
