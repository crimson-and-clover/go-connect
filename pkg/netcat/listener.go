package netcat

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Listener provides listen mode functionality for TCP and Unix domain sockets.
type Listener struct {
	network string
	address string
	verbose bool
	hexDump bool
}

// NewListener creates a new TCP listener.
func NewListener(port int, verbose bool, hexDump bool) *Listener {
	return &Listener{
		network: "tcp",
		address: fmt.Sprintf(":%d", port),
		verbose: verbose,
		hexDump: hexDump,
	}
}

// NewUnixListener creates a new Unix Domain Socket listener.
func NewUnixListener(socketPath string, verbose bool, hexDump bool) *Listener {
	return &Listener{
		network: "unix",
		address: socketPath,
		verbose: verbose,
		hexDump: hexDump,
	}
}

// Listen starts listening on the specified network address.
func (l *Listener) Listen() error {
	if l.network == "unix" {
		_ = os.Remove(l.address)
		defer func() { _ = os.Remove(l.address) }()
	}

	ln, err := net.Listen(l.network, l.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s (%s): %w", l.address, l.network, err)
	}
	defer func() { _ = ln.Close() }()

	if l.network == "unix" {
		fmt.Fprintf(os.Stderr, "Listening on unix socket %s...\n", l.address)
	} else {
		fmt.Fprintf(os.Stderr, "Listening on %s...\n", l.address)
	}

	// Handle shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Accept connections in a goroutine
	connCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}()

	select {
	case <-sigCh:
		fmt.Fprintln(os.Stderr, "\nInterrupted")
		return nil
	case err := <-errCh:
		return err
	case conn := <-connCh:
		return l.handleConnection(conn)
	}
}

// handleConnection handles a single client connection.
func (l *Listener) handleConnection(conn net.Conn) error {
	defer func() { _ = conn.Close() }()

	if l.verbose {
		fmt.Fprintf(os.Stderr, "Connection from %s\n", conn.RemoteAddr())
	}

	fmt.Fprintln(os.Stderr, "Connection established. Press Ctrl+C to close.")

	// Handle shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	remoteDone := make(chan struct{})
	stdinDone := make(chan struct{})

	var dumpMu sync.Mutex
	var stdinWriter io.Writer = conn
	var stdoutWriter io.Writer = os.Stdout

	if l.hexDump {
		stdinWriter = NewHexDumpWriter(conn, ">>> Sent", &dumpMu)
		stdoutWriter = NewHexDumpWriter(os.Stdout, "<<< Received", &dumpMu)
	}

	// Stdin -> Conn (sends half-close FIN on stdin EOF)
	go func() {
		_, _ = io.Copy(stdinWriter, os.Stdin)
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		close(stdinDone)
	}()

	// Conn -> Stdout
	go func() {
		_, _ = io.Copy(stdoutWriter, conn)
		close(remoteDone)
	}()

	select {
	case <-remoteDone:
		if l.verbose {
			fmt.Fprintln(os.Stderr, "Connection closed by remote")
		}
	case <-sigCh:
		if l.verbose {
			fmt.Fprintln(os.Stderr, "\nInterrupted")
		}
	}

	return nil
}
