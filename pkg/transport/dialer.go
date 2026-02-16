// Package transport provides low-level transport implementations.
package transport

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

	return (&net.Dialer{}).DialContext(ctx, network, address)
}

// TimeoutDialer wraps a connection with read/write timeouts.
type TimeoutDialer struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// NewTimeoutDialer wraps a connection with timeout support.
func NewTimeoutDialer(conn net.Conn, readTimeout, writeTimeout time.Duration) *TimeoutDialer {
	if readTimeout == 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}
	return &TimeoutDialer{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// Read implements io.Reader with timeout.
func (t *TimeoutDialer) Read(p []byte) (int, error) {
	if err := t.SetReadDeadline(time.Now().Add(t.readTimeout)); err != nil {
		return 0, err
	}
	return t.Conn.Read(p)
}

// Write implements io.Writer with timeout.
func (t *TimeoutDialer) Write(p []byte) (int, error) {
	if err := t.SetWriteDeadline(time.Now().Add(t.writeTimeout)); err != nil {
		return 0, err
	}
	return t.Conn.Write(p)
}
