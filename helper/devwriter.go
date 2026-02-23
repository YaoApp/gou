package helper

import (
	"io"
	"os"
	"sync"
)

var (
	devWriter   io.Writer = os.Stdout
	devWriterMu sync.RWMutex
)

// SetDevWriter replaces the development mode output target.
// Called by TUI initialization to redirect output into the TUI-managed area.
func SetDevWriter(w io.Writer) {
	devWriterMu.Lock()
	devWriter = w
	devWriterMu.Unlock()
}

// GetDevWriter returns the current development mode output target.
func GetDevWriter() io.Writer {
	devWriterMu.RLock()
	defer devWriterMu.RUnlock()
	return devWriter
}
