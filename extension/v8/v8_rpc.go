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

import (
	"encoding/json"
	"github.com/ry/v8worker"
	"github.com/twinj/uuid"
	"log"
	"reflect"
)

//RPCMessage is a struct for rpc message.
type RPCMessage struct {
	Context *Context      `json:"context"`
	Object  string        `json:"object"`
	Method  string        `json:"method"`
	Args    []interface{} `json:"args"`
	Reply   interface{}   `json:"reply"`
}

//Context has rpc context information
type Context struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

//RPC represents go side RPC server
type RPC struct {
	worker    *v8worker.Worker
	callbacks map[string]func(interface{})
	objects   map[string]interface{}
	contexts  map[string]map[string]interface{}
}

//NewRPC makes RPC Server
func NewRPC() *RPC {
	rpc := &RPC{
		callbacks: map[string]func(interface{}){},
		objects:   map[string]interface{}{},
		contexts:  map[string]map[string]interface{}{},
	}
	rpc.worker = v8worker.New(rpc.Recv)
	return rpc
}

//Recv recieves message from v8
func (server *RPC) Recv(messageString string) {
	var message RPCMessage
	json.Unmarshal([]byte(messageString), &message)
	server.handleMessage(&message)
}

//Send sends data for v8
func (server *RPC) Send(message *RPCMessage) error {
	bytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("error %s", err)
		return err
	}
	server.worker.Send(string(bytes))
	return nil
}

func (server *RPC) handleMessage(message *RPCMessage) {
	contextType := message.Context.Type
	switch contextType {
	case "call":
		server.handleCall(message)
		return
	case "cast":
		server.handleCast(message)
		return
	case "reply":
		server.handleReply(message)
		return
	}
}

//Invoke calls function by name
func (server *RPC) Invoke(any interface{}, name string, args []interface{}) interface{} {
	inputs := make([]reflect.Value, len(args))
	contextID := args[0].(string)
	context := server.contexts[contextID]
	args[0] = context
	for i, arg := range args {
		inputs[i] = reflect.ValueOf(arg)
	}
	values := reflect.ValueOf(any).MethodByName(name).Call(inputs)
	if len(values) < 1 {
		return nil
	}
	return values[0].Interface()
}

func (server *RPC) handleCast(message *RPCMessage) {
	log.Printf("cast from js: %s", message.Method)
	object, ok := server.objects[message.Object]
	if !ok {
		return
	}
	server.Invoke(object, message.Method, message.Args)
}

func (server *RPC) handleCall(message *RPCMessage) {
	log.Printf("call from js: %s", message.Method)
	object, ok := server.objects[message.Object]
	if !ok {
		return
	}
	message.Context.Type = "reply"
	message.Reply = server.Invoke(object, message.Method, message.Args)
	message.Args = nil
	server.Send(message)
}

func (server *RPC) handleReply(message *RPCMessage) {
	id := message.Context.ID
	callback := server.callbacks[id]
	callback(message.Reply)
	delete(server.callbacks, id)
}

//Cast send cast message for v8
func (server *RPC) Cast(object, method string, args []interface{}) {
	server.Send(&RPCMessage{
		Context: &Context{
			Type: "cast",
		},
		Object: object,
		Method: method,
		Args:   args,
	})
}

//Call send call message for v8
func (server *RPC) Call(object, method string, args []interface{}, callback func(interface{})) {
	id := uuid.NewV4().String()
	server.callbacks[id] = callback
	server.Send(&RPCMessage{
		Context: &Context{
			Type: "call",
			ID:   id,
		},
		Method: method,
		Object: object,
		Args:   args,
	})
}

//Load loads new javascript code
func (server *RPC) Load(source, code string) error {
	return server.worker.Load(source, code)
}

//BlockCall send call message for v8 and block until response
func (server *RPC) BlockCall(object, method string, args []interface{}) interface{} {
	replyChan := make(chan interface{}, 1)
	server.Call(object, method, args, func(reply interface{}) {
		replyChan <- reply
	})
	return <-replyChan
}

//RegistObject registers new object
func (server *RPC) RegistObject(objectID string, object interface{}) {
	server.objects[objectID] = object
}
