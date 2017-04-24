// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

	"github.com/cloudwan/gohan/extension/goext"
)

func handleSchemaEvent(ctx goext.Context, res goext.Resource, env *goext.Environment) error {
	todo := res.(*Todo)
	env.Logger.Warningf("Example log from goext extension (SCHEMA CALLBACK), %v      (ID: %v)", todo, todo.ID)
	updateContextOnEvent(ctx, env)

	return nil
}

func updateContextOnEvent(context goext.Context, env *goext.Environment) error {
	context["example_event_handled"] = true
	return nil
}

func customEventHandler(ctx goext.Context, env *goext.Environment) error {
	env.Logger.Info("Example log from goext extension")

	schemas := env.Schemas.List()
	env.Logger.Infof("Number of schemas: %d", len(schemas))

	for _, s := range schemas {
		env.Logger.Debugf("Found schema: %s", s.ID())
	}

	todoSchema := env.Schemas.Find("todo")

	if todoSchema == nil {
		return fmt.Errorf("schema todo not found")
	}

	// list
	todos := []Todo{}

	if err := todoSchema.List(&todos); err != nil {
		return err
	}

	env.Logger.Infof("Found %d TODO resources", len(todos))

	for _, todo := range todos {
		env.Logger.Warningf("Resource TODO: id = %s, name = %s", todo.ID, todo.Name)
	}

	todoSchema.Delete("1001")

	// create
	if err := todoSchema.Create(&Todo{
		Description: "Code is incorrectly formatted - it must be fixed with gofmt",
		ID:          "1001",
		Name:        "Fix code formatting",
		TenantID:    "admin",
	}); err != nil {
		return err
	}

	// fetch
	todo := Todo{}

	if err := todoSchema.Fetch("1001", &todo); err != nil {
		return err
	}

	// Update
	if err := todoSchema.Update(&Todo{
		Description: "Lines of code are too long - it must be fixed with gofmt (description changed)",
		ID:          "1001",
		Name:        "Fix code formatting",
		TenantID:    "admin",
	}); err != nil {
		return err
	}

	// List
	if err := todoSchema.List(&todos); err != nil {
		return err
	}

	env.Logger.Infof("Found %d TODO resources", len(todos))

	for _, todo := range todos {
		env.Logger.Warningf("Resource TODO: id = %s, name = %s", todo.ID, todo.Name)
	}

	return nil
}

func Init(env *goext.Environment) error {
	// register runtime types for this extension
	todoSchema := env.Schemas.Find("todo")
	todoSchema.RegisterResourceType(Todo{})

	// event handlers
	env.Core.RegisterEventHandler("custom_event", customEventHandler, goext.PRIORITY_DEFAULT)
	env.Core.RegisterEventHandler(goext.POST_UPDATE, updateContextOnEvent, goext.PRIORITY_DEFAULT)

	return nil
}
