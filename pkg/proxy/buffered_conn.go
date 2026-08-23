package proxy

import (
	"io"
	"net"
)

// bufferedConn wraps a net.Conn with a reader that may contain pre-buffered data.
type bufferedConn struct {
	net.Conn
	r io.Reader
}

func newBufferedConn(conn net.Conn, r io.Reader) net.Conn {
	return &bufferedConn{
		Conn: conn,
		r:    r,
	}
}

// Read reads from the buffered reader first, and then directly from the underlying connection.
func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

// CloseWrite closes the write side of the connection if the underlying connection supports it.
func (b *bufferedConn) CloseWrite() error {
	if cw, ok := b.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}
