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
	"github.com/cloudwan/gohan/extension/goext"
	l "github.com/cloudwan/gohan/log"
	"fmt"
)

const logModule = "[goext]"

// Logger is an implementation of ILogger
type Logger struct {
	environment *Environment
}

func (thisLogger *Logger) dispatchLog(module string, level goext.Level, format string) {
	logger := l.NewLoggerForModule(module)
	format = fmt.Sprintf("[%s] %s", thisLogger.environment.traceID, format)

	switch level {
	case goext.LevelCritical:
		logger.Critical(format)
	case goext.LevelError:
		logger.Error(format)
	case goext.LevelWarning:
		logger.Warning(format)
	case goext.LevelNotice:
		logger.Notice(format)
	case goext.LevelInfo:
		logger.Info(format)
	case goext.LevelDebug:
		logger.Debug(format)
	default:
		panic("Invalid error level")
	}
}

func (thisLogger *Logger) dispatchLogf(module string, level goext.Level, format string, args ...interface{}) {
	logger := l.NewLoggerForModule(module)
	format = fmt.Sprintf("[%s] %s", thisLogger.environment.traceID, format)

	switch level {
	case goext.LevelCritical:
		logger.Critical(format, args...)
	case goext.LevelError:
		logger.Error(format, args...)
	case goext.LevelWarning:
		logger.Warning(format, args...)
	case goext.LevelNotice:
		logger.Notice(format, args...)
	case goext.LevelInfo:
		logger.Info(format, args...)
	case goext.LevelDebug:
		logger.Debug(format, args...)
	default:
		panic("Invalid error level")
	}
}

// Critical emits a critical log message
func (thisLogger *Logger) Critical(format string) {
	thisLogger.dispatchLog(logModule, goext.LevelCritical, format)
}

// Criticalf emits a formatted critical log message
func (thisLogger *Logger) Criticalf(format string, args ...interface{}) {
	thisLogger.dispatchLogf(logModule, goext.LevelCritical, format, args...)
}

// Error emits an error log message
func (thisLogger *Logger) Error(format string) {
	thisLogger.dispatchLog(logModule, goext.LevelError, format)
}

// Errorf emits a formatted error log message
func (thisLogger *Logger) Errorf(format string, args ...interface{}) {
	thisLogger.dispatchLogf(logModule, goext.LevelError, format, args...)
}

// Warning emits a warning log message
func (thisLogger *Logger) Warning(format string) {
	thisLogger.dispatchLog(logModule, goext.LevelWarning, format)
}

// Warningf emits a formatted warning log message
func (thisLogger *Logger) Warningf(format string, args ...interface{}) {
	thisLogger.dispatchLogf(logModule, goext.LevelWarning, format, args...)
}

// Notice emits a notice log message
func (thisLogger *Logger) Notice(format string) {
	thisLogger.dispatchLog(logModule, goext.LevelNotice, format)
}

// Noticef emits a formatted notice log message
func (thisLogger *Logger) Noticef(format string, args ...interface{}) {
	thisLogger.dispatchLogf(logModule, goext.LevelNotice, format, args...)
}

// Info emits an info log message
func (thisLogger *Logger) Info(format string) {
	thisLogger.dispatchLog(logModule, goext.LevelInfo, format)
}

// Infof emits a formatted info log message
func (thisLogger *Logger) Infof(format string, args ...interface{}) {
	thisLogger.dispatchLogf(logModule, goext.LevelInfo, format, args...)
}

// Debug emits a debug log message
func (thisLogger *Logger) Debug(format string) {
	thisLogger.dispatchLog(logModule, goext.LevelDebug, format)
}

// Debugf emits a formatted debug log message
func (thisLogger *Logger) Debugf(format string, args ...interface{}) {
	thisLogger.dispatchLogf(logModule, goext.LevelDebug, format, args...)
}

// NewLogger allocates Logger
func NewLogger(environment *Environment) goext.ILogger {
	return &Logger{environment: environment}
}
