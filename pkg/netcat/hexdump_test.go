package netcat

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestHexDumpWriter(t *testing.T) {
	var dest bytes.Buffer
	var dump bytes.Buffer
	var mu sync.Mutex

	hw := NewHexDumpWriter(&dest, ">>> Sent", &mu)
	hw.SetDumpOutput(&dump)

	data := []byte("Hello, GoConnect!\r\n")
	n, err := hw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("written bytes = %d, want %d", n, len(data))
	}

	// Verify destination buffer received data
	if dest.String() != string(data) {
		t.Errorf("dest = %q, want %q", dest.String(), string(data))
	}

	// Verify dump output format
	dumpStr := dump.String()
	if !strings.Contains(dumpStr, ">>> Sent (19 bytes):") {
		t.Errorf("dump header missing, got:\n%s", dumpStr)
	}
	if !strings.Contains(dumpStr, "48 65 6c 6c 6f 2c 20 47") { // "Hello, G" in hex
		t.Errorf("dump content missing hex, got:\n%s", dumpStr)
	}
	if !strings.Contains(dumpStr, "|Hello, GoConnect|") {
		t.Errorf("dump content missing ASCII, got:\n%s", dumpStr)
	}
}
