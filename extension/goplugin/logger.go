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

package goplugin

import (
	"fmt"

	"github.com/cloudwan/gohan/extension/goext"
	gohan_log "github.com/cloudwan/gohan/log"
)

const logModule = "[GOEXT]"

// Logger is an implementation of ILogger
type Logger struct {
	env *Environment
}

func (logger *Logger) dispatchLog(module string, level goext.Level, format string) {
	log := gohan_log.NewLoggerForModule(module)
	format = fmt.Sprintf("[%s] %s", logger.env.traceID, format)

	switch level {
	case goext.LevelCritical:
		log.Critical(format)
	case goext.LevelError:
		log.Error(format)
	case goext.LevelWarning:
		log.Warning(format)
	case goext.LevelNotice:
		log.Notice(format)
	case goext.LevelInfo:
		log.Info(format)
	case goext.LevelDebug:
		log.Debug(format)
	default:
		panic("Invalid error level")
	}
}

func (logger *Logger) dispatchLogf(module string, level goext.Level, format string, args ...interface{}) {
	log := gohan_log.NewLoggerForModule(module)
	format = fmt.Sprintf("[%s] %s", logger.env.traceID, format)

	switch level {
	case goext.LevelCritical:
		log.Critical(format, args...)
	case goext.LevelError:
		log.Error(format, args...)
	case goext.LevelWarning:
		log.Warning(format, args...)
	case goext.LevelNotice:
		log.Notice(format, args...)
	case goext.LevelInfo:
		log.Info(format, args...)
	case goext.LevelDebug:
		log.Debug(format, args...)
	default:
		panic("Invalid error level")
	}
}

// Critical emits a critical log message
func (logger *Logger) Critical(format string) {
	logger.dispatchLog(logModule, goext.LevelCritical, format)
}

// Criticalf emits a formatted critical log message
func (logger *Logger) Criticalf(format string, args ...interface{}) {
	logger.dispatchLogf(logModule, goext.LevelCritical, format, args...)
}

// Error emits an error log message
func (logger *Logger) Error(format string) {
	logger.dispatchLog(logModule, goext.LevelError, format)
}

// Errorf emits a formatted error log message
func (logger *Logger) Errorf(format string, args ...interface{}) {
	logger.dispatchLogf(logModule, goext.LevelError, format, args...)
}

// Warning emits a warning log message
func (logger *Logger) Warning(format string) {
	logger.dispatchLog(logModule, goext.LevelWarning, format)
}

// Warningf emits a formatted warning log message
func (logger *Logger) Warningf(format string, args ...interface{}) {
	logger.dispatchLogf(logModule, goext.LevelWarning, format, args...)
}

// Notice emits a notice log message
func (logger *Logger) Notice(format string) {
	logger.dispatchLog(logModule, goext.LevelNotice, format)
}

// Noticef emits a formatted notice log message
func (logger *Logger) Noticef(format string, args ...interface{}) {
	logger.dispatchLogf(logModule, goext.LevelNotice, format, args...)
}

// Info emits an info log message
func (logger *Logger) Info(format string) {
	logger.dispatchLog(logModule, goext.LevelInfo, format)
}

// Infof emits a formatted info log message
func (logger *Logger) Infof(format string, args ...interface{}) {
	logger.dispatchLogf(logModule, goext.LevelInfo, format, args...)
}

// Debug emits a debug log message
func (logger *Logger) Debug(format string) {
	logger.dispatchLog(logModule, goext.LevelDebug, format)
}

// Debugf emits a formatted debug log message
func (logger *Logger) Debugf(format string, args ...interface{}) {
	logger.dispatchLogf(logModule, goext.LevelDebug, format, args...)
}

// NewLogger allocates Logger
func NewLogger(env *Environment) *Logger {
	return &Logger{env: env}
}

// Clone allocates a clone of Logger; object may be nil
func (logger *Logger) Clone() *Logger {
	if logger == nil {
		return nil
	}
	return &Logger{
		env: logger.env,
	}
}
