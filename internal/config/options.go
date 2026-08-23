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
	ProxyURL           string
	TLSEnable          bool
	InsecureSkipVerify bool // Skip TLS certificate verification (-k)
	Timeout            time.Duration
	Verbose            bool
	ZeroMode           bool // Port scanning mode
	ListenMode         bool
	ListenPort         int
	ShowVersion        bool
	HexDump            bool // Hex dump incoming/outgoing traffic
	TargetHost         string
	TargetPort         string
}

// Parse parses command-line arguments and returns Options.
func Parse() (*Options, error) {
	return ParseArgs(os.Args[1:])
}

// ParseArgs parses arguments from a given string slice.
func ParseArgs(arguments []string) (*Options, error) {
	opts := &Options{}
	fs := flag.NewFlagSet("go-connect", flag.ContinueOnError)

	fs.StringVar(&opts.ProxyURL, "x", "", "Proxy URL (http://host:port, socks5://host:port, etc.)")
	fs.BoolVar(&opts.TLSEnable, "T", false, "Enable TLS")
	fs.BoolVar(&opts.InsecureSkipVerify, "k", false, "Skip TLS certificate verification")
	fs.DurationVar(&opts.Timeout, "t", 30*time.Second, "Connection timeout")
	fs.BoolVar(&opts.Verbose, "v", false, "Verbose output")
	fs.BoolVar(&opts.HexDump, "X", false, "Hex dump incoming and outgoing traffic")
	fs.BoolVar(&opts.HexDump, "C", false, "Hex dump incoming and outgoing traffic (alias)")
	fs.BoolVar(&opts.ZeroMode, "z", false, "Zero I/O mode (port scanning)")
	fs.BoolVar(&opts.ListenMode, "l", false, "Listen mode")
	fs.IntVar(&opts.ListenPort, "p", 0, "Port to listen on")
	fs.BoolVar(&opts.ShowVersion, "V", false, "Show version")
	fs.BoolVar(&opts.ShowVersion, "version", false, "Show version")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] host port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -l -p port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fs.PrintDefaults()
	}

	// Custom -w flag that overrides -t
	wFlag := fs.Duration("w", 0, "Timeout (alias, nc compatible)")

	if err := fs.Parse(arguments); err != nil {
		return nil, err
	}

	if opts.ShowVersion {
		return opts, nil
	}

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
	args := fs.Args()
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
