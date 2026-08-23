package proxy

import (
	"crypto/tls"
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
		InsecureSkipVerify: p.config.InsecureSkipVerify,
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
	return doHTTPConnect(tlsConn, address, p.proxyURL.User, timeout, p.config.Verbose)
}
