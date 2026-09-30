// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Package net gives a TCP client, a TCP server, a UDP socket, and a Unix
// socket.
//
// An address holds a numeric host or a host name that Dart resolves. There is
// no TLS, no IPv6 form with brackets, and no address type of Go.
// `docs/parity.md` lists each gap.
package net

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"syscall/dart"
	"time"
)

// Conn is a stream between 2 programs.
type Conn interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
	LocalAddr() Addr
	RemoteAddr() Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// PacketConn reads and writes 1 datagram at a time.
type PacketConn interface {
	ReadFrom(b []byte) (int, Addr, error)
	WriteTo(b []byte, addr Addr) (int, error)
	Close() error
	LocalAddr() Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// Listener waits for a connection of a client.
type Listener interface {
	Accept() (Conn, error)
	Close() error
	Addr() Addr
}

// Addr is the address of a listener or of a datagram socket.
type Addr interface {
	Network() string
	String() string
}

// Error is the error of a network operation. A deadline that passed gives an
// error that reports true.
//
// Go marks Temporary as deprecated, and it keeps the method in the interface.
type Error interface {
	error
	Timeout() bool
	Temporary() bool
}

// OpError holds the error of a network operation. It names the operation, the
// network, the 2 addresses, and the cause.
//
// The overlay holds no system call error, thus Timeout and Temporary read the
// cause alone. Go also reads an os.SyscallError inside the cause, and this
// package builds no such error.
type OpError struct {
	// Op names the operation, such as "read", "write", or "dial".
	Op string

	// Net names the network, such as "tcp" or "udp6".
	Net string

	// Source holds the local address of an operation over a connection.
	Source Addr

	// Addr holds the address of the operation. A local operation names the
	// local endpoint, and a remote operation names the peer.
	Addr Addr

	// Err holds the cause. Error stops the program for a nil cause, exactly
	// as in Go.
	Err error
}

func (e *OpError) Unwrap() error { return e.Err }

func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}
	s := e.Op
	if e.Net != "" {
		s += " " + e.Net
	}
	if e.Source != nil {
		s += " " + e.Source.String()
	}
	if e.Addr != nil {
		if e.Source != nil {
			s += "->"
		} else {
			s += " "
		}
		s += e.Addr.String()
	}
	s += ": " + e.Err.Error()
	return s
}

// Timeout reports a cause that names a deadline.
func (e *OpError) Timeout() bool {
	t, ok := e.Err.(interface{ Timeout() bool })
	return ok && t.Timeout()
}

// Temporary reports a cause that a second try may pass.
func (e *OpError) Temporary() bool {
	t, ok := e.Err.(interface{ Temporary() bool })
	return ok && t.Temporary()
}

// deadlineNanos gives the count of nanoseconds of a deadline. The zero Time
// holds no deadline, exactly as in Go.
func deadlineNanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// conn holds a stream connection of the network tcp or unix.
type conn struct {
	handle  int
	network string
	local   Addr
	remote  Addr
	read    int64
	write   int64
}

// LocalAddr gives the address of this side of the connection.
func (c *conn) LocalAddr() Addr { return c.local }

// RemoteAddr gives the address of the peer.
func (c *conn) RemoteAddr() Addr { return c.remote }

func newConn(handle int, network string) conn {
	lHost, lPort, rHost, rPort, _ := dart.ConnAddrs(handle)
	c := conn{handle: handle, network: network}
	if network == "unix" {
		c.local = &UnixAddr{Name: lHost, Net: "unix"}
		c.remote = &UnixAddr{Name: rHost, Net: "unix"}
	} else {
		c.local = tcpSocketAddrOf(lHost, lPort)
		c.remote = tcpSocketAddrOf(rHost, rPort)
	}
	return c
}

