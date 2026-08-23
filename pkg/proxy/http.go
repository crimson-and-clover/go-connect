package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// HTTPProxy implements HTTP CONNECT proxy support.
type HTTPProxy struct {
	proxyURL *url.URL
	config   Config
}

// NewHTTPProxy creates a new HTTP CONNECT proxy dialer.
func NewHTTPProxy(proxyURL *url.URL, config Config) *HTTPProxy {
	return &HTTPProxy{
		proxyURL: proxyURL,
		config:   config,
	}
}

// Dial connects to the target through the HTTP proxy.
func (p *HTTPProxy) Dial(network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	// Connect to the proxy server
	timeout := p.config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	proxyAddr := p.proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		proxyAddr += ":8080" // Default HTTP proxy port
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to HTTP proxy at %s\n", proxyAddr)
	}

	conn, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy %s: %w", proxyAddr, err)
	}

	return doHTTPConnect(conn, address, p.proxyURL.User, timeout, p.config.Verbose)
}

// doHTTPConnect performs an HTTP CONNECT handshake over the provided connection.
func doHTTPConnect(conn net.Conn, address string, user *url.Userinfo, timeout time.Duration, verbose bool) (net.Conn, error) {
	targetHost, targetPort, err := net.SplitHostPort(address)
	if err != nil {
		targetHost = address
		targetPort = "80"
	}

	req := fmt.Sprintf("CONNECT %s:%s HTTP/1.1\r\n", targetHost, targetPort)
	req += fmt.Sprintf("Host: %s:%s\r\n", targetHost, targetPort)
	req += "User-Agent: goconnect/1.0\r\n"

	if user != nil {
		username := user.Username()
		password, _ := user.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req += "Proxy-Authorization: Basic " + auth + "\r\n"
	}

	req += "\r\n"

	if verbose {
		fmt.Fprintf(os.Stderr, "Sending CONNECT request for %s:%s\n", targetHost, targetPort)
	}

	if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Proxy response: %s\n", resp.Status)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = conn.Close()
		return nil, fmt.Errorf("proxy connection failed: %s", resp.Status)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Tunnel established to %s\n", address)
	}

	// Reset deadline
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return newBufferedConn(conn, reader), nil
}
