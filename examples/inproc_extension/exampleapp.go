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

package main

import (
	"fmt"

	"github.com/cloudwan/gohan/cli"
	"github.com/cloudwan/gohan/extension/inproc"
	"github.com/cloudwan/gohan/extension/otto"
)

//ExampleModule shows example javascript module
type ExampleModule struct {
}

//HelloWorld shows example function
func (example *ExampleModule) HelloWorld(name string, profile map[string]interface{}) {
	fmt.Printf("Hello %s %v\n", name, profile)
}

func main() {
	//Customize code

	//Register inproc callback
	inproc.RegisterHandler("exampleapp_callback",
		func(event string, context map[string]interface{}) error {
			fmt.Printf("callback on %s : %v", event, context)
			return nil
		})

	exampleModule := &ExampleModule{}

	//Register go based module for javascript
	otto.RegisterModule("exampleapp", exampleModule)
	cli.Run("exampleapp", "exampleapp", "0.0.1")
}

