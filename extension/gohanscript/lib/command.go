// Copyright (C) 2016  Juniper Networks, Inc.
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

package lib

import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudwan/gohan/util"
)

func init() {
	gohanscript.RegisterStmtParser("command", command)
}

func command(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
	var err error
	stmt.Args, err = gohanscript.MapToValue(util.MaybeMap(stmt.RawData["args"]))
	if err != nil {
		return nil, err
	}
	stmt.Args["command"], err = gohanscript.NewValue(stmt.RawData["command"])
	if err != nil {
		return nil, err
	}
	return func(context *gohanscript.Context) (interface{}, error) {
		chdir := stmt.Arg("chdir", context)
		if chdir != nil {
			currentDir, _ := filepath.Abs(".")
			os.Chdir(util.MaybeString(chdir))
			defer os.Chdir(currentDir)
		}
		command := util.MaybeString(stmt.Arg("command", context))
		parts := strings.Fields(command)
		result, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
		return string(result), err
	}, nil
}