func (c *conn) Read(b []byte) (int, error) {
	if c.network == "" {
		return 0, syscall.EINVAL
	}
	n, cause, code := dart.ConnRead(c.handle, b)
	if cause != "" {
		return n, c.opError("read", socketSyscallCause("read", cause, code))
	}
	if n == 0 && len(b) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (c *conn) Write(b []byte) (int, error) {
	if c.network == "" {
		return 0, syscall.EINVAL
	}
	n, cause, code := dart.ConnWrite(c.handle, b)
	if cause != "" {
		return n, c.opError("write", socketSyscallCause("write", cause, code))
	}
	return n, nil
}

// setOption writes 1 socket option of the connection. A flag takes the value
// 0 or 1.
func (c *conn) setOption(name string, value int) error {
	if c.network == "" {
		return syscall.EINVAL
	}
	if cause, code := dart.ConnOption(c.handle, name, value); cause != "" {
		if name == "rcvbuf" || name == "sndbuf" {
			return &OpError{Op: "set", Net: c.network, Addr: c.local, Err: socketSyscallCause("setsockopt", cause, code)}
		}
		return c.opError("set", socketSyscallCause("setsockopt", cause, code))
	}
	return nil
}

// flagValue gives the value of a socket option that holds a flag.
func flagValue(on bool) int {
	if on {
		return 1
	}
	return 0
}

func (c *conn) Close() error {
	if c.network == "" {
		return syscall.EINVAL
	}
	handle := c.handle
	c.handle = -1
	if cause := dart.ConnClose(handle); cause != "" {
		return c.opError("close", socketCause(cause))
	}
	return nil
}

func socketSyscallCause(op, cause string, code int) error {
	if code == 0 {
		return socketCause(cause)
	}
	if runtime.GOOS == "windows" {
		if op == "read" {
			op = "wsarecv"
		} else if op == "recvfrom" {
			op = "wsarecvfrom"
		} else if op == "sendto" {
			op = "wsasendto"
		} else if op == "write" {
			op = "wsasend"
		}
	}
	return &os.SyscallError{Syscall: op, Err: syscall.Errno(code)}
}

func socketCause(cause string) error {
	if cause == "i/o timeout" {
		return os.ErrDeadlineExceeded
	}
	if cause == ErrClosed.Error() {
		return ErrClosed
	}
	return errors.New(cause)
}

func (c *conn) opError(op string, cause error) error {
	return &OpError{Op: op, Net: c.network, Source: c.local, Addr: c.remote, Err: cause}
}

// connCloseWrite ends the write side of a connection. The result is empty on
// success, and it names the cause on a failure.
//
//gart:dart package:goruntime/goruntime.dart $connCloseWrite
func connCloseWrite(handle int) string

// connCloseRead ends the read side of a connection. The result is empty on
// success, and it names the cause on a failure.
//
//gart:dart package:goruntime/goruntime.dart $connCloseRead
func connCloseRead(handle int) string

// closeWrite ends the write side of the connection. The peer then reads the
// end of the stream, and this side still reads. A write after it gives an
// error.
//
// A plain connection rides a RawSocket of dart:io, thus the runtime writes the
// shutdown of the send direction. Go writes the same call of the system.
func (c *conn) closeWrite() error {
	if c.network == "" {
		return syscall.EINVAL
	}
	if cause := connCloseWrite(c.handle); cause != "" {
		return c.opError("close", socketCause(cause))
	}
	return nil
}

// closeRead ends the read side of the connection. Each read of this side then
// gives io.EOF, and the system drops each byte that arrives after it.
//
// The runtime writes the shutdown of the receive direction of the RawSocket,
// exactly as CloseRead of Go does. A peer of a BSD system reads a reset for
// the bytes that it wrote after the call, because the system holds that rule.
func (c *conn) closeRead() error {
	if c.network == "" {
		return syscall.EINVAL
	}
	if cause := connCloseRead(c.handle); cause != "" {
		return c.opError("close", socketCause(cause))
	}
	return nil
}

func (c *conn) SetDeadline(t time.Time) error {
	c.read = deadlineNanos(t)
	c.write = c.read
	return c.apply()
}

func (c *conn) SetReadDeadline(t time.Time) error {
	c.read = deadlineNanos(t)
	return c.apply()
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	c.write = deadlineNanos(t)
	return c.apply()
}

func (c *conn) apply() error {
	if c.network == "" {
		return syscall.EINVAL
	}
	if cause := dart.ConnDeadline(c.handle, c.read, c.write); cause != "" {
		return &OpError{Op: "set", Net: c.network, Addr: c.local, Err: socketCause(cause)}
	}
	return nil
}

// packet holds a datagram socket.
type packet struct {
	network string
	remote  Addr
	handle  int
	local   *UDPAddr
	read    int64
	write   int64
}

func (p *packet) ReadFrom(b []byte) (int, Addr, error) {
	return p.readFrom(b, "recvfrom")
}

func (p *packet) readFrom(b []byte, call string) (int, Addr, error) {
	if p == nil {
		return 0, nil, syscall.EINVAL
	}
	n, host, port, cause, code := dart.PacketRead(p.handle, b)
	if cause != "" {
		return n, nil, &OpError{Op: "read", Net: p.network, Source: p.local, Addr: p.remote, Err: socketSyscallCause(call, cause, code)}
	}
	return n, udpAddrOf(host, port), nil
}

func (p *packet) Close() error {
	handle := p.handle
	p.handle = -1
	if cause := dart.PacketClose(handle); cause != "" {
		return &OpError{Op: "close", Net: p.network, Source: p.local, Addr: p.remote, Err: socketCause(cause)}
	}
	return nil
}

func (p *packet) LocalAddr() Addr {
	if p == nil {
		return nil
	}
	return p.local
}

func (p *packet) SetDeadline(t time.Time) error {
	if p == nil {
		return syscall.EINVAL
	}
	p.read = deadlineNanos(t)
	p.write = p.read
	return p.apply()
}

func (p *packet) SetReadDeadline(t time.Time) error {
	if p == nil {
		return syscall.EINVAL
	}
	p.read = deadlineNanos(t)
	return p.apply()
}

func (p *packet) SetWriteDeadline(t time.Time) error {
	if p == nil {
		return syscall.EINVAL
	}
	p.write = deadlineNanos(t)
	return p.apply()
}

func (p *packet) apply() error {
	if cause := dart.PacketDeadline(p.handle, p.read, p.write); cause != "" {
		return &OpError{Op: "set", Net: p.network, Addr: p.local, Err: socketCause(cause)}
	}
	return nil
}

type listener struct {
	handle          int
	addr            Addr
	network         string
	keepAlive       time.Duration
	keepAliveConfig KeepAliveConfig
}

func (l *listener) Accept() (Conn, error) {
	if l.network == "" {
		return nil, syscall.EINVAL
	}
	handle, cause := dart.Accept(l.handle)
	if cause != "" {
		err := error(errors.New(cause))
		if cause == "i/o timeout" {
			err = os.ErrDeadlineExceeded
		} else if cause == errClosed.Error() {
			err = ErrClosed
		}
		return nil, &OpError{Op: "accept", Net: l.network, Addr: l.addr, Err: err}
	}
	if strings.HasPrefix(l.network, "tcp") {
		return newTCPConn(handle, l.network, l.keepAlive, l.keepAliveConfig), nil
	}
	return newUnixConn(handle), nil
}

func (l *listener) Close() error {
	if l.network == "" {
		return syscall.EINVAL
	}
	if cause := dart.ListenClose(l.handle); cause != "" {
		return &OpError{Op: "close", Net: l.network, Addr: l.addr, Err: socketCause(cause)}
	}
	l.handle = -1
	return nil
}

func (l *listener) Addr() Addr { return l.addr }

func (l *listener) SetDeadline(t time.Time) error {
	if cause := dart.ListenDeadline(l.handle, deadlineNanos(t)); cause != "" {
		if cause == errClosed.Error() {
			return ErrClosed
		}
		return errors.New(cause)
	}
	return nil
}

// Dial opens a connection to an address. The network is tcp, udp, or unix.
//
// A host name can hold more than 1 address. The dial reads the whole list and
// it tries each address in turn, exactly as Go does.
func Dial(network, address string) (Conn, error) {
	var d Dialer
	return d.DialContext(context.Background(), network, address)
}

// dialOne opens 1 connection to 1 address. The host of the address holds a
// numeric form, or the name of a socket of the Unix domain.
func (d *Dialer) dialOne(network, address string) (Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		host, port, err := hostPort(address)
		if err != nil {
			return nil, err
		}
		remote := tcpAddrOf(host, port)
		var local *TCPAddr
		if d.LocalAddr != nil {
			local = d.LocalAddr.(*TCPAddr)
		}
		fail := func(err error) (Conn, error) {
			op := &OpError{Op: "dial", Net: network, Addr: remote, Err: err}
			if local != nil {
				op.Source = local
			}
			return nil, op
		}
		if host == "" {
			host = "127.0.0.1"
			if network == "tcp6" || local != nil && local.IP.To4() == nil && len(local.IP) != 0 {
				host = "::1"
			}
		}
		localHost, localPort := "", 0
		if local != nil {
			if err := tcpAddrFamilyError(network, local); err != nil {
				return fail(err)
			}
			if local.Port < 0 || local.Port > 65535 {
				return fail(&os.SyscallError{Syscall: "bind", Err: syscall.EINVAL})
			}
			localPort = local.Port
			ip := local.IP
			if len(ip) == 0 || ip.IsUnspecified() {
				ip = IPv4zero
				if network == "tcp6" || strings.Contains(host, ":") {
					ip = IPv6zero
				}
			}
			localHost = ip.String()
			if local.Zone != "" && ip.To4() == nil {
				localHost += "%" + local.Zone
			}
		}
		handle, cause, code, call := dart.DialTCP(host, port, localHost, localPort)
		if cause != "" {
			return fail(socketSyscallCause(call, cause, code))
		}
		return newTCPConn(handle, network, d.KeepAlive, d.KeepAliveConfig), nil
	case "unix":
		handle, cause := dart.DialUnix(address)
		if cause != "" {
			return nil, errors.New("dial unix " + address + ": " + cause)
		}
		return newUnixConn(handle), nil
	case "udp", "udp4", "udp6":
		host, port, err := hostPort(address)
		if err != nil {
			return nil, err
		}
		var local *UDPAddr
		if d.LocalAddr != nil {
			local = d.LocalAddr.(*UDPAddr)
		}
		if host == "" {
			host = "127.0.0.1"
			if network == "udp6" || local != nil && local.IP.To4() == nil && len(local.IP) != 0 {
				host = "::1"
			}
		}
		return DialUDP(network, local, udpAddrOf(host, port))
	}
	return nil, errors.New("dial " + network + ": the subset holds tcp, udp, and unix")
}

