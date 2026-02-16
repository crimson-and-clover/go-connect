# go-connect

[![CI](https://github.com/crimson-and-clover/go-connect/actions/workflows/ci.yml/badge.svg)](https://github.com/crimson-and-clover/go-connect/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/crimson-and-clover/go-connect)](https://github.com/crimson-and-clover/go-connect/releases/latest)
[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://golang.org)
[![License](https://img.shields.io/github/license/crimson-and-clover/go-connect)](LICENSE)
[![CodeQL](https://github.com/crimson-and-clover/go-connect/actions/workflows/codeql.yml/badge.svg)](https://github.com/crimson-and-clover/go-connect/actions/workflows/codeql.yml)

A netcat-like tool with proxy support, written in Go.

## Features

- **Direct TCP connections** - Connect directly to any TCP port
- **HTTP/HTTPS Proxy** - Connect through HTTP CONNECT proxies with authentication
- **SOCKS5 Proxy** - Support for SOCKS5 proxies with authentication (using golang.org/x/net/proxy)
- **TLS/SSL Support** - Direct TLS connections or TLS over proxy
- **Port Scanning** - Zero I/O mode for port scanning
- **Listen Mode** - Act as a server and accept connections
- **Verbose Output** - Detailed connection information

## Installation

```bash
# Install from source
go install github.com/crimson-and-clover/go-connect/cmd/go-connect@latest

# Or clone and build locally
git clone https://github.com/crimson-and-clover/go-connect.git
cd go-connect
go install ./cmd/go-connect

# Or use make
make build
make install
```

## Usage

### Direct TCP Connection

```bash
# Simple connection (like nc host port)
go-connect example.com 80
go-connect -v example.com 443
```

### HTTP Proxy

```bash
# HTTP CONNECT proxy
go-connect -x http://proxy.company.com:8080 target.example.com 443

# With authentication
go-connect -x http://user:pass@proxy.company.com:8080 target.example.com 443
```

### SOCKS5 Proxy

```bash
# SOCKS5
go-connect -x socks5://proxy.example.com:1080 target.com 80

# SOCKS5 with authentication
go-connect -x socks5://user:pass@proxy.example.com:1080 target.com 80
```

### HTTPS Proxy (HTTP CONNECT over TLS)

```bash
go-connect -x https://proxy.company.com:443 target.example.com 443
```

### TLS Connections

```bash
# Direct TLS connection
go-connect -T -v api.example.com 443

# TLS through proxy
go-connect -x socks5://proxy:1080 -T api.example.com 443

# Skip certificate verification
go-connect -T -k self-signed.example.com 443
```

### Port Scanning

```bash
# Scan single port
go-connect -z -v target.example.com 443

# Scan port range
go-connect -z -v -w 1s target.example.com 1-1000
```

### Listen Mode

```bash
# Listen on port 8080
go-connect -l -p 8080

# With verbose output
go-connect -l -p 8080 -v
```

## Options

| Option | Description |
|--------|-------------|
| `-x URL` | Proxy URL (http://, https://, socks5://) |
| `-T` | Enable TLS |
| `-k` | Skip TLS certificate verification |
| `-t duration` | Connection timeout (default: 30s) |
| `-v` | Verbose output |
| `-z` | Zero I/O mode (port scanning) |
| `-l` | Listen mode |
| `-p port` | Port to listen on (with -l) |
| `-w duration` | Timeout alias (nc compatible) |

## Examples

### SSH through HTTP Proxy

```bash
ssh -o ProxyCommand="go-connect -x http://proxy:8080 %h %p" user@remotehost
```

### Git through SOCKS5 Proxy

```bash
export GIT_SSH_COMMAND="ssh -o ProxyCommand='go-connect -x socks5://127.0.0.1:1080 %h %p'"
git clone git@github.com:user/repo.git
```

### Test HTTPS Service

```bash
echo -e "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n" | go-connect -T example.com 443
```

## License

MIT License
