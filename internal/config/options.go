// Package config provides configuration structures and parsing.
package config

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
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
	UnixSocket         bool // Use Unix Domain Socket (-U)
	UDPMode            bool // Use UDP mode (-u)
	TargetHost         string
	TargetPort         string
	ScanPorts          []int // Parsed ports for scan mode
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
	fs.BoolVar(&opts.UnixSocket, "U", false, "Use Unix Domain Socket")
	fs.BoolVar(&opts.UDPMode, "u", false, "Use UDP mode")
	fs.BoolVar(&opts.ZeroMode, "z", false, "Zero I/O mode (port scanning)")
	fs.BoolVar(&opts.ListenMode, "l", false, "Listen mode")
	fs.IntVar(&opts.ListenPort, "p", 0, "Port to listen on")
	fs.BoolVar(&opts.ShowVersion, "V", false, "Show version")
	fs.BoolVar(&opts.ShowVersion, "version", false, "Show version")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] host port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -u [options] host port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -U [options] socket_path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -l -p port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -l -u -p port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -l -U socket_path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -z [options] host port(s)...\n", os.Args[0])
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

	if opts.UnixSocket {
		args := fs.Args()
		if len(args) != 1 {
			return nil, fmt.Errorf("unix socket mode requires socket path")
		}
		opts.TargetHost = args[0]
		opts.TargetPort = ""
		return opts, nil
	}

	if opts.ListenMode {
		if opts.ListenPort == 0 {
			return nil, fmt.Errorf("listen mode requires -p port")
		}
		return opts, nil
	}

	args := fs.Args()
	if opts.ZeroMode {
		if len(args) < 2 {
			return nil, fmt.Errorf("scan mode requires target host and port(s)")
		}
		opts.TargetHost = args[0]
		ports, err := ParsePortSpecs(args[1:])
		if err != nil {
			return nil, err
		}
		opts.ScanPorts = ports
		if len(ports) == 1 {
			opts.TargetPort = strconv.Itoa(ports[0])
		} else {
			opts.TargetPort = fmt.Sprintf("%d-%d", ports[0], ports[len(ports)-1])
		}
		return opts, nil
	}

	if len(args) != 2 {
		return nil, fmt.Errorf("requires target host and port")
	}
	opts.TargetHost = args[0]
	opts.TargetPort = args[1]

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

// ParsePortSpecs parses a slice of port specifications (e.g. ["80", "443", "8000-8010,9000"])
// and returns a deduplicated, sorted slice of valid port numbers.
func ParsePortSpecs(specs []string) ([]int, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("no ports specified")
	}

	portMap := make(map[int]struct{})

	for _, spec := range specs {
		parts := strings.Split(spec, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			if strings.Contains(part, "-") {
				rangeParts := strings.Split(part, "-")
				if len(rangeParts) != 2 {
					return nil, fmt.Errorf("invalid port range: %s", part)
				}
				start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				if err != nil {
					return nil, fmt.Errorf("invalid start port in range: %s", rangeParts[0])
				}
				end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err != nil {
					return nil, fmt.Errorf("invalid end port in range: %s", rangeParts[1])
				}
				if start < 1 || end > 65535 || start > end {
					return nil, fmt.Errorf("port range out of bounds (1-65535): %d-%d", start, end)
				}
				for p := start; p <= end; p++ {
					portMap[p] = struct{}{}
				}
			} else {
				port, err := strconv.Atoi(part)
				if err != nil {
					return nil, fmt.Errorf("invalid port: %s", part)
				}
				if port < 1 || port > 65535 {
					return nil, fmt.Errorf("port out of range (1-65535): %d", port)
				}
				portMap[port] = struct{}{}
			}
		}
	}

	if len(portMap) == 0 {
		return nil, fmt.Errorf("no valid ports found")
	}

	ports := make([]int, 0, len(portMap))
	for p := range portMap {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports, nil
}

// TargetAddress returns the full target address (host:port or unix socket path).
func (o *Options) TargetAddress() string {
	if o.UnixSocket || o.TargetPort == "" {
		return o.TargetHost
	}
	return o.TargetHost + ":" + o.TargetPort
}
