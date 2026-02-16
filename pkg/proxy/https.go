package proxy

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

// HTTPSProxy implements HTTP CONNECT over TLS (HTTPS proxy).
type HTTPSProxy struct {
	proxyURL *url.URL
	config   Config
}

// NewHTTPSProxy creates a new HTTPS proxy dialer.
func NewHTTPSProxy(proxyURL *url.URL, config Config) *HTTPSProxy {
	return &HTTPSProxy{
		proxyURL: proxyURL,
		config:   config,
	}
}

// Dial connects to the target through the HTTPS proxy.
func (p *HTTPSProxy) Dial(network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	timeout := p.config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	proxyAddr := p.proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		proxyAddr += ":443" // Default HTTPS port
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to HTTPS proxy at %s\n", proxyAddr)
	}

	// First establish TCP connection to proxy
	plainConn, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy %s: %w", proxyAddr, err)
	}

	// Wrap with TLS
	tlsConfig := &tls.Config{
		ServerName:         p.proxyURL.Hostname(),
		InsecureSkipVerify: !p.config.TLSVerify,
	}

	tlsConn := tls.Client(plainConn, tlsConfig)
	if err := tlsConn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = plainConn.Close()
		return nil, err
	}

	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "TLS connection established to proxy\n")
	}

	// Now perform HTTP CONNECT through the TLS connection
	return p.doConnect(tlsConn, address, timeout)
}

// doConnect performs the HTTP CONNECT handshake.
func (p *HTTPSProxy) doConnect(conn net.Conn, address string, timeout time.Duration) (net.Conn, error) {
	targetHost, targetPort, err := net.SplitHostPort(address)
	if err != nil {
		targetHost = address
		targetPort = "443"
	}

	req := fmt.Sprintf("CONNECT %s:%s HTTP/1.1\r\n", targetHost, targetPort)
	req += fmt.Sprintf("Host: %s:%s\r\n", targetHost, targetPort)
	req += "User-Agent: goconnect/1.0\r\n"

	if p.proxyURL.User != nil {
		username := p.proxyURL.User.Username()
		password, _ := p.proxyURL.User.Password()
		auth := basicAuth(username, password)
		req += "Proxy-Authorization: Basic " + auth + "\r\n"
	}

	req += "\r\n"

	if p.config.Verbose {
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

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	response := string(buf[:n])
	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Proxy response: %s\n", strings.Split(response, "\n")[0])
	}

	if !strings.Contains(response, "200") {
		_ = conn.Close()
		return nil, fmt.Errorf("proxy connection failed: %s", strings.Split(response, "\n")[0])
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Tunnel established to %s\n", address)
	}

	// Reset deadlines
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
