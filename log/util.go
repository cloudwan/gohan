package log

import (
	"fmt"
	"runtime/debug"
)

//Panic logs panic and prevent crash
func Panic(log Logger) {
	err := recover()
	if err != nil {
		log.Error(fmt.Sprintf("Panic %s: %s", err, debug.Stack()))
	}
}

//FatalPanic logs panic and crashes Gohan process
func FatalPanic(log Logger) {
	err := recover()
	if err != nil {
		log.Fatalf("Panic %s: %s", err, debug.Stack())
	}
}
