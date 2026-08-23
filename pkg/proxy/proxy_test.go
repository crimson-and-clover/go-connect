package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBufferedConn(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	bufferedData := "pre-buffered content\n"
	buf := bytes.NewBufferString(bufferedData)
	wrapped := newBufferedConn(client, io.MultiReader(buf, client))

	// Write subsequent data to server
	go func() {
		_, _ = server.Write([]byte("live network data\n"))
	}()

	readBuf := make([]byte, 100)
	n1, err := wrapped.Read(readBuf)
	if err != nil {
		t.Fatalf("first Read failed: %v", err)
	}
	if string(readBuf[:n1]) != bufferedData {
		t.Errorf("got %q, want %q", string(readBuf[:n1]), bufferedData)
	}

	n2, err := wrapped.Read(readBuf)
	if err != nil {
		t.Fatalf("second Read failed: %v", err)
	}
	if string(readBuf[:n2]) != "live network data\n" {
		t.Errorf("got %q, want live network data", string(readBuf[:n2]))
	}
}

func TestNewDialer(t *testing.T) {
	cfg := Config{Timeout: 5 * time.Second}

	// Direct dialer
	d1, err := NewDialer("", cfg)
	if err != nil {
		t.Fatalf("expected no error for empty proxy URL, got %v", err)
	}
	if _, ok := d1.(*DirectDialer); !ok {
		t.Errorf("expected *DirectDialer, got %T", d1)
	}

	// HTTP dialer
	d2, err := NewDialer("http://127.0.0.1:8080", cfg)
	if err != nil {
		t.Fatalf("expected no error for http proxy, got %v", err)
	}
	if _, ok := d2.(*HTTPProxy); !ok {
		t.Errorf("expected *HTTPProxy, got %T", d2)
	}

	// HTTPS dialer
	d3, err := NewDialer("https://127.0.0.1:8443", cfg)
	if err != nil {
		t.Fatalf("expected no error for https proxy, got %v", err)
	}
	if _, ok := d3.(*HTTPSProxy); !ok {
		t.Errorf("expected *HTTPSProxy, got %T", d3)
	}

	// SOCKS5 dialer
	d4, err := NewDialer("socks5://127.0.0.1:1080", cfg)
	if err != nil {
		t.Fatalf("expected no error for socks5 proxy, got %v", err)
	}
	if _, ok := d4.(*SOCKS5Proxy); !ok {
		t.Errorf("expected *SOCKS5Proxy, got %T", d4)
	}

	// Unsupported scheme
	_, err = NewDialer("ftp://127.0.0.1:21", cfg)
	if err == nil {
		t.Errorf("expected error for unsupported scheme ftp, got nil")
	}
}

func TestHTTPConnect_ServerSpeaksFirst_NoBufferLoss(t *testing.T) {
	// Mock HTTP Proxy that immediately sends SSH banner after 200 OK
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock proxy listener: %v", err)
	}
	defer ln.Close()

	proxyAddr := ln.Addr().String()
	sshBanner := "SSH-2.0-OpenSSH_9.0\r\n"

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read CONNECT request
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		req := string(buf[:n])
		if !strings.HasPrefix(req, "CONNECT") {
			return
		}

		// Send 200 OK + SSH Banner in one single packet
		response := fmt.Sprintf("HTTP/1.1 200 Connection Established\r\nProxy-Agent: MockProxy\r\n\r\n%s", sshBanner)
		_, _ = conn.Write([]byte(response))

		// Echo anything received back
		_, _ = io.Copy(conn, conn)
	}()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	httpProxy := NewHTTPProxy(proxyURL, Config{Timeout: 2 * time.Second})

	conn, err := httpProxy.Dial("tcp", "example.com:22")
	if err != nil {
		t.Fatalf("httpProxy.Dial failed: %v", err)
	}
	defer conn.Close()

	// Read initial banner from target through proxy
	bannerBuf := make([]byte, len(sshBanner))
	_, err = io.ReadFull(conn, bannerBuf)
	if err != nil {
		t.Fatalf("failed to read banner: %v", err)
	}

	if string(bannerBuf) != sshBanner {
		t.Fatalf("banner mismatch! got %q, want %q", string(bannerBuf), sshBanner)
	}
}
