// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"os"
	"strings"
	"syscall"
	"syscall/dart"
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
func newTCPConn(handle int, network string, idle time.Duration, config KeepAliveConfig) *TCPConn {
	c := &TCPConn{c: newConn(handle, network)}
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
	if t.c.network == "" {
		return 0, syscall.EINVAL
	}
	return io.Copy(onlyWriter{Writer: t}, r)
}

// WriteTo reads the connection up to its end and writes each byte to the
// writer.
func (t *TCPConn) WriteTo(w io.Writer) (int64, error) {
	if t.c.network == "" {
		return 0, syscall.EINVAL
	}
	return io.Copy(w, onlyReader{Reader: t})
}

// DialTCP opens a connection to a TCP address. The network is tcp, tcp4, or tcp6.
func DialTCP(network string, laddr, raddr *TCPAddr) (*TCPConn, error) {
	return dialTCPAddr(context.Background(), Dialer{}, network, laddr, raddr)
}

func dialTCPAddr(ctx context.Context, d Dialer, network string, local, remote *TCPAddr) (*TCPConn, error) {
	fail := func(err error) (*TCPConn, error) {
		if op, ok := err.(*OpError); ok {
			err = op.Err
		}
		op := &OpError{Op: "dial", Net: network, Err: err}
		if local != nil {
			op.Source = local
		}
		if remote != nil {
			op.Addr = remote
		}
		return nil, op
	}
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return fail(UnknownNetworkError(network))
	}
	if remote == nil {
		return fail(errMissingAddress)
	}
	if err := tcpAddrFamilyError(network, remote); err != nil {
		return fail(err)
	}
	if local != nil {
		if err := tcpAddrFamilyError(network, local); err != nil {
			return fail(err)
		}
	}
	if local != nil && (local.Port < 0 || local.Port > 65535) {
		return fail(&os.SyscallError{Syscall: "bind", Err: syscall.EINVAL})
	}
	if remote.Port < 0 || remote.Port > 65535 {
		return fail(&os.SyscallError{Syscall: "connect", Err: syscall.EINVAL})
	}
	d.LocalAddr = local
	c, err := d.DialContext(ctx, network, remote.String())
	if err != nil {
		return fail(err)
	}
	return c.(*TCPConn), nil
}

func tcpAddrFamilyError(network string, addr *TCPAddr) error {
	if len(addr.IP) == 0 {
		return nil
	}
	if network == "tcp4" && addr.IP.To4() == nil {
		return &AddrError{Err: "non-IPv4 address", Addr: addr.IP.String()}
	}
	if addr.IP.To16() == nil {
		return &AddrError{Err: "non-IPv6 address", Addr: addr.IP.String()}
	}
	return nil
}

// A TCPListener waits for a connection of the network tcp. `Listen` and
// `ListenTCP` each give one, thus a program that names the type in an
// assertion reads it.
type TCPListener struct {
	l listener
}

// newTCPListener wraps the handle of an open listener of the network tcp.
func newTCPListener(handle int, network string, addr Addr) *TCPListener {
	return &TCPListener{l: listener{handle: handle, addr: addr, network: network}}
}

// Accept waits for 1 connection and gives it.
func (l *TCPListener) Accept() (Conn, error) {
	if l == nil {
		return nil, syscall.EINVAL
	}
	return l.l.Accept()
}

// AcceptTCP waits for 1 connection and gives its TCP form.
func (l *TCPListener) AcceptTCP() (*TCPConn, error) {
	if l == nil {
		return nil, syscall.EINVAL
	}
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
func (l *TCPListener) Close() error {
	if l == nil {
		return syscall.EINVAL
	}
	return l.l.Close()
}

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
		return nil, &OpError{Op: "listen", Net: network, Addr: laddr, Err: UnknownNetworkError(network)}
	}
	if laddr == nil {
		laddr = &TCPAddr{}
	}
	fail := func(err error) (*TCPListener, error) {
		return nil, &OpError{Op: "listen", Net: network, Addr: laddr, Err: err}
	}
	if laddr.Port < 0 || laddr.Port > 65535 {
		return fail(&os.SyscallError{Syscall: "bind", Err: syscall.EINVAL})
	}
	ip := laddr.IP
	host := ""
	if network == "tcp4" {
		if len(ip) == 0 {
			ip = IPv4zero
		}
		if ip.To4() == nil {
			return fail(&AddrError{Err: "non-IPv4 address", Addr: ip.String()})
		}
		host = ip.String()
	} else if network == "tcp6" {
		if len(ip) == 0 || ip.Equal(IPv4zero) {
			ip = IPv6zero
		}
		addr, ok := netip.AddrFromSlice(ip.To16())
		if !ok {
			return fail(&AddrError{Err: "non-IPv6 address", Addr: ip.String()})
		}
		host = addr.String()
	} else if len(ip) != 0 && !(ip.To4() != nil && ip.IsUnspecified()) {
		if ip.To16() == nil {
			return fail(&AddrError{Err: "non-IPv6 address", Addr: ip.String()})
		}
		host = ip.String()
	}
	if laddr.Zone != "" && strings.Contains(host, ":") {
		host += "%" + laddr.Zone
	}
	handle, port, cause := dart.ListenTCP(host, laddr.Port, network == "tcp6")
	if cause != "" {
		return fail(errors.New(cause))
	}
	bound := dart.ListenTCPHost(handle)
	zone := ""
	if i := strings.LastIndexByte(bound, '%'); i >= 0 {
		bound, zone = bound[:i], bound[i+1:]
	}
	return newTCPListener(handle, network, &TCPAddr{IP: ParseIP(bound), Port: port, Zone: zone}), nil
}

func resolveListenTCP(network, address string) (*TCPAddr, error) {
	if address == "" {
		return &TCPAddr{}, nil
	}
	host, service, err := SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := LookupPort(network, service)
	if err != nil {
		return nil, err
	}
	zone := ""
	if i := strings.LastIndexByte(host, '%'); i >= 0 {
		host, zone = host[:i], host[i+1:]
	}
	if host == "" {
		return &TCPAddr{Port: port, Zone: zone}, nil
	}
	family := "ip"
	if network != "tcp" {
		family += network[3:]
	}
	ip := ParseIP(host)
	if ip != nil {
		if !keepIP(family, ip) {
			return nil, &AddrError{Err: "no suitable address found", Addr: host}
		}
	} else {
		ips, err := DefaultResolver.LookupIP(context.Background(), family, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, &AddrError{Err: "no suitable address found", Addr: host}
		}
		ip = ips[0]
		if network == "tcp" {
			for _, candidate := range ips {
				if candidate.To4() != nil {
					ip = candidate
					break
				}
			}
		}
	}
	return &TCPAddr{IP: ip, Port: port, Zone: zone}, nil
}
