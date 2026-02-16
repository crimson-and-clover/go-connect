package proxy

import (
	"context"
	"net"
	"time"
)

// DirectDialer implements a direct TCP connection without any proxy.
type DirectDialer struct {
	timeout time.Duration
}

// NewDirectDialer creates a new direct dialer with the specified timeout.
func NewDirectDialer(timeout time.Duration) *DirectDialer {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &DirectDialer{timeout: timeout}
}

// Dial connects directly to the target address.
func (d *DirectDialer) Dial(network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	dialer := &net.Dialer{}
	return dialer.DialContext(ctx, network, address)
}
