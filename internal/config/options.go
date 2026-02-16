// Package config provides configuration structures and parsing.
package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Options holds all command-line options.
type Options struct {
	ProxyURL   string
	TLSEnable  bool
	TLSVerify  bool
	Timeout    time.Duration
	Verbose    bool
	ZeroMode   bool // Port scanning mode
	ListenMode bool
	ListenPort int
	SourceAddr string
	QuitDelay  time.Duration
	TargetHost string
	TargetPort string
}

// Parse parses command-line arguments and returns Options.
func Parse() (*Options, error) {
	opts := &Options{}

	flag.StringVar(&opts.ProxyURL, "x", "", "Proxy URL (http://host:port, socks5://host:port, etc.)")
	flag.BoolVar(&opts.TLSEnable, "T", false, "Enable TLS")
	flag.BoolVar(&opts.TLSVerify, "k", false, "Skip TLS certificate verification")
	flag.DurationVar(&opts.Timeout, "t", 30*time.Second, "Connection timeout")
	flag.BoolVar(&opts.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&opts.ZeroMode, "z", false, "Zero I/O mode (port scanning)")
	flag.BoolVar(&opts.ListenMode, "l", false, "Listen mode")
	flag.IntVar(&opts.ListenPort, "p", 0, "Port to listen on")
	flag.StringVar(&opts.SourceAddr, "s", "", "Source address")
	flag.DurationVar(&opts.QuitDelay, "q", 0, "Quit after EOF on stdin (with delay)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] host port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -l -p port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	// Custom -w flag that overrides -t
	wFlag := flag.Duration("w", 0, "Timeout (alias, nc compatible)")

	flag.Parse()

	// If -w was explicitly set (non-zero), use it instead of -t
	if *wFlag != 0 {
		opts.Timeout = *wFlag
	}

	if opts.ListenMode {
		if opts.ListenPort == 0 {
			return nil, fmt.Errorf("listen mode requires -p port")
		}
		return opts, nil
	}

	// Validate target host and port
	args := flag.Args()
	if opts.ZeroMode && len(args) >= 2 {
		// Port scanning mode can have host and port range
		opts.TargetHost = args[0]
		opts.TargetPort = args[1]
	} else if len(args) != 2 {
		return nil, fmt.Errorf("requires target host and port")
	} else {
		opts.TargetHost = args[0]
		opts.TargetPort = args[1]
	}

	// Validate port
	if opts.TargetPort != "" {
		var port int
		if _, err := fmt.Sscanf(opts.TargetPort, "%d", &port); err != nil {
			return nil, fmt.Errorf("invalid port: %s", opts.TargetPort)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("port out of range: %d", port)
		}
	}

	return opts, nil
}

// TargetAddress returns the full target address (host:port).
func (o *Options) TargetAddress() string {
	if o.TargetPort == "" {
		return o.TargetHost
	}
	return o.TargetHost + ":" + o.TargetPort
}
