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

package golang

import (
	"github.com/cloudwan/gohan/extension/goext"
	l "github.com/cloudwan/gohan/log"
)

const logModule = "[goext]"

type Logger struct {
	environment *Environment
}

func (thisLogger *Logger) internalLog(module string, level goext.Level, format string) {
	logger := l.NewLoggerForModule(module)

	switch level {
	case goext.LEVEL_CRITICAL:
		logger.Critical(format)
	case goext.LEVEL_ERROR:
		logger.Error(format)
	case goext.LEVEL_WARNING:
		logger.Warning(format)
	case goext.LEVEL_NOTICE:
		logger.Notice(format)
	case goext.LEVEL_INFO:
		logger.Info(format)
	case goext.LEVEL_DEBUG:
		logger.Debug(format)
	default:
		panic("Invalid error level")
	}
}

func (thisLogger *Logger) internalLogf(module string, level goext.Level, format string, args ...interface{}) {
	logger := l.NewLoggerForModule(module)

	switch level {
	case goext.LEVEL_CRITICAL:
		logger.Critical(format, args...)
	case goext.LEVEL_ERROR:
		logger.Error(format, args...)
	case goext.LEVEL_WARNING:
		logger.Warning(format, args...)
	case goext.LEVEL_NOTICE:
		logger.Notice(format, args...)
	case goext.LEVEL_INFO:
		logger.Info(format, args...)
	case goext.LEVEL_DEBUG:
		logger.Debug(format, args...)
	default:
		panic("Invalid error level")
	}
}

func (thisLogger *Logger) Critical(format string) {
	thisLogger.internalLog(logModule, goext.LEVEL_CRITICAL, format)
}

func (thisLogger *Logger) Criticalf(format string, args ...interface{}) {
	thisLogger.internalLogf(logModule, goext.LEVEL_CRITICAL, format, args...)
}

func (thisLogger *Logger) Error(format string) {
	thisLogger.internalLog(logModule, goext.LEVEL_ERROR, format)
}

func (thisLogger *Logger) Errorf(format string, args ...interface{}) {
	thisLogger.internalLogf(logModule, goext.LEVEL_ERROR, format, args...)
}

func (thisLogger *Logger) Warning(format string) {
	thisLogger.internalLog(logModule, goext.LEVEL_WARNING, format)
}

func (thisLogger *Logger) Warningf(format string, args ...interface{}) {
	thisLogger.internalLogf(logModule, goext.LEVEL_WARNING, format, args...)
}

func (thisLogger *Logger) Notice(format string) {
	thisLogger.internalLog(logModule, goext.LEVEL_NOTICE, format)
}

func (thisLogger *Logger) Noticef(format string, args ...interface{}) {
	thisLogger.internalLogf(logModule, goext.LEVEL_NOTICE, format, args...)
}

func (thisLogger *Logger) Info(format string) {
	thisLogger.internalLog(logModule, goext.LEVEL_INFO, format)
}

func (thisLogger *Logger) Infof(format string, args ...interface{}) {
	thisLogger.internalLogf(logModule, goext.LEVEL_INFO, format, args...)
}

func (thisLogger *Logger) Debug(format string) {
	thisLogger.internalLog(logModule, goext.LEVEL_DEBUG, format)
}

func (thisLogger *Logger) Debugf(format string, args ...interface{}) {
	thisLogger.internalLogf(logModule, goext.LEVEL_DEBUG, format, args...)
}

// NewLogger return a new implementation of a logger interface
func NewLogger(environment *Environment) goext.ILogger {
	return &Logger{environment: environment}
}
