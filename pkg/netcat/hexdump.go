package netcat

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
)

// HexDumpWriter wraps an io.Writer and prints formatted hex dumps to a dump output (default os.Stderr).
type HexDumpWriter struct {
	w       io.Writer
	prefix  string
	dumpOut io.Writer
	mu      *sync.Mutex
}

// NewHexDumpWriter creates a new HexDumpWriter.
func NewHexDumpWriter(w io.Writer, prefix string, mu *sync.Mutex) *HexDumpWriter {
	if mu == nil {
		mu = &sync.Mutex{}
	}
	return &HexDumpWriter{
		w:       w,
		prefix:  prefix,
		dumpOut: os.Stderr,
		mu:      mu,
	}
}

// SetDumpOutput sets a custom writer for the hex dump output (useful for tests).
func (h *HexDumpWriter) SetDumpOutput(out io.Writer) {
	h.dumpOut = out
}

// Write writes data to the underlying writer and outputs the hex dump to dumpOut.
func (h *HexDumpWriter) Write(p []byte) (int, error) {
	n, err := h.w.Write(p)
	if n > 0 {
		h.mu.Lock()
		fmt.Fprintf(h.dumpOut, "\n%s (%d bytes):\n%s", h.prefix, n, hex.Dump(p[:n]))
		h.mu.Unlock()
	}
	return n, err
}
