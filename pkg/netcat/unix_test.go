package netcat

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnixSocketListenerAndDial(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("goconnect_test_%d.sock", time.Now().UnixNano()))
	defer os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Skipf("Unix domain sockets not supported on this platform: %v", err)
		return
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.WriteString(conn, "PONG UNIX\n")
	}()

	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to dial unix socket: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 20)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from unix socket: %v", err)
	}

	if string(buf[:n]) != "PONG UNIX\n" {
		t.Errorf("got %q, want %q", string(buf[:n]), "PONG UNIX\n")
	}
}
