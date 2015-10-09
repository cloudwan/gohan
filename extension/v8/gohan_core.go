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

// +build v8
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

package v8

const gohanBuildin = `
var v8rpc = {
  callbacks: [],
  objects: [],
  call_count: 0,
  init: function(){
    var self = this;
    $recv(function(messageString){
        var message = JSON.parse(messageString)
        var context = message.context
        if(context.type === "reply"){
          var callback = self.callbacks[context.id];
          callback(message.reply);
          delete self.callbacks[context.id];
        }else if(context.type == "cast"){
          $print("cast from go: " + message.method);
          var object = self.objects[message.object];
          if(object){
            object[message.method].apply(object, message.args)
          }
        }else if(context.type == "call"){
          $print("call from go: " + message.method);
          var object = self.objects[message.object];
          if(object){
            reply = object[message.method].apply(object, message.args)
          }
          context.type = "reply";
          message.reply = reply;
          self.send(message)
        }
    });
  },
  newCallbackID: function(){
    return "" + this.call_count++;
  },
  send: function(object){
    var messageString = JSON.stringify(object);
    $send(messageString);
  },
  cast: function(object, method, args){
      this.send({"context":{"type": "cast"}, "object": object, "args": args, "method": method});
  },
  call: function(object, method, args, callback){
    var callbackID = this.newCallbackID();
    this.callbacks[callbackID] = callback;
    var context = {"id": callbackID, "type": "call"}
    this.send({"context": context, "args": args, "object": object, "method": method});
  },
  register_object: function(objectID, object){
      this.objects[objectID] = object;
  }
}

v8rpc.init();

var gohan_handler = {}

function gohan_register_handler(event_type, func){
  if(gohan_handler[event_type] == undefined){
    gohan_handler[event_type] = [];
  }
  gohan_handler[event_type].push(func)
}

var gohan_handler = {
  handle_event: function(context, event_type){
     if(gohan_handler[event_type] == undefined){
       return
     }
     for (var i = 0; i < gohan_handler[event_type].length; ++i) {
       gohan_handler[event_type][i](context);
     }
     return context
  }
};

v8rpc.register_object("gohan_handler", gohan_handler);

`

func init() {
	gohanInit := func(env *Environment) {
		env.rpc.Load("<Gohan built-ins>", gohanBuildin)
	}
	RegisterInit(gohanInit)
}
