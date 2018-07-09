package log

import (
	"fmt"
)

type tracingLogger struct {
	inner   Logger
	traceId string
}

func (logger *tracingLogger) dispatchLogf(level Level, format string, args ...interface{}) {
	format = fmt.Sprintf("[%s] %s", logger.traceId, format)

	switch level {
	case PANIC:
		logger.inner.Panic(format)
	case CRITICAL:
		logger.inner.Critical(format, args...)
	case ERROR:
		logger.inner.Error(format, args...)
	case WARNING:
		logger.inner.Warning(format, args...)
	case NOTICE:
		logger.inner.Notice(format, args...)
	case INFO:
		logger.inner.Info(format, args...)
	case DEBUG:
		logger.inner.Debug(format, args...)
	default:
		panic("Invalid error level")
	}
}

func (logger *tracingLogger) dispatchLog(level Level, args ...interface{}) {
	args = append(args, logger.traceId)

	switch level {
	case PANIC:
		logger.inner.Panic(args...)
	case FATAL:
		logger.inner.Fatal(args...)
	default:
		panic("Invalid error level")
	}
}

func (logger *tracingLogger) Panicf(format string, args ...interface{}) {
	logger.dispatchLogf(PANIC, format, args...)
}

func (logger *tracingLogger) Panic(args ...interface{}) {
	logger.dispatchLog(PANIC, args...)
}

func (logger *tracingLogger) Fatalf(format string, args ...interface{}) {
	logger.dispatchLogf(FATAL, format, args...)
}

func (logger *tracingLogger) Fatal(args ...interface{}) {
	logger.dispatchLog(FATAL, args...)
}

// Criticalf emits a formatted critical log message
func (logger *tracingLogger) Critical(format string, args ...interface{}) {
	logger.dispatchLogf(CRITICAL, format, args...)
}

// Errorf emits a formatted error log message
func (logger *tracingLogger) Error(format string, args ...interface{}) {
	logger.dispatchLogf(ERROR, format, args...)
}

// Warningf emits a formatted warning log message
func (logger *tracingLogger) Warning(format string, args ...interface{}) {
	logger.dispatchLogf(WARNING, format, args...)
}

// Noticef emits a formatted notice log message
func (logger *tracingLogger) Notice(format string, args ...interface{}) {
	logger.dispatchLogf(NOTICE, format, args...)
}

// Infof emits a formatted info log message
func (logger *tracingLogger) Info(format string, args ...interface{}) {
	logger.dispatchLogf(INFO, format, args...)
}

// Debugf emits a formatted debug log message
func (logger *tracingLogger) Debug(format string, args ...interface{}) {
	logger.dispatchLogf(DEBUG, format, args...)
}