// Listen opens a listener on an address. A port of 0 asks the system for a
// free port, and Addr then gives the port that the system took.
//
// The networks tcp, tcp4, and tcp6 give a *TCPListener.
// The network unix gives a *UnixListener.
func Listen(network, address string) (Listener, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		addr, err := resolveListenTCP(network, address)
		if err != nil {
			return nil, &OpError{Op: "listen", Net: network, Err: err}
		}
		l, err := ListenTCP(network, addr)
		if err != nil {
			return nil, err
		}
		return l, nil
	case "unix":
		handle, cause := dart.ListenUnix(address)
		if cause != "" {
			return nil, errors.New("listen unix " + address + ": " + cause)
		}
		return newUnixListener(handle, &UnixAddr{Name: address, Net: "unix"}), nil
	}
	return nil, errors.New("listen " + network + ": the subset holds tcp and unix")
}

// ListenPacket opens a datagram socket on an address. A port of 0 asks the
// system for a free port, and LocalAddr then gives the port that the system
// took.
func ListenPacket(network, address string) (PacketConn, error) {
	switch network {
	case "udp", "udp4", "udp6":
	default:
		return nil, &OpError{Op: "listen", Net: network, Err: UnknownNetworkError(network)}
	}
	addr, err := ResolveUDPAddr(network, address)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: network, Err: err}
	}
	conn, err := ListenUDP(network, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// A Dialer opens a connection with a timeout, with a deadline, or with a
// context.
//
// `Control` and `ControlContext` each take a `syscall.RawConn`,
// and `dart:io` gives no file descriptor.
// `MultipathTCP` and `SetMultipathTCP` require a transport that Dart does not expose.
// The overlay omits these controls.
//
// `DialIP` stays out too. It gives an `*IPConn`, and `dart:io` opens no raw
// socket.
type Dialer struct {
	// DualStack has no effect. Fast fallback is enabled by default.
	//
	// Deprecated: Set FallbackDelay to a negative value to disable fast fallback.
	DualStack bool
	// FallbackDelay sets the delay before the other address family starts.
	// A zero value selects 300 ms. A negative value disables fast fallback.
	FallbackDelay time.Duration
	// LocalAddr selects the local TCP or UDP address. Other networks return an error.
	LocalAddr       Addr
	KeepAlive       time.Duration
	KeepAliveConfig KeepAliveConfig
	// Timeout ends the wait of a dial after the duration. A value of 0 holds
	// no limit.
	Timeout time.Duration
	// Deadline is the point in time after which a dial fails. Timeout can end
	// the wait earlier. The zero Time holds no deadline.
	Deadline time.Time
	// Cancel ends a dial when the channel closes. A nil channel ends no dial.
	//
	// Deprecated: Use DialContext instead.
	Cancel <-chan struct{}
	// Resolver reads the host name of the address. A nil field gives the name
	// to the runtime, which reads the resolver of the platform.
	Resolver *Resolver
}

// canceledError reports an operation that a cancel ended. Go keeps the same
// text and the same Is method, thus errors.Is(err, context.Canceled) gives
// true.
type canceledError struct{}

func (canceledError) Error() string { return "operation was canceled" }

func (canceledError) Is(err error) bool { return err == context.Canceled }

type timeoutError struct{}

func (*timeoutError) Error() string { return "i/o timeout" }

func (*timeoutError) Timeout() bool { return true }

func (*timeoutError) Temporary() bool { return true }

func (*timeoutError) Is(err error) bool { return err == context.DeadlineExceeded }

// The 2 errors of a context that ended.
var (
	errCanceled error = canceledError{}
	errTimeout  error = &timeoutError{}
)

// mapErr gives the error of this package for an error of a context, exactly
// as Go does.
func mapErr(err error) error {
	switch err {
	case context.Canceled:
		return errCanceled
	case context.DeadlineExceeded:
		return errTimeout
	}
	return err
}

// dialResult holds the 2 results of a dial that runs on a goroutine.
type dialResult struct {
	conn  Conn
	cause error
}

// Dial opens a connection with the timeout of the dialer.
func (d *Dialer) Dial(network, address string) (Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialTimeout opens a connection and ends the wait after the duration.
func DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	d := Dialer{Timeout: timeout}
	return d.Dial(network, address)
}

// DialContext opens a connection and ends the wait when the context ends.
//
// The dial runs on a goroutine, thus the select takes the first of the 3
// events: the dial, the end of the context, and the close of the Cancel
// channel. A dial that ends after the wait closes its own connection.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (Conn, error) {
	if d.Timeout != 0 {
		inner, cancel := context.WithTimeout(ctx, d.Timeout)
		defer cancel()
		ctx = inner
	}
	if !d.Deadline.IsZero() {
		inner, cancel := context.WithDeadline(ctx, d.Deadline)
		defer cancel()
		ctx = inner
	}
	list, err := d.addrList(ctx, network, address)
	if err != nil {
		if network == "tcp" || network == "tcp4" || network == "tcp6" {
			return nil, &OpError{Op: "dial", Net: network, Err: err}
		}
		return nil, err
	}
	if d.LocalAddr != nil {
		var localIP IP
		var valid bool
		switch network {
		case "tcp", "tcp4", "tcp6":
			local, ok := d.LocalAddr.(*TCPAddr)
			valid = ok
			if local != nil {
				localIP = local.IP
			}
		case "udp", "udp4", "udp6":
			local, ok := d.LocalAddr.(*UDPAddr)
			valid = ok
			if local != nil {
				localIP = local.IP
			}
		default:
			return nil, &OpError{Op: "dial", Net: network, Err: errors.New("local addresses for this network are not supported")}
		}
		if !valid {
			return nil, &OpError{Op: "dial", Net: network, Err: &AddrError{Err: "mismatched local address type", Addr: d.LocalAddr.String()}}
		}
		if len(localIP) != 0 && !localIP.IsUnspecified() {
			filtered := list[:0]
			for _, address := range list {
				host, _, _ := SplitHostPort(address)
				remote := tcpAddrOf(host, 0)
				if len(remote.IP) == 0 || remote.IP.IsUnspecified() || (remote.IP.To4() == nil) == (localIP.To4() == nil) {
					filtered = append(filtered, address)
				}
			}
			list = filtered
			if len(list) == 0 {
				return nil, &OpError{Op: "dial", Net: network, Err: &AddrError{Err: "no suitable address found", Addr: d.LocalAddr.String()}}
			}
		}
	}
	if network == "tcp" || network == "tcp4" || network == "tcp6" {
		addrs := make(addrList, 0, len(list))
		for _, address := range list {
			host, port, _ := hostPort(address)
			addrs = append(addrs, tcpAddrOf(host, port))
		}
		sd := &sysDialer{Dialer: *d, network: network, address: address}
		primaries, fallbacks := addrs, addrList(nil)
		if d.dualStack() && network == "tcp" {
			primaries, fallbacks = addrs.partition(isIPv4)
		}
		return sd.dialParallel(ctx, primaries, fallbacks)
	}
	trace := traceOf(ctx)
	var first error
	for _, one := range list {
		traceConnectStart(trace, network, one)
		c, cause := d.dialOneAddr(ctx, network, one)
		traceConnectDone(trace, network, one, cause)
		if cause == nil {
			return c, nil
		}
		if first == nil {
			first = cause
		}
		if ctx.Err() != nil {
			break
		}
	}
	if first == nil {
		first = &OpError{Op: "dial", Net: network, Err: errMissingAddress}
	}
	return nil, first
}

