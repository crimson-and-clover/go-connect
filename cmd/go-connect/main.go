// goconnect is a netcat-like tool with proxy support.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/crimson-and-clover/go-connect/internal/config"
	"github.com/crimson-and-clover/go-connect/pkg/netcat"
	"github.com/crimson-and-clover/go-connect/pkg/proxy"
	"github.com/crimson-and-clover/go-connect/pkg/transport"
)

func main() {
	opts, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if opts.ListenMode {
		listener := netcat.NewListener(opts.ListenPort, opts.Verbose)
		if err := listener.Listen(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if opts.ZeroMode {
		if err := runScanMode(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runClient(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runClient(opts *config.Options) error {
	var conn net.Conn
	var err error

	if opts.TLSEnable {
		// TLS direct connection or via proxy
		conn, err = dialWithTLS(opts)
	} else {
		// Create dialer based on proxy configuration
		dialerConfig := proxy.Config{
			Timeout:   opts.Timeout,
			TLSVerify: !opts.TLSVerify, // -k means skip verification
			Verbose:   opts.Verbose,
		}

		dialer, err2 := proxy.NewDialer(opts.ProxyURL, dialerConfig)
		if err2 != nil {
			return err2
		}

		if opts.Verbose {
			if opts.ProxyURL != "" {
				fmt.Fprintf(os.Stderr, "Connecting to %s via %s\n", opts.TargetAddress(), opts.ProxyURL)
			} else {
				fmt.Fprintf(os.Stderr, "Connecting to %s (direct)\n", opts.TargetAddress())
			}
		}

		// Connect to target
		conn, err = dialer.Dial("tcp", opts.TargetAddress())
	}

	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Connected to %s\n", opts.TargetAddress())
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, os.Stdin)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stdout, conn)
	}()

	// Wait for either direction to finish or signal
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-sigCh:
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, "\nInterrupted")
		}
	}

	return nil
}

// runScanMode runs port scanning mode.
func runScanMode(opts *config.Options) error {
	// Parse port range
	ports := strings.Split(opts.TargetPort, "-")
	startPort, err := strconv.Atoi(ports[0])
	if err != nil {
		return fmt.Errorf("invalid start port: %s", ports[0])
	}

	endPort := startPort
	if len(ports) > 1 {
		endPort, err = strconv.Atoi(ports[1])
		if err != nil {
			return fmt.Errorf("invalid end port: %s", ports[1])
		}
	}

	if startPort < 1 || endPort > 65535 || startPort > endPort {
		return fmt.Errorf("invalid port range: %d-%d", startPort, endPort)
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Scanning %s ports %d-%d (timeout: %v)...\n",
			opts.TargetHost, startPort, endPort, timeout)
	}

	scanner := netcat.NewScanner(opts.TargetHost, startPort, endPort, timeout, opts.Verbose)
	results := scanner.Scan()
	scanner.PrintResults(results)

	return nil
}

// dialWithTLS handles TLS connections, optionally through a proxy.
func dialWithTLS(opts *config.Options) (net.Conn, error) {
	if opts.ProxyURL == "" {
		// Direct TLS connection
		return transport.DialAndWrap(
			opts.TargetAddress(),
			opts.Timeout,
			opts.TargetHost,
			opts.TLSVerify,
			opts.Verbose,
		)
	}

	// First connect through proxy, then wrap with TLS
	dialerConfig := proxy.Config{
		Timeout:   opts.Timeout,
		TLSVerify: !opts.TLSVerify,
		Verbose:   opts.Verbose,
	}

	dialer, err := proxy.NewDialer(opts.ProxyURL, dialerConfig)
	if err != nil {
		return nil, err
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to %s via %s, then upgrading to TLS\n",
			opts.TargetAddress(), opts.ProxyURL)
	}

	conn, err := dialer.Dial("tcp", opts.TargetAddress())
	if err != nil {
		return nil, err
	}

	// Wrap the connection with TLS
	tlsWrapper := transport.NewTLSWrapper(opts.TargetHost, opts.TLSVerify, opts.Verbose)
	return tlsWrapper.Wrap(conn, opts.Timeout)
}

