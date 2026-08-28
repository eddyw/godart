//go:build dart

// Package net gives a TCP client and a TCP server.
//
// The surface holds TCP only, and an address holds a numeric host or a host
// name that Dart resolves. There is no UDP, no Unix socket, no TLS, no
// deadline, and no IPv6 form with brackets. `docs/parity.md` lists each gap.
package net

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"syscall/dart"
)

// Conn is a stream between 2 programs.
type Conn interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
}

// Listener waits for a connection of a client.
type Listener interface {
	Accept() (Conn, error)
	Close() error
	Addr() Addr
}

// Addr is the address of a listener.
type Addr interface {
	Network() string
	String() string
}

// tcpAddr holds a host and a port. Go exports TCPAddr with an IP and a zone,
// and that type stays outside the subset. Thus a type assertion of an Addr
// gives a compile error, and not a wrong value.
type tcpAddr struct {
	host string
	port int
}

func (a tcpAddr) Network() string { return "tcp" }

func (a tcpAddr) String() string { return a.host + ":" + strconv.Itoa(a.port) }

type conn struct {
	handle int
}

func (c *conn) Read(b []byte) (int, error) {
	n, cause := dart.ConnRead(c.handle, b)
	if cause != "" {
		return n, errors.New("read tcp: " + cause)
	}
	if n == 0 && len(b) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (c *conn) Write(b []byte) (int, error) {
	n, cause := dart.ConnWrite(c.handle, b)
	if cause != "" {
		return n, errors.New("write tcp: " + cause)
	}
	return n, nil
}

func (c *conn) Close() error {
	if cause := dart.ConnClose(c.handle); cause != "" {
		return errors.New("close tcp: " + cause)
	}
	return nil
}

type listener struct {
	handle int
	addr   tcpAddr
}

func (l *listener) Accept() (Conn, error) {
	handle, cause := dart.Accept(l.handle)
	if cause != "" {
		return nil, errors.New("accept tcp: " + cause)
	}
	return &conn{handle: handle}, nil
}

func (l *listener) Close() error {
	if cause := dart.ListenClose(l.handle); cause != "" {
		return errors.New("close tcp: " + cause)
	}
	return nil
}

func (l *listener) Addr() Addr { return l.addr }

// Dial opens a connection to an address. The network must be tcp.
func Dial(network, address string) (Conn, error) {
	if network != "tcp" {
		return nil, errors.New("dial " + network + ": the subset holds tcp only")
	}
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
	return &conn{handle: handle}, nil
}

// Listen opens a listener on an address. A port of 0 asks the system for a
// free port, and Addr then gives the port that the system took.
func Listen(network, address string) (Listener, error) {
	if network != "tcp" {
		return nil, errors.New("listen " + network + ": the subset holds tcp only")
	}
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
	return &listener{handle: handle, addr: tcpAddr{host: host, port: taken}}, nil
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
