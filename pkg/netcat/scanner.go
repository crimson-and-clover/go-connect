// Package netcat provides netcat-like functionality.
package netcat

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Dialer connects to an address.
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

// ScanResult represents the result of scanning a single port.
type ScanResult struct {
	Port    int
	Open    bool
	Error   error
	Latency time.Duration
}

// Scanner provides port scanning functionality.
type Scanner struct {
	host    string
	ports   []int
	timeout time.Duration
	verbose bool
	workers int
	dialer  Dialer
}

// NewScanner creates a new port scanner from a list of ports.
func NewScanner(host string, ports []int, timeout time.Duration, verbose bool, dialer Dialer) *Scanner {
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	// Deduplicate and filter ports
	portMap := make(map[int]struct{}, len(ports))
	for _, p := range ports {
		if p >= 1 && p <= 65535 {
			portMap[p] = struct{}{}
		}
	}
	sortedPorts := make([]int, 0, len(portMap))
	for p := range portMap {
		sortedPorts = append(sortedPorts, p)
	}
	sort.Ints(sortedPorts)

	workers := 100
	if len(sortedPorts) < workers {
		workers = len(sortedPorts)
	}
	if workers < 1 {
		workers = 1
	}

	if dialer == nil {
		dialer = &net.Dialer{Timeout: timeout}
	}

	return &Scanner{
		host:    host,
		ports:   sortedPorts,
		timeout: timeout,
		verbose: verbose,
		workers: workers,
		dialer:  dialer,
	}
}

// NewRangeScanner creates a new port scanner for a continuous range of ports.
func NewRangeScanner(host string, startPort, endPort int, timeout time.Duration, verbose bool, dialer Dialer) *Scanner {
	if startPort > endPort {
		startPort, endPort = endPort, startPort
	}
	ports := make([]int, 0, endPort-startPort+1)
	for p := startPort; p <= endPort; p++ {
		ports = append(ports, p)
	}
	return NewScanner(host, ports, timeout, verbose, dialer)
}

// NewSinglePortScanner creates a scanner for a single port check.
func NewSinglePortScanner(host string, port int, timeout time.Duration, verbose bool, dialer Dialer) *Scanner {
	return NewScanner(host, []int{port}, timeout, verbose, dialer)
}

// Scan performs the port scan and returns sorted results.
func (s *Scanner) Scan() []ScanResult {
	results := make([]ScanResult, 0, len(s.ports))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Port channel
	portCh := make(chan int, s.workers)

	// Start workers
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portCh {
				result := s.scanPort(port)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}()
	}

	// Send ports to scan
	for _, port := range s.ports {
		portCh <- port
	}
	close(portCh)

	wg.Wait()

	// Sort results by port number ascending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Port < results[j].Port
	})

	return results
}

// scanPort scans a single port.
func (s *Scanner) scanPort(port int) ScanResult {
	address := net.JoinHostPort(s.host, strconv.Itoa(port))
	start := time.Now()

	conn, err := s.dialer.Dial("tcp", address)
	latency := time.Since(start)

	if err != nil {
		return ScanResult{
			Port:    port,
			Open:    false,
			Error:   err,
			Latency: latency,
		}
	}
	_ = conn.Close()

	return ScanResult{
		Port:    port,
		Open:    true,
		Latency: latency,
	}
}

// PrintResults prints scan results in a readable format.
func (s *Scanner) PrintResults(results []ScanResult) {
	openCount := 0
	for _, r := range results {
		if r.Open {
			openCount++
			fmt.Fprintf(os.Stderr, "Port %d open (%.2f ms)\n", r.Port, float64(r.Latency.Microseconds())/1000.0)
		} else if s.verbose {
			fmt.Fprintf(os.Stderr, "Port %d closed/filtered\n", r.Port)
		}
	}
	fmt.Fprintf(os.Stderr, "\nScan complete: %d ports scanned, %d open\n", len(results), openCount)
}

// CheckSinglePort checks if a single port is open.
func CheckSinglePort(host string, port int, timeout time.Duration, dialer Dialer) bool {
	scanner := NewSinglePortScanner(host, port, timeout, false, dialer)
	results := scanner.Scan()
	return len(results) > 0 && results[0].Open
}
