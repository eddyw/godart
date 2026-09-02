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
	"strconv"
	"strings"
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

// timeoutError reports a deadline that passed. Go gives an error that holds a
// Timeout method, and a caller reads that method.
type timeoutError struct {
	op string
}

func (e *timeoutError) Error() string { return e.op + ": i/o timeout" }

func (e *timeoutError) Timeout() bool { return true }

func (e *timeoutError) Temporary() bool { return true }

// netError builds the error of a failed operation.
func netError(op, cause string) error {
	if cause == "i/o timeout" {
		return &timeoutError{op: op}
	}
	return errors.New(op + ": " + cause)
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
	read    int64
	write   int64
}

// LocalAddr gives the address of this side of the connection.
func (c *conn) LocalAddr() Addr { return c.addrOf(true) }

// RemoteAddr gives the address of the peer.
func (c *conn) RemoteAddr() Addr { return c.addrOf(false) }

// addrOf reads the 2 addresses of the socket. A closed connection gives an
// address with no IP and a port of 0.
func (c *conn) addrOf(local bool) Addr {
	lHost, lPort, rHost, rPort, cause := dart.ConnAddrs(c.handle)
	if cause != "" {
		lHost, lPort, rHost, rPort = "", 0, "", 0
	}
	if c.network == "unix" {
		return &UnixAddr{Name: lHost, Net: "unix"}
	}
	if local {
		return tcpAddrOf(lHost, lPort)
	}
	return tcpAddrOf(rHost, rPort)
}