// errMissingAddress names a dial that holds no address. Go gives the same
// text.
var errMissingAddress = errors.New("missing address")

// addrList gives each address that a dial tries, in the order of the
// resolver. Go resolves the host of a dial 1 time and it then tries each
// address of the list.
//
// A socket of the Unix domain names a path, and a numeric host names itself,
// thus each of the 2 gives a list of 1 item with no lookup.
func (d *Dialer) addrList(ctx context.Context, network, address string) ([]string, error) {
	if network == "unix" || network == "unixgram" || network == "unixpacket" {
		return []string{address}, nil
	}
	host, port, err := SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if network == "tcp" || network == "tcp4" || network == "tcp6" || network == "udp" || network == "udp4" || network == "udp6" {
		number, err := d.resolver().LookupPort(ctx, network, port)
		if err != nil {
			return nil, err
		}
		port = strconv.Itoa(number)
		address = JoinHostPort(host, port)
	}
	family := "ip"
	if network == "tcp4" || network == "tcp6" || network == "udp4" || network == "udp6" {
		family += network[3:]
	}
	if host == "" {
		return []string{address}, nil
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		ip := IP(addr.AsSlice())
		if (network == "tcp4" || network == "udp4") && ip.Equal(IPv6unspecified) {
			return []string{JoinHostPort(IPv4zero.String(), port)}, nil
		}
		if !keepIP(family, ip) {
			return nil, &AddrError{Err: "no suitable address found", Addr: host}
		}
		return []string{address}, nil
	}
	ips, err := d.resolver().LookupIP(ctx, family, host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, JoinHostPort(ip.String(), port))
	}
	return out, nil
}

