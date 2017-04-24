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

package goext

type Level int

const (
	LEVEL_CRITICAL Level = iota
	LEVEL_ERROR
	LEVEL_WARNING
	LEVEL_NOTICE
	LEVEL_INFO
	LEVEL_DEBUG
)

type LoggerInterface interface {
	Generic(module string, level Level, format string)
	Genericf(module string, level Level, format string, args ...interface{})
	Critical(format string)
	Criticalf(format string, args ...interface{})
	Error(format string)
	Errorf(format string, args ...interface{})
	Warning(format string)
	Warningf(format string, args ...interface{})
	Notice(format string)
	Noticef(format string, args ...interface{})
	Info(format string)
	Infof(format string, args ...interface{})
	Debug(format string)
	Debugf(format string, args ...interface{})
}
