package log

import (
	"bytes"
	"io"
	"sync"

	"github.com/tylerb/gls"
)

// BufWritter is a thread safe in memory writer, different go routines trees
// write to different buffers.
type BufWritter struct {
}

func (w BufWritter) Write(p []byte) (n int, err error) {
	return buffer().Write(p)
}

// Dump copies buffered bytes to dst.
func (w BufWritter) Dump(dst io.Writer) {
	buffer().WriteTo(dst)
}

// Reset truncates buffer to 0.
func (w BufWritter) Reset() {
	buffer().Reset()
}

var bufferMu sync.Mutex

func buffer() *bytes.Buffer {
	bufferMu.Lock()
	defer bufferMu.Unlock()

	b := gls.Get("log_buffer")
	if b == nil {
		b = bytes.NewBuffer([]byte{})
		gls.Set("log_buffer", b)
	}
	return b.(*bytes.Buffer)
}
