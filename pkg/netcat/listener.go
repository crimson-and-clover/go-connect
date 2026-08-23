package netcat

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// Listener provides listen mode functionality.
type Listener struct {
	port    int
	verbose bool
}

// NewListener creates a new listener.
func NewListener(port int, verbose bool) *Listener {
	return &Listener{
		port:    port,
		verbose: verbose,
	}
}

// Listen starts listening on the specified port.
func (l *Listener) Listen() error {
	address := fmt.Sprintf(":%d", l.port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	defer func() { _ = ln.Close() }()

	fmt.Fprintf(os.Stderr, "Listening on port %d...\n", l.port)

	// Handle shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

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

	// Stdin -> Conn (sends half-close FIN on stdin EOF)
	go func() {
		_, _ = io.Copy(conn, os.Stdin)
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		close(stdinDone)
	}()

	// Conn -> Stdout
	go func() {
		_, _ = io.Copy(os.Stdout, conn)
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

// ListenAndServe listens on the port and serves connections.
// If single is true, accepts only one connection; otherwise accepts continuously.
func (l *Listener) ListenAndServe(single bool) error {
	address := fmt.Sprintf(":%d", l.port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	defer func() { _ = ln.Close() }()

	fmt.Fprintf(os.Stderr, "Listening on port %d...\n", l.port)

	// Handle shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	acceptCh := make(chan net.Conn)
	errCh := make(chan error)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			acceptCh <- conn
			if single {
				return
			}
		}
	}()

	for {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nShutting down...")
			return nil
		case err := <-errCh:
			return err
		case conn := <-acceptCh:
			if single {
				return l.handleConnection(conn)
			}
			go func() { _ = l.handleConnection(conn) }()
		}
	}
}
