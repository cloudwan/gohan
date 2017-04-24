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
	return nil
}
