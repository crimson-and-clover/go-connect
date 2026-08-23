package config

import (
	"testing"
	"time"
)

func TestTargetAddress(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		expected string
	}{
		{
			name: "host and port",
			opts: Options{
				TargetHost: "example.com",
				TargetPort: "80",
			},
			expected: "example.com:80",
		},
		{
			name: "host only",
			opts: Options{
				TargetHost: "example.com",
			},
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.TargetAddress(); got != tt.expected {
				t.Errorf("TargetAddress() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantHost  string
		wantPort  string
		wantProxy string
		wantTLS   bool
		wantSkip  bool
		wantTout  time.Duration
		wantVerb  bool
		wantZero  bool
		wantList  bool
		wantLPort int
		wantErr   bool
	}{
		{
			name:     "basic direct connection",
			args:     []string{"example.com", "80"},
			wantHost: "example.com",
			wantPort: "80",
			wantTout: 30 * time.Second,
		},
		{
			name:      "http proxy with tls",
			args:      []string{"-x", "http://127.0.0.1:8080", "-T", "-k", "-v", "example.com", "443"},
			wantHost:  "example.com",
			wantPort:  "443",
			wantProxy: "http://127.0.0.1:8080",
			wantTLS:   true,
			wantSkip:  true,
			wantVerb:  true,
			wantTout:  30 * time.Second,
		},
		{
			name:     "timeout override with -w",
			args:     []string{"-t", "10s", "-w", "5s", "example.com", "80"},
			wantHost: "example.com",
			wantPort: "80",
			wantTout: 5 * time.Second,
		},
		{
			name:      "listen mode",
			args:      []string{"-l", "-p", "9000", "-v"},
			wantList:  true,
			wantLPort: 9000,
			wantVerb:  true,
		},
		{
			name:     "scan mode with port range",
			args:     []string{"-z", "example.com", "20-80"},
			wantHost: "example.com",
			wantPort: "20-80",
			wantZero: true,
		},
		{
			name:    "listen mode missing port error",
			args:    []string{"-l"},
			wantErr: true,
		},
		{
			name:    "missing target args error",
			args:    []string{"example.com"},
			wantErr: true,
		},
		{
			name:    "invalid port number error",
			args:    []string{"example.com", "999999"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if opts.TargetHost != tt.wantHost {
				t.Errorf("TargetHost = %v, want %v", opts.TargetHost, tt.wantHost)
			}
			if opts.TargetPort != tt.wantPort {
				t.Errorf("TargetPort = %v, want %v", opts.TargetPort, tt.wantPort)
			}
			if opts.ProxyURL != tt.wantProxy {
				t.Errorf("ProxyURL = %v, want %v", opts.ProxyURL, tt.wantProxy)
			}
			if opts.TLSEnable != tt.wantTLS {
				t.Errorf("TLSEnable = %v, want %v", opts.TLSEnable, tt.wantTLS)
			}
			if opts.InsecureSkipVerify != tt.wantSkip {
				t.Errorf("InsecureSkipVerify = %v, want %v", opts.InsecureSkipVerify, tt.wantSkip)
			}
			if tt.wantTout != 0 && opts.Timeout != tt.wantTout {
				t.Errorf("Timeout = %v, want %v", opts.Timeout, tt.wantTout)
			}
			if opts.Verbose != tt.wantVerb {
				t.Errorf("Verbose = %v, want %v", opts.Verbose, tt.wantVerb)
			}
			if opts.ZeroMode != tt.wantZero {
				t.Errorf("ZeroMode = %v, want %v", opts.ZeroMode, tt.wantZero)
			}
			if opts.ListenMode != tt.wantList {
				t.Errorf("ListenMode = %v, want %v", opts.ListenMode, tt.wantList)
			}
			if opts.ListenPort != tt.wantLPort {
				t.Errorf("ListenPort = %v, want %v", opts.ListenPort, tt.wantLPort)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	opts, err := ParseArgs([]string{"-V"})
	if err != nil {
		t.Fatalf("ParseArgs(-V) error = %v", err)
	}
	if !opts.ShowVersion {
		t.Errorf("ShowVersion = %v, want true", opts.ShowVersion)
	}
}
