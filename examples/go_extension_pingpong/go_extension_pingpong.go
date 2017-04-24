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
	"github.com/cloudwan/gohan/extension/goext"
)

func Init(env *goext.Environment) error {
	env.Logger.Error("(PP) Initializing GO ext")

	todoSchema := env.Schemas.Find("pingpongGO")
	todoSchema.RegisterResourceType(PingPong{})

	todoSchema.RegisterEventHandler(goext.POST_UPDATE, func(context goext.Context, resource goext.Resource, environment *goext.Environment) error {
		// res := resource.(*PingPong)
		// res.Name += "go;"

		// env.Logger.Errorf("go: %s", res.Name)

		env.Logger.Error("(PP) GO: Trigger ping")

		env.Core.TriggerEvent("ping", context)

		// env.Logger.Errorf("AFTER `ping`, %s", context)

		return nil

	}, goext.PRIORITY_DEFAULT)

	env.Core.RegisterEventHandler("pong", func(context goext.Context, environment *goext.Environment) error {
		env.Logger.Errorf("(PP) GO: Handle pong")

		return nil
	}, goext.PRIORITY_DEFAULT)

	return nil
}