// resolver gives the resolver of the dialer, or the resolver of the package.
func (d *Dialer) resolver() *Resolver {
	if d.Resolver != nil {
		return d.Resolver
	}
	return DefaultResolver
}

// dialOneAddr opens 1 connection and it ends the wait when the context ends.
//
// A context that holds no end and a dialer that holds no Cancel need no
// second goroutine, thus a plain dial costs 1 call.
func (d *Dialer) dialOneAddr(ctx context.Context, network, address string) (Conn, error) {
	fail := func(err error) (Conn, error) {
		if network == "tcp" || network == "tcp4" || network == "tcp6" {
			host, port, _ := hostPort(address)
			return nil, &OpError{Op: "dial", Net: network, Source: d.LocalAddr, Addr: tcpAddrOf(host, port), Err: err}
		}
		if network == "udp" || network == "udp4" || network == "udp6" {
			host, port, _ := hostPort(address)
			return nil, &OpError{Op: "dial", Net: network, Source: d.LocalAddr, Addr: udpAddrOf(host, port), Err: err}
		}
		return nil, err
	}
	select {
	case <-ctx.Done():
		return fail(mapErr(ctx.Err()))
	case <-d.Cancel:
		return fail(errCanceled)
	default:
	}
	if ctx.Done() == nil && d.Cancel == nil {
		return d.dialOne(network, address)
	}
	done := make(chan dialResult, 1)
	go func() {
		c, cause := d.dialOne(network, address)
		done <- dialResult{conn: c, cause: cause}
	}()
	select {
	case r := <-done:
		return r.conn, r.cause
	case <-ctx.Done():
		go closeLate(done)
		return fail(mapErr(ctx.Err()))
	case <-d.Cancel:
		go closeLate(done)
		return fail(errCanceled)
	}
}

