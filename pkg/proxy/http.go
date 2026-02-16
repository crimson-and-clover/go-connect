package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
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

	// Build CONNECT request
	targetHost, targetPort, err := net.SplitHostPort(address)
	if err != nil {
		// Assume the address includes the default port
		targetHost = address
		targetPort = "80"
	}

	req := fmt.Sprintf("CONNECT %s:%s HTTP/1.1\r\n", targetHost, targetPort)
	req += fmt.Sprintf("Host: %s:%s\r\n", targetHost, targetPort)
	req += "User-Agent: goconnect/1.0\r\n"

	// Add authentication if present
	if p.proxyURL.User != nil {
		username := p.proxyURL.User.Username()
		password, _ := p.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req += "Proxy-Authorization: Basic " + auth + "\r\n"
	}

	req += "\r\n"

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Sending CONNECT request for %s:%s\n", targetHost, targetPort)
	}

	// Send the request
	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	// Read the response
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Proxy response: %s", strings.TrimSpace(response))
	}

	// Check for successful response (HTTP/1.1 200 or HTTP/1.0 200)
	if !strings.Contains(response, "200") {
		// Read rest of the response for error details
		var rest string
		for {
			line, err := reader.ReadString('\n')
			if err != nil || line == "\r\n" || line == "\n" {
				break
			}
			rest += line
		}
		conn.Close()
		return nil, fmt.Errorf("proxy connection failed: %s %s", strings.TrimSpace(response), strings.TrimSpace(rest))
	}

	// Consume the rest of the headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("error reading response headers: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Tunnel established to %s\n", address)
	}

	// Reset deadline
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}
