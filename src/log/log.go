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

package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	logging "github.com/op/go-logging"

	"github.com/cloudwan/gohan/util"
)

var defaultLoggerName = "unknown"
var defaultLogLevel = logging.INFO.String()

func mustGohanJSONFormatter(componentName string) *gohanJSONFormatter {
	return &gohanJSONFormatter{componentName: componentName}
}

type gohanJSONFormatter struct {
	componentName string
}

// GetModuleName returns module name
func GetModuleName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return defaultLoggerName
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return defaultLoggerName
	}
	// componentName will be equal to something like:
	// dir_to_gohan/gohan/some_dirs/package_name/(*class_name).func_name
	componentName := f.Name()
	componentName = strings.Replace(componentName, "/", ".", -1)
	nameStart := strings.Index(componentName, "gohan.")
	nameStop := strings.LastIndex(componentName, "(") - 1
	if nameStop < 0 {
		nameStop = strings.LastIndex(componentName, ".")
		if nameStop < 0 {
			nameStop = len(componentName)
		}
	}
	if nameStart < 0 {
		nameStart = 0
	}
	return componentName[nameStart:nameStop]
}

//Format formats message to JSON format
func (f *gohanJSONFormatter) Format(calldepth int, record *logging.Record, output io.Writer) error {
	result := map[string]interface{}{
		"timestamp":      record.Time,
		"log_level":      record.Level.String(),
		"log_type":       "log",
		"msg":            record.Message(),
		"component_name": record.Module,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "%s\n", resultJSON)
	return nil
}

//SetUpLogging configures logging based on configuration
func SetUpLogging(config *util.Config) error {
	var backends = []logging.Backend{}

	if prefix := "logging/file/"; config.GetBool(prefix+"enabled", false) {
		logFile, err := os.OpenFile(config.GetString(prefix+"filename", "gohan.log"),
			os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			return err
		}
		fileBackendLeveled := getLeveledBackend(logFile, mustGohanJSONFormatter("gohan"))
		addLevelsToBackend(config, prefix, fileBackendLeveled)
		backends = append(backends, fileBackendLeveled)
	}

	if prefix := "logging/stderr/"; config.GetBool(prefix+"enabled", true) {
		stringFormatter := logging.MustStringFormatter(
			"%{color}%{time:15:04:05.000} %{module} %{level} %{color:reset} %{message}",
		)
		stderrBackendLeveled := getLeveledBackend(os.Stderr, stringFormatter)
		addLevelsToBackend(config, prefix, stderrBackendLeveled)
		backends = append(backends, stderrBackendLeveled)
	}

	logging.SetBackend(backends...)
	return nil
}

func getLeveledBackend(out io.Writer, formatter logging.Formatter) logging.LeveledBackend {
	backend := logging.NewLogBackend(out, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, formatter)
	return logging.AddModuleLevel(backendFormatter)
}

func addLevelsToBackend(config *util.Config, prefix string, backend logging.LeveledBackend) {
	level, _ := logging.LogLevel(config.GetString(prefix+"level", defaultLogLevel))
	backend.SetLevel(level, "")
	modules := config.GetList(prefix+"modules", []interface{}{})
	for _, rawModule := range modules {
		module, _ := rawModule.(map[string]interface{})
		moduleName, _ := module["name"].(string)
		moduleLevel, _ := module["level"].(string)
		level, err := logging.LogLevel(moduleLevel)
		if moduleName == "" || err != nil {
			continue
		}
		backend.SetLevel(level, moduleName)
	}
}
