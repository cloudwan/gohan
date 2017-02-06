package buflog

import (
	"bytes"
	"io"
	"os"

	l "github.com/cloudwan/gohan/log"
)

// LogBuffer an io.Writer that buffers data and prints it when instructed
type LogBuffer struct {
	*bytes.Buffer
}

var (
	logBuffer = NewBuffer()

	// DefaultLogOutput the default output for test framework logging
	DefaultLogOutput io.Writer = os.Stderr
)

// Buf returns the global LogBuffer
func Buf() *LogBuffer {
	return logBuffer
}

// NewBuffer creates a new LogBuffer
func NewBuffer() (buf *LogBuffer) {
	return &LogBuffer{
		new(bytes.Buffer),
	}
}

// SetUpDefaultLogging sets up logging to the default output for the test framework
func SetUpDefaultLogging() {
	SetUpLogging(DefaultLogOutput)
}

// SetUpLogging sets up logging to output for the test framework
func SetUpLogging(w io.Writer) {
	l.SetUpBasicLogging(w, l.DefaultFormat, "", l.DEBUG, "extest", l.DEBUG)
}

// Activate redirects logging to the buffer
func (buf *LogBuffer) Activate() {
	buf.Reset()
	SetUpLogging(buf)
}

// PrintLogs prints the buffered logs to the default output and clears the buffer
func (buf *LogBuffer) PrintLogs() {
	DefaultLogOutput.Write(buf.Bytes())
	buf.Reset()
}

// Deactivate restores logging to the default output
func (buf *LogBuffer) Deactivate() {
	SetUpDefaultLogging()
}
