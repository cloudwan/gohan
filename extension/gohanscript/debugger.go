// Copyright (C) 2016  Juniper Networks, Inc.
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

package gohanscript

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"

	"gopkg.in/yaml.v2"
)

const debugPortBase = 40000
const debugPortMax = 50000
const helpMessage = "s: step, n: next, r: return, c: continue, p: print context, l: print current line\n"

//DebuggerRPC is a proecess to handle remote debuggging
type DebuggerRPC struct {
	inputCh  chan string
	outputCh chan string
	server   *net.TCPListener
	conn     net.Conn
}

//Command accepts debug command
func (d *DebuggerRPC) Command(param []byte, ack *string) error {
	d.inputCh <- string(param)
	*ack = <-d.outputCh
	return nil
}

func (d *DebuggerRPC) waitInput() string {
	input := <-d.inputCh
	input = strings.TrimRight(input, "\n")
	return strings.TrimRight(input, "\r")
}

func (d *DebuggerRPC) output(message string) {
	d.conn.Write([]byte(message))
}

func (d *DebuggerRPC) start() {
	var addy *net.TCPAddr
	for port := debugPortBase; port < debugPortMax; port++ {
		addy, _ = net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
		listener, err := net.ListenTCP("tcp", addy)
		if err != nil {
			continue
		}
		log.Info("Debugger port: telnet localhost %d", port)

		conn, err := listener.Accept()
		if conn == nil {
			log.Warning("failed to accept debugger connection: %s", err)
			return
		}
		d.conn = conn
		go func() {
			b := bufio.NewReader(conn)
			for {
				line, err := b.ReadString('\n')
				if err != nil { // EOF, or worse
					break
				}
				d.inputCh <- line
			}
		}()
		return
	}
}

func (d *DebuggerRPC) close() {
	d.conn.Close()
	d.server.Close()
}

func newDebugger() *DebuggerRPC {
	d := &DebuggerRPC{
		inputCh:  make(chan string),
		outputCh: make(chan string),
	}
	d.start()
	return d
}

func printValue(data interface{}) string {
	switch d := data.(type) {
	case string:
		return d
	case map[string]interface{}:
		var buffer bytes.Buffer
		for key, value := range d {
			buffer.WriteString(fmt.Sprintf("%s : %v \n", key, value))
		}
		return buffer.String()
	case []interface{}:
		var buffer bytes.Buffer
		for key, value := range d {
			buffer.WriteString(fmt.Sprintf("%d : %v \n", key, value))
		}
		return buffer.String()
	default:
		return fmt.Sprintf("%v\n", d)
	}
}

func debugWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (value interface{}, err error) {
		debugNext := false
		vm := context.VM
		if vm.debug {
			if vm.debuggerRPC == nil {
				vm.debuggerRPC = newDebugger()
			}
		DEBUG_LOOP:
			for {
				currentLine := fmt.Sprintf("%s:%d %s > ", stmt.File, stmt.Line, stmt.Name)
				vm.debuggerRPC.output(currentLine)
				input := vm.debuggerRPC.waitInput()
				commands := strings.SplitN(input, " ", 2)
				switch commands[0] {
				case "s":
					vm.debuggerRPC.output("")
					break DEBUG_LOOP
				case "n":
					vm.debuggerRPC.output("")
					debugNext = true
					vm.debug = false
					break DEBUG_LOOP
				case "r":
					vm.debuggerRPC.output("")
					vm.debugReturn = true
					vm.debug = false
					break DEBUG_LOOP
				case "c":
					vm.debuggerRPC.output("")
					vm.debug = false
					break DEBUG_LOOP
				case "p":
					if len(commands) < 2 {
						vm.debuggerRPC.output(printValue(context.data))
						continue
					}
					minigo, err := CompileExpr(commands[1])
					if err != nil {
						vm.debuggerRPC.output(err.Error())
						continue
					}
					result, err := minigo.Run(context)
					if err != nil {
						vm.debuggerRPC.output(err.Error())
					}
					vm.debuggerRPC.output(printValue(result))
				case "C":
					vm.debuggerRPC.output(currentLine)
				case "l":
					yamlCode, _ := yaml.Marshal(&stmt.RawData)
					vm.debuggerRPC.output(string(yamlCode))
				default:
					vm.debuggerRPC.output(currentLine + "\n" + helpMessage)
				}
			}
		}
		value, err = f(context)
		if debugNext {
			vm.debug = true
		}
		return
	}, nil
}