// DialTCP opens a TCP connection with the context and explicit local address.
func (d *Dialer) DialTCP(ctx context.Context, network string, laddr, raddr netip.AddrPort) (*TCPConn, error) {
	return dialTCPAddr(ctx, *d, network, TCPAddrFromAddrPort(laddr), TCPAddrFromAddrPort(raddr))
}

// DialUDP opens a UDP connection with the context and explicit local address.
func (d *Dialer) DialUDP(ctx context.Context, network string, laddr, raddr netip.AddrPort) (*UDPConn, error) {
	if ctx == nil {
		panic("nil context")
	}
	dialer := *d
	if dialer.Timeout != 0 {
		inner, cancel := context.WithTimeout(ctx, dialer.Timeout)
		defer cancel()
		ctx = inner
	}
	if !dialer.Deadline.IsZero() {
		inner, cancel := context.WithDeadline(ctx, dialer.Deadline)
		defer cancel()
		ctx = inner
	}
	if dialer.Cancel != nil {
		inner, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-dialer.Cancel:
				cancel()
			case <-inner.Done():
			}
		}()
		ctx = inner
	}
	local, remote := UDPAddrFromAddrPort(laddr), UDPAddrFromAddrPort(raddr)
	// Native UDP bind and connect errors precede context cancellation.
	c, err := DialUDP(network, local, remote)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		c.Close()
		return nil, &OpError{Op: "dial", Net: network, Source: local, Addr: remote, Err: mapErr(err)}
	}
	return c, nil
}

