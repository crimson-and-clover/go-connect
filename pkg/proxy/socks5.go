package proxy

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// SOCKS5Proxy implements SOCKS5 proxy support using golang.org/x/net/proxy.
type SOCKS5Proxy struct {
	dialer proxy.Dialer
	config Config
}

// NewSOCKS5Proxy creates a new SOCKS5 proxy dialer.
func NewSOCKS5Proxy(proxyURL *url.URL, config Config) (*SOCKS5Proxy, error) {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	proxyAddr := proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		proxyAddr += ":1080" // Default SOCKS5 port
	}

	var auth *proxy.Auth
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     proxyURL.User.Username(),
			Password: password,
		}
	}

	forward := &net.Dialer{Timeout: timeout}
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, forward)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	return &SOCKS5Proxy{
		dialer: dialer,
		config: config,
	}, nil
}

// Dial connects to the target through the SOCKS5 proxy.
func (p *SOCKS5Proxy) Dial(network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("SOCKS5 only supports TCP, got: %s", network)
	}

	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to %s via SOCKS5 proxy\n", address)
	}

	return p.dialer.Dial(network, address)
}
