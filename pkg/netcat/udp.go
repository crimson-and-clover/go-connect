package netcat

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// RunUDPClient handles a client UDP connection.
func RunUDPClient(targetAddress string, timeout time.Duration, verbose bool, hexDump bool) error {
	raddr, err := net.ResolveUDPAddr("udp", targetAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return fmt.Errorf("failed to connect to UDP target: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if verbose {
		fmt.Fprintf(os.Stderr, "Connected to %s (UDP)\n", targetAddress)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	var dumpMu sync.Mutex
	var stdinWriter io.Writer = conn
	var stdoutWriter io.Writer = os.Stdout

	if hexDump {
		stdinWriter = NewHexDumpWriter(conn, ">>> Sent", &dumpMu)
		stdoutWriter = NewHexDumpWriter(os.Stdout, "<<< Received", &dumpMu)
	}

	stdinDone := make(chan struct{})
	remoteDone := make(chan struct{})

	// Stdin -> UDP
	go func() {
		_, _ = io.Copy(stdinWriter, os.Stdin)
		close(stdinDone)
	}()

	// UDP -> Stdout
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				_, _ = stdoutWriter.Write(buf[:n])
			}
		}
		close(remoteDone)
	}()

	select {
	case <-stdinDone:
		idleTimeout := timeout
		if idleTimeout == 0 || idleTimeout == 30*time.Second {
			idleTimeout = 2 * time.Second
		}
		_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
		select {
		case <-remoteDone:
		case <-sigCh:
		case <-time.After(idleTimeout):
		}
	case <-remoteDone:
	case <-sigCh:
		if verbose {
			fmt.Fprintln(os.Stderr, "\nInterrupted")
		}
	}

	return nil
}

// RunUDPListener handles UDP listen mode.
func RunUDPListener(port int, verbose bool, hexDump bool) error {
	laddr := &net.UDPAddr{Port: port}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", port, err)
	}
	defer func() { _ = conn.Close() }()

	fmt.Fprintf(os.Stderr, "Listening on UDP port %d...\n", port)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	var dumpMu sync.Mutex
	var stdoutWriter io.Writer = os.Stdout
	if hexDump {
		stdoutWriter = NewHexDumpWriter(os.Stdout, "<<< Received", &dumpMu)
	}

	var lastClient *net.UDPAddr
	var clientMu sync.Mutex

	// Stdin -> Last seen UDP Client
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				clientMu.Lock()
				client := lastClient
				clientMu.Unlock()
				if client != nil {
					if hexDump {
						dumpMu.Lock()
						dump := NewHexDumpWriter(io.Discard, ">>> Sent", &dumpMu)
						_, _ = dump.Write(buf[:n])
						dumpMu.Unlock()
					}
					_, _ = conn.WriteToUDP(buf[:n], client)
				}
			}
		}
	}()

	readCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 65535)
		for {
			n, raddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				readCh <- err
				return
			}
			clientMu.Lock()
			lastClient = raddr
			clientMu.Unlock()

			if verbose {
				fmt.Fprintf(os.Stderr, "Datagram received from %s (%d bytes)\n", raddr, n)
			}
			if n > 0 {
				_, _ = stdoutWriter.Write(buf[:n])
			}
		}
	}()

	select {
	case <-sigCh:
		if verbose {
			fmt.Fprintln(os.Stderr, "\nInterrupted")
		}
		return nil
	case err := <-readCh:
		return err
	}
}
