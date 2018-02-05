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

// Logger is an implementation of ILogger
type Logger struct {
	traceId string
	inner   gohan_log.Logger
}

var _ goext.ILogger = &Logger{}

func (l *Logger) dispatchLog(level goext.Level, msg string) {
	msg = fmt.Sprintf("[%s] %s", l.traceId, msg)

	switch level {
	case goext.LevelCritical:
		l.inner.Critical(msg)
	case goext.LevelError:
		l.inner.Error(msg)
	case goext.LevelWarning:
		l.inner.Warning(msg)
	case goext.LevelNotice:
		l.inner.Notice(msg)
	case goext.LevelInfo:
		l.inner.Info(msg)
	case goext.LevelDebug:
		l.inner.Debug(msg)
	default:
		panic("Invalid error level")
	}
}

func (l *Logger) dispatchLogf(level goext.Level, format string, args ...interface{}) {
	format = fmt.Sprintf("[%s] %s", l.traceId, format)

	switch level {
	case goext.LevelCritical:
		l.inner.Critical(format, args...)
	case goext.LevelError:
		l.inner.Error(format, args...)
	case goext.LevelWarning:
		l.inner.Warning(format, args...)
	case goext.LevelNotice:
		l.inner.Notice(format, args...)
	case goext.LevelInfo:
		l.inner.Info(format, args...)
	case goext.LevelDebug:
		l.inner.Debug(format, args...)
	default:
		panic("Invalid error level")
	}
}

// Critical emits a critical log message
func (l *Logger) Critical(format string) {
	l.dispatchLog(goext.LevelCritical, format)
}

// Criticalf emits a formatted critical log message
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.dispatchLogf(goext.LevelCritical, format, args...)
}

// Error emits an error log message
func (l *Logger) Error(format string) {
	l.dispatchLog(goext.LevelError, format)
}

// Errorf emits a formatted error log message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.dispatchLogf(goext.LevelError, format, args...)
}

// Warning emits a warning log message
func (l *Logger) Warning(format string) {
	l.dispatchLog(goext.LevelWarning, format)
}

// Warningf emits a formatted warning log message
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.dispatchLogf(goext.LevelWarning, format, args...)
}

// Notice emits a notice log message
func (l *Logger) Notice(format string) {
	l.dispatchLog(goext.LevelNotice, format)
}

// Noticef emits a formatted notice log message
func (l *Logger) Noticef(format string, args ...interface{}) {
	l.dispatchLogf(goext.LevelNotice, format, args...)
}

// Info emits an info log message
func (l *Logger) Info(format string) {
	l.dispatchLog(goext.LevelInfo, format)
}

// Infof emits a formatted info log message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.dispatchLogf(goext.LevelInfo, format, args...)
}

// Debug emits a debug log message
func (l *Logger) Debug(format string) {
	l.dispatchLog(goext.LevelDebug, format)
}

// Debugf emits a formatted debug log message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.dispatchLogf(goext.LevelDebug, format, args...)
}

// NewLogger allocates Logger
func NewLogger(env IEnvironment) *Logger {
	return newLogger(env.getTraceID(), env.getName())
}

func NewLoggerForSchema(schema *Schema) *Logger {
	return newLogger(schema.env.getTraceID(), schema.ID())
}

func newLogger(traceId string, name string) *Logger {
	return &Logger{traceId, gohan_log.NewLoggerForModule(prefixed(name))}
}

func prefixed(suffix string) string {
	return fmt.Sprintf("goplugin.%s", suffix)
}

// Clone allocates a clone of Logger; object may be nil
func (l *Logger) Clone(env *Environment) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{env.getTraceID(), l.inner}
}
