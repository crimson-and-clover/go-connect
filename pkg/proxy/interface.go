// Package proxy provides proxy dialer implementations for various proxy protocols.
package proxy

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

// Dialer is the common interface for all proxy types.
type Dialer interface {
	// Dial connects to the given address through the proxy.
	Dial(network, address string) (net.Conn, error)
}

// Config holds configuration for creating a dialer.
type Config struct {
	Timeout   time.Duration
	TLSVerify bool
	Verbose   bool
}

// NewDialer creates a Dialer based on the proxy URL.
// Supported schemes: http, https, socks5, socks5h
// If proxyURL is empty, returns a direct dialer.
func NewDialer(proxyURL string, config Config) (Dialer, error) {
	if proxyURL == "" {
		return NewDirectDialer(config.Timeout), nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	switch u.Scheme {
	case "http":
		return NewHTTPProxy(u, config), nil
	case "https":
		return NewHTTPSProxy(u, config), nil
	case "socks5", "socks5h":
		return NewSOCKS5Proxy(u, config)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
}
