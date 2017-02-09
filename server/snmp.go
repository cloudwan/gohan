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

package server

import (
	"fmt"
	"net"

	"github.com/cdevr/WapSNMP"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
)

//SNMP Process
//Experimental
func startSNMPProcess(server *Server) {
	config := util.GetConfig()
	enabled := config.GetParam("snmp", nil)
	if enabled == nil {
		return
	}
	host := config.GetString("snmp/address", "localhost:162")

	path := "snmp://"
	env, err := server.NewEnvironmentForPath("snmp", path)
	if err != nil {
		log.Fatal(err.Error())
	}

	addr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1024)
	go func() {
		defer l.LogFatalPanic(log)
		defer conn.Close()
		for server.running {
			rlen, remote, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Error(fmt.Sprintf("[SNMP] failed read bytes %s", err))
				return
			}
			decoded, err := wapsnmp.DecodeSequence(buf[:rlen])
			if err != nil {
				log.Error(fmt.Sprintf("[SNMP] failed decode bytes %s", err))
				continue
			}
			infos := decoded[3].([]interface{})[4].([]interface{})[1:]
			trap := map[string]string{}
			for _, info := range infos {
				listInfo := info.([]interface{})
				oid := listInfo[1].(wapsnmp.Oid)
				trap[oid.String()] = fmt.Sprintf("%v", listInfo[2])
			}

			context := map[string]interface{}{
				"trap":   trap,
				"remote": remote,
			}
			if err := env.HandleEvent("notification", context); err != nil {
				log.Warning(fmt.Sprintf("extension error: %s", err))
			}
		}
	}()
}

func stopSNMPProcess(server *Server) {

}
