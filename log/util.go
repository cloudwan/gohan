package log

import (
	"fmt"
	"runtime/debug"
)

//LogPanic logs panic and prevent crash
func LogPanic(log Logger) {
	err := recover()
	if err != nil {
		log.Error(fmt.Sprintf("Panic %s: %s", err, debug.Stack()))
	}
}

//LogFatalPanic logs panic and crashes Gohan process
func LogFatalPanic(log Logger) {
	err := recover()
	if err != nil {
		log.Fatalf("Panic %s: %s", err, debug.Stack())
	}
}
