package netcat

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestUDPCommunication(t *testing.T) {
	// Start local UDP listener
	laddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to resolve local UDP addr: %v", err)
	}

	serverConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		t.Fatalf("failed to start UDP server: %v", err)
	}
	defer func() { _ = serverConn.Close() }()

	serverPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	// UDP Server echo goroutine
	go func() {
		buf := make([]byte, 1024)
		n, clientAddr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if string(buf[:n]) == "PING UDP" {
			_, _ = serverConn.WriteToUDP([]byte("PONG UDP"), clientAddr)
		}
	}()

	// UDP Client
	targetAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("127.0.0.1", strconv.Itoa(serverPort)))
	if err != nil {
		t.Fatalf("failed to resolve target UDP addr: %v", err)
	}

	clientConn, err := net.DialUDP("udp", nil, targetAddr)
	if err != nil {
		t.Fatalf("failed to dial UDP: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	_, err = clientConn.Write([]byte("PING UDP"))
	if err != nil {
		t.Fatalf("failed to write UDP: %v", err)
	}

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	respBuf := make([]byte, 1024)
	n, err := clientConn.Read(respBuf)
	if err != nil {
		t.Fatalf("failed to read UDP response: %v", err)
	}

	if string(respBuf[:n]) != "PONG UDP" {
		t.Errorf("got response %q, want %q", string(respBuf[:n]), "PONG UDP")
	}
}
