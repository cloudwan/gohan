package buflog

import (
	"bytes"
	"io"
	"os"

	logging "github.com/op/go-logging"
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
func SetUpLogging(output io.Writer) {
	backend := logging.NewLogBackend(output, "", 0)
	format := logging.MustStringFormatter(
		"%{color}%{time:15:04:05.000}: %{module} %{level} %{color:reset} %{message}")
	backendFormatter := logging.NewBackendFormatter(backend, format)
	leveledBackendFormatter := logging.AddModuleLevel(backendFormatter)
	leveledBackendFormatter.SetLevel(logging.INFO, "")
	leveledBackendFormatter.SetLevel(logging.DEBUG, "extest")
	logging.SetBackend(leveledBackendFormatter)
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
