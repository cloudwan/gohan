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

// +build !v8
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

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/golang"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
)

func (server *Server) newEnvironment(name string) extension.Environment {
	envs := []extension.Environment{}
	for _, extension := range server.extensions {
		switch extension {
		case "javascript":
			envs = append(envs, otto.NewEnvironment(name, server.db, server.keystoneIdentity, server.sync))
		case "gohanscript":
			envs = append(envs, gohanscript.NewEnvironment())
		case "go":
			envs = append(envs, golang.NewEnvironment())
		case "goext":
			env := goplugin.NewEnvironment(name, nil, nil)
			env.SetDatabase(server.db)
			env.SetSync(server.sync)
			envs = append(envs, env)
		}
	}
	return extension.NewEnvironment(envs)
}

// NewEnvironmentForPath creates an extension environment and loads extensions for path
func (server *Server) NewEnvironmentForPath(name string, path string) (env extension.Environment, err error) {
	manager := schema.GetManager()
	env = server.newEnvironment(name)
	err = env.LoadExtensionsForPath(manager.Extensions, manager.TimeLimit, manager.TimeLimits, path)
	if err != nil {
		err = fmt.Errorf("Extensions parsing error: %v", err)
	}
	return
}
