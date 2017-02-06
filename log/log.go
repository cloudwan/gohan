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
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/op/go-logging"

	"github.com/cloudwan/gohan/util"
)

// Level defines all available log levels for log messages.
type Level int

// Level values.
const (
	CRITICAL Level = iota
	ERROR
	WARNING
	NOTICE
	INFO
	DEBUG
)

// Logger provides logging capabilities.
type Logger interface {
	// Fatal is equivalent to l.Critical(fmt.Sprint()) followed by a call to os.Exit(1).
	Fatal(args ...interface{})
	// Fatalf is equivalent to l.Critical followed by a call to os.Exit(1).
	Fatalf(format string, args ...interface{})
	// Panic is equivalent to l.Critical(fmt.Sprint()) followed by a call to panic().
	Panic(args ...interface{})
	// Panicf is equivalent to l.Critical followed by a call to panic().
	Panicf(format string, args ...interface{})
	// Critical logs a message using CRITICAL as log level.
	Critical(format string, args ...interface{})
	// Error logs a message using ERROR as log level.
	Error(format string, args ...interface{})
	// Warning logs a message using WARNING as log level.
	Warning(format string, args ...interface{})
	// Notice logs a message using NOTICE as log level.
	Notice(format string, args ...interface{})
	// Info logs a message using INFO as log level.
	Info(format string, args ...interface{})
	// Debug logs a message using DEBUG as log level.
	Debug(format string, args ...interface{})
}

// NewLogger creates new logger for automatically retrieved module name.
func NewLogger() Logger {
	return NewLoggerForModule(getModuleName())
}

// NewLoggerForModule creates new logger for a given module name.
func NewLoggerForModule(module string) Logger {
	return logging.MustGetLogger(module)
}

var (
	defaultLoggerName = "unknown"
	defaultLogLevel   = logging.INFO.String()
)

// getModuleName returns module name.
func getModuleName() string {
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

//SetUpLogging configures logging based on configuration
func SetUpLogging(config *util.Config) error {
	var backends []logging.Backend

	if prefix := "logging/file/"; config.GetBool(prefix+"enabled", false) {
		logFile, err := os.OpenFile(config.GetString(prefix+"filename", "gohan.log"),
			os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			return err
		}
		fileBackendLeveled := getLeveledBackend(logFile, &jsonFormatter{componentName: "gohan"})
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

// Log formats.
var (
	DefaultFormat = "%{color}%{time:15:04:05.000}: %{module} %{level} %{color:reset} %{message}"
	CliFormat     = "%{color}%{message}%{color:reset}"
)

//SetUpBasicLogging configures logging to output logs to w.
func SetUpBasicLogging(w io.Writer, format string, modlevs ...interface{}) {
	if len(modlevs)%2 != 0 {
		panic("Invalid number of parameters")
	}

	backendFormatter := logging.NewBackendFormatter(
		logging.NewLogBackend(w, "", 0),
		logging.MustStringFormatter(format),
	)
	leveledBackendFormatter := logging.AddModuleLevel(backendFormatter)

	for i := 0; i < len(modlevs); i += 2 {
		m := modlevs[i].(string)
		l := modlevs[i+1].(Level)
		leveledBackendFormatter.SetLevel(logging.Level(l), m)
	}

	logging.SetBackend(leveledBackendFormatter)
}
