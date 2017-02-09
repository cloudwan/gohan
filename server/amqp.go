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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwan/gohan/extension"

	"github.com/streadway/amqp"
	"github.com/twinj/uuid"

	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
)

const connectionWait = 10 * time.Second

//AMQP Process
func startAMQPProcess(server *Server) {
	config := util.GetConfig()
	enableAMQP := config.GetParam("amqp", nil)
	if enableAMQP == nil {
		return
	}
	listenAMQP(server)
}

func listenAMQP(server *Server) {

	hostname, _ := os.Hostname()
	processID := hostname + uuid.NewV4().String()
	config := util.GetConfig()
	connection := config.GetString("amqp/connection", "amqp://guest:guest@127.0.0.1:5672/")
	queues := config.GetStringList("amqp/queues", []string{"notifications.info", "notifications.error"})
	events := config.GetStringList("amqp/events", []string{})
	extensions := map[string]extension.Environment{}
	for _, event := range events {
		path := "amqp://" + event
		env, err := server.NewEnvironmentForPath("amqp."+event, path)
		if err != nil {
			log.Fatal(err.Error())
		}
		extensions[event] = env
	}

	for _, queue := range queues {
		go func(queue string) {
			defer l.LogFatalPanic(log)
			for server.running {
				conn, err := amqp.Dial(connection)
				if err != nil {
					log.Error(fmt.Sprintf("[AMQP] connection error: %s", err))
					time.Sleep(connectionWait)
					continue
				}
				defer conn.Close()

				ch, err := conn.Channel()
				if err != nil {
					log.Error(fmt.Sprintf("[AMQP] channel: %s", err))
					return
				}
				defer ch.Close()
				q, err := ch.QueueDeclare(
					queue, // name
					false, // durable
					false, // delete when usused
					false, // exclusive
					false, // no-wait
					nil,   // arguments
				)
				if err != nil {
					log.Error(fmt.Sprintf("[AMQP] queue declare error: %s", err))
					return
				}

				for server.running {
					msgs, err := ch.Consume(
						q.Name, // queue
						"gohan-"+processID+"-"+queue, // consumer
						true,  // auto-ack
						false, // exclusive
						false, // no-local
						false, // no-wait
						nil,   // args
					)

					if err != nil {
						log.Error(fmt.Sprintf("[AMQP] consume queue error: %s", err))
						break
					}
					for d := range msgs {
						var message map[string]interface{}
						err = json.Unmarshal(d.Body, &message)
						log.Debug(fmt.Sprintf("Received a message: %s %s", queue, d.Body))
						if err != nil {
							log.Error(fmt.Sprintf("[AMQP] json decode error: %s", err))
							continue
						}
						eventType, ok := message["event_type"].(string)
						if !ok {
							log.Error("[AMQP] wrong event type")
							continue
						}
						for _, event := range events {
							if strings.HasPrefix(eventType, event) {
								env := extensions[event]

								context := map[string]interface{}{
									"event": message,
								}
								if err := env.HandleEvent("notification", context); err != nil {
									log.Warning(fmt.Sprintf("extension error: %s", err))
								}
							}
						}
					}
				}
			}
		}(queue)
	}
}

//AMQP Process
func stopAMQPProcess(server *Server) {
}
