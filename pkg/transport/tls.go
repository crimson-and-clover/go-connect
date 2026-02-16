// Package transport provides low-level transport implementations.
package transport

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"
)

// TLSWrapper wraps a connection with TLS.
type TLSWrapper struct {
	serverName string
	skipVerify bool
	verbose    bool
}

// NewTLSWrapper creates a new TLS wrapper.
func NewTLSWrapper(serverName string, skipVerify, verbose bool) *TLSWrapper {
	return &TLSWrapper{
		serverName: serverName,
		skipVerify: skipVerify,
		verbose:    verbose,
	}
}

// Wrap wraps an existing connection with TLS.
func (t *TLSWrapper) Wrap(conn net.Conn, timeout time.Duration) (net.Conn, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	if t.verbose {
		fmt.Fprintf(os.Stderr, "Starting TLS handshake with %s\n", t.serverName)
	}

	config := &tls.Config{
		ServerName:         t.serverName,
		InsecureSkipVerify: t.skipVerify,
	}

	tlsConn := tls.Client(conn, config)

	if err := tlsConn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	if err := tlsConn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	if err := tlsConn.SetDeadline(time.Time{}); err != nil {
		_ = tlsConn.Close()
		return nil, err
	}

	if t.verbose {
		state := tlsConn.ConnectionState()
		fmt.Fprintf(os.Stderr, "TLS established: version=%x, cipher=%s\n", state.Version, tls.CipherSuiteName(state.CipherSuite))
	}

	return tlsConn, nil
}

// DialAndWrap connects to a server using TLS directly.
func DialAndWrap(address string, timeout time.Duration, serverName string, skipVerify, verbose bool) (net.Conn, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	if serverName == "" {
		host, _, _ := net.SplitHostPort(address)
		serverName = host
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Connecting to %s with TLS\n", address)
	}

	config := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: skipVerify,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}

	if verbose {
		state := conn.ConnectionState()
		fmt.Fprintf(os.Stderr, "TLS established: version=%x, cipher=%s\n", state.Version, tls.CipherSuiteName(state.CipherSuite))
	}

	return conn, nil
}
