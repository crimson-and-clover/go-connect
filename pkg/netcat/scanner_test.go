package netcat

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestScannerSortAndDetection(t *testing.T) {
	// Start a local TCP server on an available port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer func() { _ = ln.Close() }()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host port: %v", err)
	}
	openPort, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("invalid port: %v", err)
	}

	// Range covering openPort - 2 to openPort + 2
	startPort := openPort - 2
	endPort := openPort + 2

	scanner := NewRangeScanner("127.0.0.1", startPort, endPort, 500*time.Millisecond, false, nil)
	results := scanner.Scan()

	if len(results) != endPort-startPort+1 {
		t.Fatalf("expected %d results, got %d", endPort-startPort+1, len(results))
	}

	// Verify strict ascending order
	foundOpen := false
	for i, r := range results {
		expectedPort := startPort + i
		if r.Port != expectedPort {
			t.Errorf("result[%d].Port = %d, want %d", i, r.Port, expectedPort)
		}
		if r.Port == openPort {
			if !r.Open {
				t.Errorf("port %d was expected to be open, but reported closed", openPort)
			}
			foundOpen = true
		}
	}

	if !foundOpen {
		t.Errorf("open port %d was not found in scan results", openPort)
	}

	// Test slice scanner with discrete unordered ports
	discreteScanner := NewScanner("127.0.0.1", []int{openPort + 5, openPort, openPort + 2}, 500*time.Millisecond, false, nil)
	discreteResults := discreteScanner.Scan()
	if len(discreteResults) != 3 {
		t.Fatalf("expected 3 results, got %d", len(discreteResults))
	}
	if discreteResults[0].Port != openPort || !discreteResults[0].Open {
		t.Errorf("first result expected to be open port %d, got %v", openPort, discreteResults[0])
	}
	if discreteResults[1].Port != openPort+2 {
		t.Errorf("second result expected port %d, got %d", openPort+2, discreteResults[1].Port)
	}
	if discreteResults[2].Port != openPort+5 {
		t.Errorf("third result expected port %d, got %d", openPort+5, discreteResults[2].Port)
	}
}

func TestCheckSinglePort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer func() { _ = ln.Close() }()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	openPort, _ := strconv.Atoi(portStr)

	if !CheckSinglePort("127.0.0.1", openPort, 500*time.Millisecond, nil) {
		t.Errorf("expected port %d to be open", openPort)
	}

	// Check a closed port (e.g. 1)
	if CheckSinglePort("127.0.0.1", 1, 100*time.Millisecond, nil) {
		t.Errorf("expected port 1 to be closed")
	}
}
