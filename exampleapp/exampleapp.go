package main

import (
	"fmt"
	"github.com/cloudwan/gohan/cli"
	"github.com/cloudwan/gohan/extension"
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

	//Register go callback
	extension.RegisterGoCallback("exampleapp_callback",
		func(event string, context map[string]interface{}) error {
			fmt.Printf("callback on %s : %v", event, context)
			return nil
		})

	exampleModule := &ExampleModule{}

	//Register go based module for javascript
	extension.RegisterModule("exampleapp", exampleModule)
	cli.Run("exampleapp", "exampleapp", "0.0.1")
}
