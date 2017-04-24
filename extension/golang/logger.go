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

package golang

import (
	"github.com/cloudwan/gohan/extension/goext"
	l "github.com/cloudwan/gohan/log"
)

const logModule = "[goext]"

type loggerBinder struct {
	rawEnvironment *Environment
}

func (thisLoggerBinder *loggerBinder) internalLog(module string, level goext.Level, format string) {
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

func (thisLoggerBinder *loggerBinder) internalLogf(module string, level goext.Level, format string, args ...interface{}) {
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

func (thisLoggerBinder *loggerBinder) Generic(module string, level goext.Level, format string) {
	thisLoggerBinder.internalLog(module, level, format)
}

func (thisLoggerBinder *loggerBinder) Genericf(module string, level goext.Level, format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(module, level, format, args...)
}

func (thisLoggerBinder *loggerBinder) Critical(format string) {
	thisLoggerBinder.internalLog(logModule, goext.LEVEL_CRITICAL, format)
}

func (thisLoggerBinder *loggerBinder) Criticalf(format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(logModule, goext.LEVEL_CRITICAL, format, args...)
}

func (thisLoggerBinder *loggerBinder) Error(format string) {
	thisLoggerBinder.internalLog(logModule, goext.LEVEL_ERROR, format)
}

func (thisLoggerBinder *loggerBinder) Errorf(format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(logModule, goext.LEVEL_ERROR, format, args...)
}

func (thisLoggerBinder *loggerBinder) Warning(format string) {
	thisLoggerBinder.internalLog(logModule, goext.LEVEL_WARNING, format)
}

func (thisLoggerBinder *loggerBinder) Warningf(format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(logModule, goext.LEVEL_WARNING, format, args...)
}

func (thisLoggerBinder *loggerBinder) Notice(format string) {
	thisLoggerBinder.internalLog(logModule, goext.LEVEL_NOTICE, format)
}

func (thisLoggerBinder *loggerBinder) Noticef(format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(logModule, goext.LEVEL_NOTICE, format, args...)
}

func (thisLoggerBinder *loggerBinder) Info(format string) {
	thisLoggerBinder.internalLog(logModule, goext.LEVEL_INFO, format)
}

func (thisLoggerBinder *loggerBinder) Infof(format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(logModule, goext.LEVEL_INFO, format, args...)
}

func (thisLoggerBinder *loggerBinder) Debug(format string) {
	thisLoggerBinder.internalLog(logModule, goext.LEVEL_DEBUG, format)
}

func (thisLoggerBinder *loggerBinder) Debugf(format string, args ...interface{}) {
	thisLoggerBinder.internalLogf(logModule, goext.LEVEL_DEBUG, format, args...)
}

func bindLogger(rawEnvironment *Environment) goext.LoggerInterface {
	return &loggerBinder{rawEnvironment: rawEnvironment}
}
