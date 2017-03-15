// Copyright (C) 2016 NTT Innovation Institute, Inc.
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

package otto

import (
	"github.com/xyproto/otto"
	//Import otto underscore lib
	_ "github.com/xyproto/otto/underscore"

	l "github.com/cloudwan/gohan/log"
)

const (
	// e.g. {Log level} must be {an int}: "ERROR"
	wrongTypeErrorMessageFormat = "%s must be %s: %v"
)

func init() {
	gohanLoggingInit := func(env *Environment) {
		vm := env.VM

		builtins := map[string]interface{}{
			"gohan_log_impl": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_log_impl", 4)

				// TODO:
				// Taking this as an argument is a workaround
				// for Otto returning stale values of variables
				// that have been changed in javascript.
				// We can get this from LOG_MODULE javascript
				// variable if we fix the problem.
				module, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, "Log module: %v", err)
				}
				logger := l.NewLoggerForModule(module)

				intLevel, err := GetInt64(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, "Log level: %v", err)
				}
				level := l.Level(intLevel)

				caller, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, "Caller: %v", err)
				}

				message, err := GetString(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, "Message: %v", err)
				}

				// if caller is non-empty, add extra information about the calling handler
				if caller != "" {
					logGeneral(logger, level, "[%s] %s", caller, message)
				} else {
					// otherwise, do not put empty information about the caller
					logGeneral(logger, level, message)
				}

				return otto.Value{}
			},
		}
		for name, object := range builtins {
			vm.Set(name, object)
		}

		// op/go-logging/level.go has levelNames[], but it's unexported
		logLevels := map[string]l.Level{
			"CRITICAL": l.CRITICAL,
			"ERROR":    l.ERROR,
			"WARNING":  l.WARNING,
			"NOTICE":   l.NOTICE,
			"INFO":     l.INFO,
			"DEBUG":    l.DEBUG,
		}
		vm.Set("LOG_LEVEL", logLevels)

		vm.Set("LOG_MODULE", "gohan.extension."+env.Name)

		err := env.Load("<Gohan logging built-ins>", `
		function gohan_log_module_push(new_module){
		    var old_module = LOG_MODULE;
		    LOG_MODULE += "." + new_module;
		    return old_module;
		}

		function gohan_log_module_restore(old_module){
		    LOG_MODULE = old_module;
		}

		function gohan_log(module, level, msg) {
		    gohan_log_impl(module, level, gohan_caller, msg);
		}

		function gohan_log_critical(msg) {
		    gohan_log(LOG_MODULE, LOG_LEVEL.CRITICAL, msg);
		}

		function gohan_log_error(msg) {
		    gohan_log(LOG_MODULE, LOG_LEVEL.ERROR, msg);
		}

		function gohan_log_warning(msg) {
		    gohan_log(LOG_MODULE, LOG_LEVEL.WARNING, msg);
		}

		function gohan_log_notice(msg) {
		    gohan_log(LOG_MODULE, LOG_LEVEL.NOTICE, msg);
		}

		function gohan_log_info(msg) {
		    gohan_log(LOG_MODULE, LOG_LEVEL.INFO, msg);
		}

		function gohan_log_debug(msg) {
		    gohan_log(LOG_MODULE, LOG_LEVEL.DEBUG, msg);
		}
		`)
		if err != nil {
			log.Fatal(err)
		}
	}
	RegisterInit(gohanLoggingInit)
}

// logGeneral can be replaced with logger.Log(level, format, args) when https://github.com/op/go-logging/issues/80 gets fixed.
func logGeneral(logger l.Logger, level l.Level, format string, args ...interface{}) {
	var logAction func(format string, args ...interface{})
	switch level {
	case l.CRITICAL:
		logAction = logger.Critical
	case l.ERROR:
		logAction = logger.Error
	case l.WARNING:
		logAction = logger.Warning
	case l.NOTICE:
		logAction = logger.Notice
	case l.INFO:
		logAction = logger.Info
	case l.DEBUG:
		logAction = logger.Debug
	}

	logAction(format, args...)
}

// PushJSLogModule appends newModule to log module in env, returns a function that restores the original value
func PushJSLogModule(env *Environment, newModule string) (restore func()) {
	newModuleInVM, _ := env.VM.ToValue(newModule)
	oldModule := pushJSLogModule(env, newModuleInVM)
	return func() {
		restoreJSLogModule(env, oldModule)
	}
}

func restoreJSLogModule(env *Environment, oldModule otto.Value) {
	_, err := env.VM.Call("gohan_log_module_restore", nil, oldModule)
	if err != nil {
		log.Error("Calling gohan_log_module_restore: " + err.Error())
	}
}

func pushJSLogModule(env *Environment, newModule otto.Value) (oldModule otto.Value) {
	oldModule, err := env.VM.Call("gohan_log_module_push", nil, newModule)
	if err != nil {
		log.Error("Calling gohan_log_module_push: " + err.Error())
	}
	return
}
