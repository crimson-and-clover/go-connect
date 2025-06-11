package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

func createConnetion(upstreamProxy, targetHost, targetPort string) {
	// create tcp connection to the upstream proxy
	conn, err := net.Dial("tcp", upstreamProxy)
	if err != nil {
		fmt.Errorf("Failed to connect to upstream proxy %s: %v", upstreamProxy, err)
		os.Exit(1)
	}
	defer conn.Close()
	// send CONNECT request to the upstream proxy
	connectRequest := fmt.Sprintf("CONNECT %s:%s HTTP/1.1\r\nHost: %s:%s\r\n\r\n", targetHost, targetPort, targetHost, targetPort)
	_, err = conn.Write([]byte(connectRequest))
	if err != nil {
		fmt.Errorf("Failed to send CONNECT request: %v", err)
		os.Exit(1)
	}
	buf := make([]byte, 4096)
	// read response from the upstream proxy
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Errorf("Failed to read response from upstream proxy: %v", err)
		os.Exit(1)
	}
	response := string(buf[:n])
	if n == 0 {
		fmt.Fprintln(os.Stderr, "Error: No response from upstream proxy")
		os.Exit(1)
	}
	if response[:12] != "HTTP/1.1 200" {
		fmt.Fprintf(os.Stderr, "Failed to connect to %s:%s\n", targetHost, targetPort)
		fmt.Fprintln(os.Stderr, response)
		os.Exit(1)
	}
	go io.Copy(conn, os.Stdin)
	io.Copy(os.Stdout, conn)
}

func main() {
	var (
		upstreamProxy string
	)
	flag.StringVar(&upstreamProxy, "H", "", "Upstream HTTP proxy (host:port)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s -H proxyhost:port hostname port\n", os.Args[0])
		os.Exit(1)
	}

	targetHost := args[0]
	targetPort := args[1]

	// check if upstream proxy is provided
	if upstreamProxy == "" {
		fmt.Fprintln(os.Stderr, "Error: Upstream proxy is required")
		os.Exit(1)
	}
	// check if target host and port are provided
	if targetHost == "" || targetPort == "" {
		fmt.Fprintln(os.Stderr, "Error: Target host and port are required")
		os.Exit(1)
	}

	// check if port is a valid number
	if _, err := fmt.Sscanf(targetPort, "%d", new(int)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid port number '%s'\n", targetPort)
		os.Exit(1)
	}

	createConnetion(upstreamProxy, targetHost, targetPort)
}
