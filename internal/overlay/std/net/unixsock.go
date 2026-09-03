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
// `WriteMsgUnix` stay out. `ReadFromUnix` and `WriteToUnix` each name a
// datagram of the domain, thus they stay out too.
//
// Go embeds the unexported type `conn` in `UnixConn`. The overlay names each
// method again. A promoted method of an unexported type reaches no other
// package in the Dart output.

// A UnixConn is a connection of a Unix domain socket. It holds each method of
// the Conn interface.
type UnixConn struct {
	c conn
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
