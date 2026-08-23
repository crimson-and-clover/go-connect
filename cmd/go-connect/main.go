// goconnect is a netcat-like tool with proxy support.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/crimson-and-clover/go-connect/internal/config"
	"github.com/crimson-and-clover/go-connect/pkg/netcat"
	"github.com/crimson-and-clover/go-connect/pkg/proxy"
	"github.com/crimson-and-clover/go-connect/pkg/transport"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	opts, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if opts.ShowVersion {
		fmt.Printf("go-connect version %s (%s)\n", version, buildTime)
		return
	}

	if opts.ListenMode {
		if opts.UDPMode {
			if err := netcat.RunUDPListener(opts.ListenPort, opts.Verbose, opts.HexDump); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		var listener *netcat.Listener
		if opts.UnixSocket {
			listener = netcat.NewUnixListener(opts.TargetHost, opts.Verbose, opts.HexDump)
		} else {
			listener = netcat.NewListener(opts.ListenPort, opts.Verbose, opts.HexDump)
		}
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

	if opts.UDPMode {
		if err := netcat.RunUDPClient(opts.TargetAddress(), opts.Timeout, opts.Verbose, opts.HexDump); err != nil {
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

	if opts.UnixSocket {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Connecting to unix socket %s\n", opts.TargetHost)
		}
		conn, err = net.DialTimeout("unix", opts.TargetHost, opts.Timeout)
	} else if opts.TLSEnable {
		// TLS direct connection or via proxy
		conn, err = dialWithTLS(opts)
	} else {
		// Create dialer based on proxy configuration
		dialerConfig := proxy.Config{
			Timeout:            opts.Timeout,
			InsecureSkipVerify: opts.InsecureSkipVerify,
			Verbose:            opts.Verbose,
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
	defer func() { _ = conn.Close() }()

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Connected to %s\n", opts.TargetAddress())
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	remoteDone := make(chan struct{})
	stdinDone := make(chan struct{})

	var dumpMu sync.Mutex
	var stdinWriter io.Writer = conn
	var stdoutWriter io.Writer = os.Stdout

	if opts.HexDump {
		stdinWriter = netcat.NewHexDumpWriter(conn, ">>> Sent", &dumpMu)
		stdoutWriter = netcat.NewHexDumpWriter(os.Stdout, "<<< Received", &dumpMu)
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
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, "Connection closed by remote")
		}
	case <-sigCh:
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, "\nInterrupted")
		}
	}

	return nil
}

// runScanMode runs port scanning mode.
func runScanMode(opts *config.Options) error {
	ports := opts.ScanPorts
	if len(ports) == 0 {
		var err error
		ports, err = config.ParsePortSpecs([]string{opts.TargetPort})
		if err != nil {
			return err
		}
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	dialerConfig := proxy.Config{
		Timeout:            timeout,
		InsecureSkipVerify: opts.InsecureSkipVerify,
		Verbose:            false, // Keep proxy dialer quiet during scanning
	}

	dialer, err := proxy.NewDialer(opts.ProxyURL, dialerConfig)
	if err != nil {
		return err
	}

	if opts.Verbose {
		if opts.ProxyURL != "" {
			fmt.Fprintf(os.Stderr, "Scanning %s (%d ports) via %s (timeout: %v)...\n",
				opts.TargetHost, len(ports), opts.ProxyURL, timeout)
		} else {
			fmt.Fprintf(os.Stderr, "Scanning %s (%d ports, timeout: %v)...\n",
				opts.TargetHost, len(ports), timeout)
		}
	}

	scanner := netcat.NewScanner(opts.TargetHost, ports, timeout, opts.Verbose, dialer)
	if opts.UDPMode {
		scanner.SetNetwork("udp")
	}
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
			opts.InsecureSkipVerify,
			opts.Verbose,
		)
	}

	// First connect through proxy, then wrap with TLS
	dialerConfig := proxy.Config{
		Timeout:            opts.Timeout,
		InsecureSkipVerify: opts.InsecureSkipVerify,
		Verbose:            opts.Verbose,
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
	tlsWrapper := transport.NewTLSWrapper(opts.TargetHost, opts.InsecureSkipVerify, opts.Verbose)
	return tlsWrapper.Wrap(conn, opts.Timeout)
}

