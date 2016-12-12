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

package otto

import (
	"bytes"

	"github.com/robertkrimen/otto"
	"golang.org/x/crypto/ssh"

	"github.com/Juniper/go-netconf/netconf"
	"github.com/cloudwan/gohan/util"
	//Import otto underscore lib
	_ "github.com/robertkrimen/otto/underscore"
)

//SetUp sets up vm to with environment
func init() {
	gohanRemoteInit := func(env *Environment) {
		vm := env.VM

		builtins := map[string]interface{}{
			"gohan_netconf_open": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_netconf_open call.")
				}
				rawHost, _ := call.Argument(0).Export()
				host, ok := rawHost.(string)
				if !ok {
					return otto.NullValue()
				}
				rawUserName, _ := call.Argument(1).Export()
				userName, ok := rawUserName.(string)
				if !ok {
					return otto.NullValue()
				}
				config := util.GetConfig()
				publicKeyFile := config.GetString("ssh/key_file", "")
				if publicKeyFile == "" {
					return otto.NullValue()
				}
				sshConfig := &ssh.ClientConfig{
					User: userName,
					Auth: []ssh.AuthMethod{
						util.PublicKeyFile(publicKeyFile),
					},
				}
				s, err := netconf.DialSSH(host, sshConfig)

				if err != nil {
					ThrowOttoException(&call, "Error during gohan_netconf_open: %s", err.Error())
				}
				value, _ := vm.ToValue(s)
				return value
			},
			"gohan_netconf_close": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 1 {
					panic("Wrong number of arguments in gohan_netconf_close call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*netconf.Session)
				if !ok {
					ThrowOttoException(&call, "Error during gohan_netconf_close")
				}
				s.Close()
				return otto.NullValue()
			},
			"gohan_netconf_exec": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_netconf_exec call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*netconf.Session)
				if !ok {
					return otto.NullValue()
				}
				rawCommand, _ := call.Argument(1).Export()
				command, ok := rawCommand.(string)
				if !ok {
					return otto.NullValue()
				}
				reply, err := s.Exec(netconf.RawMethod(command))
				resp := map[string]interface{}{}
				if err != nil {
					resp["status"] = "error"
					resp["output"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["output"] = reply
				}
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_ssh_open": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_ssh_open call.")
				}
				rawHost, _ := call.Argument(0).Export()
				host, ok := rawHost.(string)
				if !ok {
					return otto.NullValue()
				}
				rawUserName, _ := call.Argument(1).Export()
				userName, ok := rawUserName.(string)
				if !ok {
					return otto.NullValue()
				}
				config := util.GetConfig()
				publicKeyFile := config.GetString("ssh/key_file", "")
				if publicKeyFile == "" {
					return otto.NullValue()
				}
				sshConfig := &ssh.ClientConfig{
					User: userName,
					Auth: []ssh.AuthMethod{
						util.PublicKeyFile(publicKeyFile),
					},
				}
				conn, err := ssh.Dial("tcp", host, sshConfig)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_ssh_open %s", err)
				}
				session, err := conn.NewSession()
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_ssh_open %s", err)
				}
				value, _ := vm.ToValue(session)
				return value
			},
			"gohan_ssh_close": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 1 {
					panic("Wrong number of arguments in gohan_ssh_close call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*ssh.Session)
				if !ok {
					ThrowOttoException(&call, "Error during gohan_ssh_close")
				}
				s.Close()
				return otto.NullValue()
			},
			"gohan_ssh_exec": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_ssh_exec call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*ssh.Session)
				if !ok {
					return otto.NullValue()
				}
				rawCommand, _ := call.Argument(1).Export()
				command, ok := rawCommand.(string)
				if !ok {
					return otto.NullValue()
				}
				var stdoutBuf bytes.Buffer
				s.Stdout = &stdoutBuf
				err := s.Run(command)
				resp := map[string]interface{}{}
				if err != nil {
					resp["status"] = "error"
					resp["output"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["output"] = stdoutBuf.String()
				}
				value, _ := vm.ToValue(resp)
				return value
			},
		}
		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanRemoteInit)
}
