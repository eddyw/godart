//go:build dart

package net

import (
	"errors"
	"io"
	"time"
)

// This file holds the TCP connection type of Go. `Dial`, `Dialer.DialContext`,
// and `Listener.Accept` each give a `*TCPConn` for the network tcp, thus a
// program that names the type in an assertion reads it.
//
// `dart:io` owns the socket. It gives no file descriptor, and it exposes only
// 1 socket option that no intrinsic of `syscall/dart` reaches. The methods
// `CloseRead`, `CloseWrite`, `SetLinger`, `SetNoDelay`, `SetKeepAlive`,
// `SetKeepAlivePeriod`, `SetKeepAliveConfig`, `SetReadBuffer`,
// `SetWriteBuffer`, `MultipathTCP`, `SyscallConn`, and `File` each need 1 of
// the 2, thus the overlay holds none of them.
//
// Go embeds the unexported type `conn` in `TCPConn`. The overlay names each
// method of the connection again, because a promoted method of an unexported
// type reaches no other package in the Dart output.

// A TCPConn is a connection of the network tcp. It holds each method of the
// Conn interface, and it adds the 2 copy methods of Go.
type TCPConn struct {
	c conn
}

// newTCPConn wraps the handle of an open TCP socket.
func newTCPConn(handle int) *TCPConn {
	return &TCPConn{c: conn{handle: handle, network: "tcp"}}
}

// Read reads bytes of the connection into the slice.
func (t *TCPConn) Read(b []byte) (int, error) { return t.c.Read(b) }

// Write writes the bytes of the slice to the connection.
func (t *TCPConn) Write(b []byte) (int, error) { return t.c.Write(b) }

// Close ends the connection in both directions.
func (t *TCPConn) Close() error { return t.c.Close() }

// LocalAddr gives the address of this side of the connection.
func (t *TCPConn) LocalAddr() Addr { return t.c.LocalAddr() }

// RemoteAddr gives the address of the peer.
func (t *TCPConn) RemoteAddr() Addr { return t.c.RemoteAddr() }

// SetDeadline writes the deadline of a read and of a write.
func (t *TCPConn) SetDeadline(d time.Time) error { return t.c.SetDeadline(d) }

// SetReadDeadline writes the deadline of a read.
func (t *TCPConn) SetReadDeadline(d time.Time) error { return t.c.SetReadDeadline(d) }

// SetWriteDeadline writes the deadline of a write.
func (t *TCPConn) SetWriteDeadline(d time.Time) error { return t.c.SetWriteDeadline(d) }

// onlyReader hides each other method of a value, thus io.Copy reads the plain
// Read of it. Go holds the same 2 types.
type onlyReader struct {
	io.Reader
}

// onlyWriter hides each other method of a value.
type onlyWriter struct {
	io.Writer
}

// ReadFrom writes each byte of the reader to the connection, and it gives the
// count. Go reads sendfile or splice for a source that the kernel holds, and
// the overlay always copies through a buffer.
func (t *TCPConn) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(onlyWriter{Writer: t}, r)
}

// WriteTo reads the connection up to its end and writes each byte to the
// writer.
func (t *TCPConn) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, onlyReader{Reader: t})
}

// DialTCP opens a connection to a TCP address. The network is tcp, tcp4, or
// tcp6.
//
// Dart binds no local address of a dial, thus a laddr that is not nil gives an
// error.
func DialTCP(network string, laddr, raddr *TCPAddr) (*TCPConn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, UnknownNetworkError(network)
	}
	if laddr != nil {
		return nil, errors.New("dial " + network +
			": a local address of a dial stays outside the subset")
	}
	if raddr == nil {
		return nil, errors.New("dial " + network + ": the address is missing")
	}
	c, err := Dial("tcp", raddr.String())
	if err != nil {
		return nil, err
	}
	tc, ok := c.(*TCPConn)
	if !ok {
		c.Close()
		return nil, errors.New("dial " + network + ": the dial gave no TCP connection")
	}
	return tc, nil
}
