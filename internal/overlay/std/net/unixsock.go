//go:build dart

package net

import (
	"errors"
	"time"
)

// This file holds the connection of a Unix domain socket.
//
// `dart:io` opens such a socket through an `InternetAddress` that holds the
// type `InternetAddressType.unix`. It gives a stream socket alone, thus the
// networks unixgram and unixpacket stay out. `ListenUnixgram` needs a
// datagram socket of the domain, and Dart opens none.
//
// The socket gives no file descriptor and no half close, thus `CloseRead`,
// `CloseWrite`, `File`, `SyscallConn`, `SetReadBuffer`, and `SetWriteBuffer`
// stay out. It reads no control message, thus `ReadMsgUnix` and
// `WriteMsgUnix` stay out.
//
// `WriteTo` and `WriteToUnix` stay out too. Go writes such a datagram with
// sendto, and a socket that a dial connected gives ErrWriteToConnected. A
// socket that an accept gave holds no such mark, and the answer of the system
// then differs: macOS writes the bytes and Linux gives EOPNOTSUPP. Dart holds
// no sendto, thus the overlay can match neither.
//
// A listener of the domain holds the same 2 limits. `File` and `SyscallConn`
// each need the descriptor, and `SetDeadline` needs an accept with a deadline
// that the runtime does not give. `SetUnlinkOnClose` stays out because the
// runtime always removes the file of the socket at the close.
//
// Go embeds the unexported type `conn` in `UnixConn`. The overlay names each
// method again. A promoted method of an unexported type reaches no other
// package in the Dart output.

// A UnixConn is a connection of a Unix domain socket. It holds each method of
// the Conn interface.
type UnixConn struct {
	c conn
}

// SetReadBuffer writes the size of the receive buffer of the socket.
func (c *UnixConn) SetReadBuffer(bytes int) error {
	return c.c.setOption("rcvbuf", bytes)
}

// SetWriteBuffer writes the size of the send buffer of the socket.
func (c *UnixConn) SetWriteBuffer(bytes int) error {
	return c.c.setOption("sndbuf", bytes)
}

// newUnixConn wraps the handle of an open Unix socket.
func newUnixConn(handle int) *UnixConn {
	return &UnixConn{c: conn{handle: handle, network: "unix"}}
}

// Read reads bytes of the connection into the slice.
func (u *UnixConn) Read(b []byte) (int, error) { return u.c.Read(b) }

// Write writes the bytes of the slice to the connection.
func (u *UnixConn) Write(b []byte) (int, error) { return u.c.Write(b) }

// Close ends the connection in both directions.
func (u *UnixConn) Close() error { return u.c.Close() }

// LocalAddr gives the address of this side of the connection.
func (u *UnixConn) LocalAddr() Addr { return u.c.LocalAddr() }

// RemoteAddr gives the address of the peer.
func (u *UnixConn) RemoteAddr() Addr { return u.c.RemoteAddr() }

// SetDeadline writes the deadline of a read and of a write.
func (u *UnixConn) SetDeadline(t time.Time) error { return u.c.SetDeadline(t) }

// SetReadDeadline writes the deadline of a read.
func (u *UnixConn) SetReadDeadline(t time.Time) error { return u.c.SetReadDeadline(t) }

// SetWriteDeadline writes the deadline of a write.
func (u *UnixConn) SetWriteDeadline(t time.Time) error { return u.c.SetWriteDeadline(t) }

// ReadFromUnix reads bytes of the connection into the slice, and it gives the
// name of the source.
//
// Dart opens a stream socket of the domain alone. Such a socket holds a peer,
// and the system reports an empty name for the source. Thus the address of
// the result is always nil, exactly as in Go for a connected socket.
func (u *UnixConn) ReadFromUnix(b []byte) (int, *UnixAddr, error) {
	n, err := u.c.Read(b)
	return n, nil, err
}

// ReadFrom reads bytes of the connection into the slice, and it gives the
// name of the source. The address of the result is always nil. ReadFromUnix
// holds the cause.
func (u *UnixConn) ReadFrom(b []byte) (int, Addr, error) {
	n, err := u.c.Read(b)
	return n, nil, err
}

// DialUnix opens a connection to a Unix domain socket. The network is unix.
//
// Dart opens a stream socket of the domain alone, thus the networks unixgram
// and unixpacket give an error. Dart binds no local address of a dial, thus a
// laddr that is not nil gives an error too.
func DialUnix(network string, laddr, raddr *UnixAddr) (*UnixConn, error) {
	switch network {
	case "unix":
	case "unixgram", "unixpacket":
		return nil, errors.New("dial " + network +
			": dart:io opens a stream socket of the Unix domain alone")
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
	c, err := Dial("unix", raddr.Name)
	if err != nil {
		return nil, err
	}
	uc, ok := c.(*UnixConn)
	if !ok {
		c.Close()
		return nil, errors.New("dial " + network + ": the dial gave no Unix connection")
	}
	return uc, nil
}

// A UnixListener waits for a connection of a Unix domain socket. `Listen` and
// `ListenUnix` each give one, thus a program that names the type in an
// assertion reads it.
type UnixListener struct {
	l listener
}

// newUnixListener wraps the handle of an open listener of the Unix domain.
func newUnixListener(handle int, addr Addr) *UnixListener {
	return &UnixListener{l: listener{handle: handle, addr: addr, network: "unix"}}
}

// Accept waits for 1 connection and gives it.
func (l *UnixListener) Accept() (Conn, error) { return l.l.Accept() }

// AcceptUnix waits for 1 connection and gives its Unix form.
func (l *UnixListener) AcceptUnix() (*UnixConn, error) {
	c, err := l.l.Accept()
	if err != nil {
		return nil, err
	}
	uc, ok := c.(*UnixConn)
	if !ok {
		c.Close()
		return nil, errors.New("accept unix: the accept gave no Unix connection")
	}
	return uc, nil
}

// Addr gives the address of the listener.
func (l *UnixListener) Addr() Addr { return l.l.Addr() }

// Close ends the listener, and it removes the file of the socket. Each
// connection of the listener stays open, exactly as in Go.
func (l *UnixListener) Close() error { return l.l.Close() }

// ListenUnix opens a listener on a Unix domain socket. The network is unix.
//
// Dart opens a stream socket of the domain alone, thus the networks unixgram
// and unixpacket give an error.
func ListenUnix(network string, laddr *UnixAddr) (*UnixListener, error) {
	switch network {
	case "unix":
	case "unixgram", "unixpacket":
		return nil, errors.New("listen " + network +
			": dart:io opens a stream socket of the Unix domain alone")
	default:
		return nil, UnknownNetworkError(network)
	}
	if laddr == nil || laddr.Name == "" {
		return nil, errors.New("listen " + network + ": the address is missing")
	}
	l, err := Listen("unix", laddr.Name)
	if err != nil {
		return nil, err
	}
	ul, ok := l.(*UnixListener)
	if !ok {
		l.Close()
		return nil, errors.New("listen " + network + ": the listen gave no Unix listener")
	}
	return ul, nil
}
