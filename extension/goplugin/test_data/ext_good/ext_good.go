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
	"context"
	"fmt"
	"time"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
)

// Schemas returns a list of required schemas
func Schemas() []string {
	return []string{
		"../test_schema.yaml",
	}
}

// Init initializes a golang plugin
func Init(env goext.IEnvironment) error {
	testSchema := env.Schemas().Find("test")
	if testSchema == nil {
		return fmt.Errorf("test schema not found")
	}
	testSchema.RegisterRawType(test.Test{})
	testSchema.RegisterEventHandler("wait_for_context_cancel", handleWaitForContextCancel, goext.PriorityDefault)
	testSchema.RegisterEventHandler("echo", handleEcho, goext.PriorityDefault)
	return nil
}

func handleWaitForContextCancel(requestContext goext.Context, _ goext.Resource, _ goext.IEnvironment) error {
	ctx := requestContext["context"].(context.Context)

	select {
	case <-ctx.Done():
		return nil
	case <-time.After(time.Minute):
		return fmt.Errorf("context should be canceled")
	}
}

func handleEcho(requestContext goext.Context, _ goext.Resource, env goext.IEnvironment) error {
	env.Logger().Debug("Handling echo")
	requestContext["response"] = requestContext["input"]
	return nil
}