func (c *conn) Read(b []byte) (int, error) {
	n, cause := dart.ConnRead(c.handle, b)
	if cause != "" {
		return n, netError("read "+c.network, cause)
	}
	if n == 0 && len(b) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (c *conn) Write(b []byte) (int, error) {
	n, cause := dart.ConnWrite(c.handle, b)
	if cause != "" {
		return n, netError("write "+c.network, cause)
	}
	return n, nil
}

func (c *conn) Close() error {
	if cause := dart.ConnClose(c.handle); cause != "" {
		return netError("close "+c.network, cause)
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
	if cause := dart.ConnDeadline(c.handle, c.read, c.write); cause != "" {
		return netError("set deadline "+c.network, cause)
	}
	return nil
}

// packet holds a datagram socket.
type packet struct {
	handle int
	local  *UDPAddr
	read   int64
	write  int64
}

func (p *packet) ReadFrom(b []byte) (int, Addr, error) {
	n, host, port, cause := dart.PacketRead(p.handle, b)
	if cause != "" {
		return n, nil, netError("read udp", cause)
	}
	return n, udpAddrOf(host, port), nil
}

func (p *packet) WriteTo(b []byte, addr Addr) (int, error) {
	host, port, err := hostPort(addr.String())
	if err != nil {
		return 0, err
	}
	n, cause := dart.PacketWrite(p.handle, b, host, port)
	if cause != "" {
		return n, netError("write udp", cause)
	}
	return n, nil
}

func (p *packet) Close() error {
	if cause := dart.PacketClose(p.handle); cause != "" {
		return netError("close udp", cause)
	}
	return nil
}

func (p *packet) LocalAddr() Addr { return p.local }

func (p *packet) SetDeadline(t time.Time) error {
	p.read = deadlineNanos(t)
	p.write = p.read
	return p.apply()
}

func (p *packet) SetReadDeadline(t time.Time) error {
	p.read = deadlineNanos(t)
	return p.apply()
}

func (p *packet) SetWriteDeadline(t time.Time) error {
	p.write = deadlineNanos(t)
	return p.apply()
}

func (p *packet) apply() error {
	if cause := dart.PacketDeadline(p.handle, p.read, p.write); cause != "" {
		return netError("set deadline udp", cause)
	}
	return nil
}

// udpConn holds a datagram socket that a dial opened. Such a socket drops a
// datagram of another source, thus a plain read gives the datagrams of the
// peer only.
type udpConn struct {
	*packet
	host string
	port int
}

// LocalAddr gives the address of the socket.
func (c *udpConn) LocalAddr() Addr { return c.packet.LocalAddr() }

// RemoteAddr gives the address that Dial named.
func (c *udpConn) RemoteAddr() Addr { return udpAddrOf(c.host, c.port) }

func (c *udpConn) Read(b []byte) (int, error) {
	n, _, err := c.ReadFrom(b)
	return n, err
}

func (c *udpConn) Write(b []byte) (int, error) {
	n, cause := dart.PacketWrite(c.handle, b, c.host, c.port)
	if cause != "" {
		return n, netError("write udp", cause)
	}
	return n, nil
}

type listener struct {
	handle  int
	addr    Addr
	network string
}

func (l *listener) Accept() (Conn, error) {
	handle, cause := dart.Accept(l.handle)
	if cause != "" {
		return nil, netError("accept "+l.network, cause)
	}
	if l.network == "tcp" {
		return newTCPConn(handle), nil
	}
	return &conn{handle: handle, network: l.network}, nil
}

func (l *listener) Close() error {
	if cause := dart.ListenClose(l.handle); cause != "" {
		return netError("close "+l.network, cause)
	}
	return nil
}

func (l *listener) Addr() Addr { return l.addr }

// Dial opens a connection to an address. The network is tcp, udp, or unix.
func Dial(network, address string) (Conn, error) {
	switch network {
	case "tcp":
		host, port, err := hostPort(address)
		if err != nil {
			return nil, err
		}
		if host == "" {
			host = "127.0.0.1"
		}
		handle, cause := dart.DialTCP(host, port)
		if cause != "" {
			return nil, errors.New("dial tcp " + address + ": " + cause)
		}
		return newTCPConn(handle), nil
	case "unix":
		handle, cause := dart.DialUnix(address)
		if cause != "" {
			return nil, errors.New("dial unix " + address + ": " + cause)
		}
		return &conn{handle: handle, network: "unix"}, nil
	case "udp":
		host, port, err := hostPort(address)
		if err != nil {
			return nil, err
		}
		if host == "" {
			host = "127.0.0.1"
		}
		handle, local, cause := dart.DialUDP(host, port)
		if cause != "" {
			return nil, errors.New("dial udp " + address + ": " + cause)
		}
		return &udpConn{
			packet: &packet{handle: handle, local: &UDPAddr{Port: local}},
			host:   host,
			port:   port,
		}, nil
	}
	return nil, errors.New("dial " + network + ": the subset holds tcp, udp, and unix")
}

// Listen opens a listener on an address. A port of 0 asks the system for a
// free port, and Addr then gives the port that the system took.
func Listen(network, address string) (Listener, error) {
	switch network {
	case "tcp":
		host, port, err := hostPort(address)
		if err != nil {
			return nil, err
		}
		if host == "" {
			host = "0.0.0.0"
		}
		handle, taken, cause := dart.ListenTCP(host, port)
		if cause != "" {
			return nil, errors.New("listen tcp " + address + ": " + cause)
		}
		return &listener{
			handle:  handle,
			addr:    tcpAddrOf(host, taken),
			network: "tcp",
		}, nil
	case "unix":
		handle, cause := dart.ListenUnix(address)
		if cause != "" {
			return nil, errors.New("listen unix " + address + ": " + cause)
		}
		return &listener{
			handle:  handle,
			addr:    &UnixAddr{Name: address, Net: "unix"},
			network: "unix",
		}, nil
	}
	return nil, errors.New("listen " + network + ": the subset holds tcp and unix")
}

// ListenPacket opens a datagram socket on an address. A port of 0 asks the
// system for a free port, and LocalAddr then gives the port that the system
// took.
func ListenPacket(network, address string) (PacketConn, error) {
	if network != "udp" {
		return nil, errors.New("listen " + network + ": the subset holds udp only")
	}
	host, port, err := hostPort(address)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = "0.0.0.0"
	}
	handle, taken, cause := dart.ListenUDP(host, port)
	if cause != "" {
		return nil, errors.New("listen udp " + address + ": " + cause)
	}
	return &packet{handle: handle, local: udpAddrOf(host, taken)}, nil
}

// A Dialer opens a connection with a timeout or with a context.
type Dialer struct {
	Timeout time.Duration
	// Resolver reads the host name of the address. A nil field gives the name
	// to the runtime, which reads the resolver of the platform.
	Resolver *Resolver
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
// The dial runs on a goroutine, thus the select takes the first of the 2. A
// dial that ends after the context closes its own connection.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (Conn, error) {
	if d.Timeout > 0 {
		inner, cancel := context.WithTimeout(ctx, d.Timeout)
		defer cancel()
		ctx = inner
	}
	if d.Resolver != nil {
		resolved, err := d.resolve(ctx, network, address)
		if err != nil {
			return nil, err
		}
		address = resolved
	}
	done := make(chan dialResult, 1)
	go func() {
		c, cause := Dial(network, address)
		done <- dialResult{conn: c, cause: cause}
	}()
	select {
	case r := <-done:
		return r.conn, r.cause
	case <-ctx.Done():
		go func() {
			if r := <-done; r.conn != nil {
				r.conn.Close()
			}
		}()
		return nil, ctx.Err()
	}
}

// resolve gives the address again with the host name replaced by the first
// address that the resolver of the dialer gives.
func (d *Dialer) resolve(ctx context.Context, network, address string) (string, error) {
	if network == "unix" {
		return address, nil
	}
	host, port, err := SplitHostPort(address)
	if err != nil {
		return "", err
	}
	if host == "" || ParseIP(host) != nil {
		return address, nil
	}
	ips, err := d.Resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", &DNSError{Err: errNoSuchHost, Name: host, IsNotFound: true}
	}
	return JoinHostPort(ips[0].String(), port), nil
}

// SplitHostPort reads the host and the port of an address. Go accepts an IPv6
// host in brackets, and this form stays outside the subset.
func SplitHostPort(address string) (string, string, error) {
	i := strings.LastIndex(address, ":")
	if i < 0 {
		return "", "", errors.New("address " + address + ": missing port")
	}
	return address[:i], address[i+1:], nil
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
