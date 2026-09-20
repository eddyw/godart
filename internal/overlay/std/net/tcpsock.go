//go:build dart

package net

import (
	"errors"
	"io"
	"syscall"
	"time"
)

// This file holds the TCP connection type of Go. `Dial`, `Dialer.DialContext`,
// and `Listener.Accept` each give a `*TCPConn` for the network tcp, thus a
// program that names the type in an assertion reads it.
//
// `dart:io` owns the socket. It gives no file descriptor. The methods
// `SetLinger`, `MultipathTCP`,
// `SyscallConn`, and `File` each need it, thus the overlay holds none of them.
// `CloseWrite` and `CloseRead` each ride `RawSocket.shutdown`, which ends 1
// direction of the socket, exactly as the same call of Go does.
//
// A TCP listener has no `File` or `SyscallConn` method because Dart owns the descriptor.
//
// Go embeds the unexported type `conn` in `TCPConn`. The overlay names each
// method of the connection again, because a promoted method of an unexported
// type reaches no other package in the Dart output.

// A TCPConn is a connection of the network tcp. It holds each method of the
// Conn interface, and it adds the 2 copy methods of Go.
type TCPConn struct {
	c conn
}

// SetNoDelay writes the option TCP_NODELAY of the socket. Go enables the
// option by default, thus a call with true changes nothing.
func (c *TCPConn) SetNoDelay(noDelay bool) error {
	return c.c.setOption("nodelay", flagValue(noDelay))
}

// SetKeepAlive writes the option SO_KEEPALIVE of the socket.
func (c *TCPConn) SetKeepAlive(keepalive bool) error {
	if c.c.network == "" {
		return syscall.EINVAL
	}
	return c.c.setOption("keepalive", flagValue(keepalive))
}

// SetReadBuffer writes the size of the receive buffer of the socket.
func (c *TCPConn) SetReadBuffer(bytes int) error {
	return c.c.setOption("rcvbuf", bytes)
}

// SetWriteBuffer writes the size of the send buffer of the socket.
func (c *TCPConn) SetWriteBuffer(bytes int) error {
	return c.c.setOption("sndbuf", bytes)
}

// newTCPConn wraps the handle of an open TCP socket.
func newTCPConn(handle int, idle time.Duration, config KeepAliveConfig) *TCPConn {
	c := &TCPConn{c: conn{handle: handle, network: "tcp"}}
	if !config.Enable && idle >= 0 {
		config = KeepAliveConfig{Enable: true, Idle: idle}
	}
	if config.Enable {
		c.SetKeepAliveConfig(config)
	}
	return c
}

// Read reads bytes of the connection into the slice.
func (t *TCPConn) Read(b []byte) (int, error) { return t.c.Read(b) }

// Write writes the bytes of the slice to the connection.
func (t *TCPConn) Write(b []byte) (int, error) { return t.c.Write(b) }

// Close ends the connection in both directions.
func (t *TCPConn) Close() error { return t.c.Close() }

// CloseWrite ends the write side of the connection. The peer then reads the
// end of the stream, and this side still reads.
func (t *TCPConn) CloseWrite() error { return t.c.closeWrite() }

// CloseRead ends the read side of the connection. Each read of this side then
// gives io.EOF, and the system drops each byte that the peer writes after it.
func (t *TCPConn) CloseRead() error { return t.c.closeRead() }

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

// A TCPListener waits for a connection of the network tcp. `Listen` and
// `ListenTCP` each give one, thus a program that names the type in an
// assertion reads it.
type TCPListener struct {
	l listener
}

// newTCPListener wraps the handle of an open listener of the network tcp.
func newTCPListener(handle int, addr Addr) *TCPListener {
	return &TCPListener{l: listener{handle: handle, addr: addr, network: "tcp"}}
}

// Accept waits for 1 connection and gives it.
func (l *TCPListener) Accept() (Conn, error) { return l.l.Accept() }

// AcceptTCP waits for 1 connection and gives its TCP form.
func (l *TCPListener) AcceptTCP() (*TCPConn, error) {
	c, err := l.l.Accept()
	if err != nil {
		return nil, err
	}
	tc, ok := c.(*TCPConn)
	if !ok {
		c.Close()
		return nil, errors.New("accept tcp: the accept gave no TCP connection")
	}
	return tc, nil
}

// Addr gives the address of the listener.
func (l *TCPListener) Addr() Addr { return l.l.Addr() }

// Close ends the listener. Each connection of it stays open, exactly as in
// Go.
func (l *TCPListener) Close() error { return l.l.Close() }

// SetDeadline sets the accept deadline. Zero removes the deadline.
func (l *TCPListener) SetDeadline(t time.Time) error {
	if l == nil || l.l.network == "" {
		return syscall.EINVAL
	}
	return l.l.SetDeadline(t)
}

// ListenTCP opens a listener on a TCP address. The network is tcp, tcp4, or
// tcp6.
//
// A nil laddr binds each address of the host and a port that the system
// takes, exactly as in Go. A port of 0 asks the system for a free port, and
// Addr then gives the port that the system took.
func ListenTCP(network string, laddr *TCPAddr) (*TCPListener, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, UnknownNetworkError(network)
	}
	address := ":0"
	if laddr != nil {
		address = laddr.String()
	}
	l, err := Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	tl, ok := l.(*TCPListener)
	if !ok {
		l.Close()
		return nil, errors.New("listen " + network + ": the listen gave no TCP listener")
	}
	return tl, nil
}