// DialUnix opens a connection of the Unix domain with the context. The
// network is unix.
//
// Dart opens a stream socket of the domain alone, thus the networks unixgram
// and unixpacket give an error. Dart binds no local address of a dial, thus a
// laddr that is not nil gives an error too.
func (d *Dialer) DialUnix(ctx context.Context, network string, laddr, raddr *UnixAddr) (*UnixConn, error) {
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
	dialer := *d
	dialer.LocalAddr = nil
	c, err := dialer.DialContext(ctx, "unix", raddr.Name)
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

// closeLate closes a connection of a dial that ended after the caller left.
func closeLate(done chan dialResult) {
	if r := <-done; r.conn != nil {
		r.conn.Close()
	}
}

// The 2 texts that SplitHostPort reports. Go holds the same 2.
const (
	errMissingPort  = "missing port in address"
	errTooManyColon = "too many colons in address"
)

// SplitHostPort reads the host and the port of an address. An IPv6 host stands
// in brackets, exactly as in Go, thus `[::1]:80` gives the host `::1`.
func SplitHostPort(address string) (string, string, error) {
	fail := func(why string) (string, string, error) {
		return "", "", &AddrError{Err: why, Addr: address}
	}
	i := strings.LastIndex(address, ":")
	if i < 0 {
		return fail(errMissingPort)
	}
	var host string
	j, k := 0, 0
	if len(address) > 0 && address[0] == '[' {
		end := strings.IndexByte(address, ']')
		if end < 0 {
			return fail("missing ']' in address")
		}
		switch end + 1 {
		case len(address):
			return fail(errMissingPort)
		case i:
			// The bracket ends right in front of the colon of the port.
		default:
			if address[end+1] == ':' {
				return fail(errTooManyColon)
			}
			return fail(errMissingPort)
		}
		host = address[1:end]
		j, k = 1, end+1
	} else {
		host = address[:i]
		if strings.IndexByte(host, ':') >= 0 {
			return fail(errTooManyColon)
		}
	}
	if strings.IndexByte(address[j:], '[') >= 0 {
		return fail("unexpected '[' in address")
	}
	if strings.IndexByte(address[k:], ']') >= 0 {
		return fail("unexpected ']' in address")
	}
	return host, address[i+1:], nil
}

// JoinHostPort writes an address from a host and a port.
func JoinHostPort(host, port string) string {
	// A host that holds a colon is an IPv6 address, and it takes brackets.
	if strings.IndexByte(host, ':') >= 0 {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}

func hostPort(address string) (string, int, error) {
	host, text, err := SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(text)
	if err != nil {
		return "", 0, errors.New("address " + address + ": the port holds no number")
	}
	return host, port, nil
}
