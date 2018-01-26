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
	"time"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin/test_data/ext_good/test"
	"github.com/pkg/errors"
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
		return errors.New("test schema not found")
	}
	testSchema.RegisterRawType(test.Test{})
	testSchema.RegisterCustomEventHandler("wait_for_context_cancel", handleWaitForContextCancel, goext.PriorityDefault)
	testSchema.RegisterCustomEventHandler("echo", handleEcho, goext.PriorityDefault)
	testSchema.RegisterCustomEventHandler("invoke_js", handleInvokeJs, goext.PriorityDefault)
	testSchema.RegisterResourceEventHandler("pre_create", handlePreCreate, goext.PriorityDefault)

	testSuiteSchema := env.Schemas().Find("test_suite")
	if testSuiteSchema == nil {
		return errors.New("test suite schema not found")
	}
	testSuiteSchema.RegisterRawType(test.TestSuite{})
	return nil
}

func handleWaitForContextCancel(requestContext goext.Context, _ goext.IEnvironment) *goext.Error {
	ctx, _ := requestContext.GetContext()

	select {
	case <-ctx.Done():
		return nil
	case <-time.After(time.Minute):
		return goext.NewErrorInternalServerError(errors.New("context should be canceled"))
	}

	panic("test extension: something went terribly wrong")
}

func handleEcho(requestContext goext.Context, env goext.IEnvironment) *goext.Error {
	env.Logger().Debug("Handling echo")
	typedContext := requestContext.(goext.GohanContext)
	typedContext["response"] = typedContext["input"]
	return nil
}

func handleInvokeJs(requestContext goext.Context, env goext.IEnvironment) *goext.Error {
	env.Logger().Debug("Handling invoke JS")

	ctx := requestContext.Clone().(goext.GohanContext)
	ctx.SetSchemaID("test")
	env.Core().TriggerEvent("js_listener", ctx)

	requestContext.SetResponse(ctx["js_result"])
	return nil
}

func handlePreCreate(requestContext goext.Context, _ goext.Resource, env goext.IEnvironment) *goext.Error {
	env.Logger().Debug("Handling pre create")
	return nil
}
