// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"errors"
	"runtime"
	"syscall"
	"time"
)

// Unix streams use dart:io. Unix datagrams use native nonblocking descriptors.
// Stream control messages, File and SyscallConn remain unsupported.
//
// A Unix listener has no `File` or `SyscallConn` method because Dart owns the descriptor.
// `SetUnlinkOnClose` stays out because the runtime always removes the socket file at close.
//
// Go embeds the unexported type `conn` in `UnixConn`. The overlay names each
// method again. A promoted method of an unexported type reaches no other
// package in the Dart output.

// A UnixConn is a connection of a Unix domain socket. It holds each method of
// the Conn interface.
type UnixConn struct {
	c    conn
	gram *unixgram
}

// SetReadBuffer writes the size of the receive buffer of the socket.
func (c *UnixConn) SetReadBuffer(bytes int) error {
	if c.gram != nil {
		return c.gram.buffer(true, bytes)
	}
	return c.c.setOption("rcvbuf", bytes)
}

// SetWriteBuffer writes the size of the send buffer of the socket.
func (c *UnixConn) SetWriteBuffer(bytes int) error {
	if c.gram != nil {
		return c.gram.buffer(false, bytes)
	}
	return c.c.setOption("sndbuf", bytes)
}

// newUnixConn wraps the handle of an open Unix socket.
func newUnixConn(handle int) *UnixConn {
	return &UnixConn{c: newConn(handle, "unix")}
}

// Read reads bytes of the connection into the slice.
func (u *UnixConn) Read(b []byte) (int, error) {
	if u.gram != nil {
		if u.gram.closed {
			return 0, u.gram.opError("read", ErrClosed)
		}
		if len(b) == 0 {
			return 0, nil
		}
		n, _, _, _, err := u.gram.readMsg(b, nil)
		return n, err
	}
	return u.c.Read(b)
}

// Write writes the bytes of the slice to the connection.
func (u *UnixConn) Write(b []byte) (int, error) {
	if u.gram != nil {
		n, _, err := u.gram.writeMsg(b, nil, nil, false)
		return n, err
	}
	return u.c.Write(b)
}

// Close ends the connection in both directions.
func (u *UnixConn) Close() error {
	if u.gram != nil {
		return u.gram.close()
	}
	return u.c.Close()
}

// CloseWrite ends the write side of the connection. The peer then reads the
// end of the stream, and this side still reads.
func (u *UnixConn) CloseWrite() error {
	if u.gram != nil {
		return u.gram.shutdown(1)
	}
	return u.c.closeWrite()
}

// CloseRead ends the read side of the connection. Each read of this side then
// gives io.EOF, and the system drops each byte that the peer writes after it.
func (u *UnixConn) CloseRead() error {
	if u.gram != nil {
		return u.gram.shutdown(0)
	}
	return u.c.closeRead()
}

// LocalAddr gives the address of this side of the connection.
func (u *UnixConn) LocalAddr() Addr {
	if u.gram != nil {
		return u.gram.local
	}
	return u.c.LocalAddr()
}

// RemoteAddr gives the address of the peer.
func (u *UnixConn) RemoteAddr() Addr {
	if u.gram != nil {
		return u.gram.remote.opAddr()
	}
	return u.c.RemoteAddr()
}

// SetDeadline writes the deadline of a read and of a write.
func (u *UnixConn) SetDeadline(t time.Time) error {
	if u.gram != nil {
		return u.gram.deadline(t, 3)
	}
	return u.c.SetDeadline(t)
}

// SetReadDeadline writes the deadline of a read.
func (u *UnixConn) SetReadDeadline(t time.Time) error {
	if u.gram != nil {
		return u.gram.deadline(t, 1)
	}
	return u.c.SetReadDeadline(t)
}

// SetWriteDeadline writes the deadline of a write.
func (u *UnixConn) SetWriteDeadline(t time.Time) error {
	if u.gram != nil {
		return u.gram.deadline(t, 2)
	}
	return u.c.SetWriteDeadline(t)
}

// ReadFromUnix reads bytes of the connection into the slice, and it gives the
// name of the source.
//
// Linux reports a named peer for a successful stream read. macOS reports no
// source address for a stream read.
func (u *UnixConn) ReadFromUnix(b []byte) (int, *UnixAddr, error) {
	if u.gram != nil {
		n, _, _, from, err := u.gram.readMsg(b, nil)
		return n, from, err
	}
	n, err := u.c.Read(b)
	if err == nil && runtime.GOOS == "linux" {
		addr := u.RemoteAddr().(*UnixAddr)
		if addr.Name != "" {
			return n, addr, nil
		}
	}
	return n, nil, err
}

// ReadFrom reads bytes of the connection into the slice, and it gives the
// name of the source.
func (u *UnixConn) ReadFrom(b []byte) (int, Addr, error) {
	n, addr, err := u.ReadFromUnix(b)
	if addr != nil {
		return n, addr, err
	}
	return n, nil, err
}

// DialUnix opens a Unix stream or datagram connection.
// Unix streams do not support a local dial address.
func DialUnix(network string, laddr, raddr *UnixAddr) (*UnixConn, error) {
	if network == "unixgram" {
		if laddr == nil && raddr == nil {
			return nil, &OpError{Op: "dial", Net: network, Err: errMissingAddress}
		}
		return openUnixgram("dial", laddr, raddr)
	}
	switch network {
	case "unix":
	case "unixpacket":
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
func (l *UnixListener) Accept() (Conn, error) {
	if l == nil {
		return nil, syscall.EINVAL
	}
	return l.l.Accept()
}

// AcceptUnix waits for 1 connection and gives its Unix form.
func (l *UnixListener) AcceptUnix() (*UnixConn, error) {
	if l == nil {
		return nil, syscall.EINVAL
	}
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
func (l *UnixListener) Close() error {
	if l == nil {
		return syscall.EINVAL
	}
	return l.l.Close()
}

// SetDeadline sets the accept deadline. Zero removes the deadline.
func (l *UnixListener) SetDeadline(t time.Time) error {
	if l == nil || l.l.network == "" {
		return syscall.EINVAL
	}
	return l.l.SetDeadline(t)
}

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
