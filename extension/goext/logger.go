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

package goext

// Level represent logger message level
type Level int

const (
	// LevelCritical indicates critical level logger message
	LevelCritical Level = iota

	// LevelError indicates error level logger message
	LevelError

	// LevelWarning indicates warning level logger message
	LevelWarning

	// LevelNotice indicates notice level logger message
	LevelNotice

	// LevelInfo indicates info level logger message
	LevelInfo

	// LevelDebug indicates debug level logger message
	LevelDebug
)

// ILogger is an interface to logger in Gohan
type ILogger interface {
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
