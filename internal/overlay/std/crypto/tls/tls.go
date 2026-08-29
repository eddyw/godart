//go:build dart

// Package tls gives a TLS client over the secure socket of Dart.
//
// Dart holds the whole handshake. It honors InsecureSkipVerify, ServerName, and
// NextProtos, and it honors no version and no cipher suite. A value of
// MinVersion, MaxVersion, or CipherSuites would reach no operation, thus a dial
// with such a value stops the program.
//
// The package holds a client only. A server needs a certificate chain, and the
// surface holds no crypto/x509.
package tls

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"syscall/dart"
	"time"
)

// The versions of the protocol. A Config holds one only to name a value that
// Dart cannot honor.
const (
	VersionTLS10 = 0x0301
	VersionTLS11 = 0x0302
	VersionTLS12 = 0x0303
	VersionTLS13 = 0x0304
)

// A Config holds the settings of a TLS connection.
type Config struct {
	// InsecureSkipVerify drops the check of the certificate of the server.
	InsecureSkipVerify bool
	// ServerName names the host of the check and of the extension of the name
	// of the server. An empty name reads the host of the address.
	ServerName string
	// NextProtos holds each protocol of ALPN that the client offers.
	NextProtos []string
	// MinVersion, MaxVersion, and CipherSuites each reach no operation. A dial
	// with a value stops the program.
	MinVersion   uint16
	MaxVersion   uint16
	CipherSuites []uint16
}

// Clone gives a copy of the config. A nil config gives nil.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

// A ConnectionState names the result of the handshake.
type ConnectionState struct {
	// NegotiatedProtocol names the protocol that ALPN took.
	NegotiatedProtocol string
	// ServerName holds the name that the client sent.
	ServerName string
}

// A Conn is a TLS connection. It holds each method of net.Conn.
type Conn struct {
	handle int
	state  ConnectionState
	read   int64
	write  int64
}

func (c *Conn) Read(b []byte) (int, error) {
	n, cause := dart.ConnRead(c.handle, b)
	if cause != "" {
		return n, errors.New("read tls: " + cause)
	}
	if n == 0 && len(b) > 0 {
		return 0, errEOF
	}
	return n, nil
}

func (c *Conn) Write(b []byte) (int, error) {
	n, cause := dart.ConnWrite(c.handle, b)
	if cause != "" {
		return n, errors.New("write tls: " + cause)
	}
	return n, nil
}

func (c *Conn) Close() error {
	if cause := dart.ConnClose(c.handle); cause != "" {
		return errors.New("close tls: " + cause)
	}
	return nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	c.read = deadlineNanos(t)
	c.write = c.read
	return c.apply()
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	c.read = deadlineNanos(t)
	return c.apply()
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.write = deadlineNanos(t)
	return c.apply()
}

func (c *Conn) apply() error {
	if cause := dart.ConnDeadline(c.handle, c.read, c.write); cause != "" {
		return errors.New("set deadline tls: " + cause)
	}
	return nil
}

// Handshake does nothing. Dart runs the handshake inside the dial.
func (c *Conn) Handshake() error { return nil }

// ConnectionState gives the result of the handshake.
func (c *Conn) ConnectionState() ConnectionState { return c.state }

// errEOF names the end of the stream. The package io imports no tls, thus the
// value compares equal to io.EOF through the text of the error.
var errEOF = errors.New("EOF")

func deadlineNanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// Dial opens a TLS connection. The network must be tcp.
func Dial(network, addr string, config *Config) (*Conn, error) {
	if network != "tcp" {
		return nil, errors.New("tls: the network " + network + " is outside the subset")
	}
	reject(config)
	host, port, err := hostPort(addr)
	if err != nil {
		return nil, err
	}
	name := ""
	insecure := false
	var protos []string
	if config != nil {
		name = config.ServerName
		insecure = config.InsecureSkipVerify
		protos = config.NextProtos
	}
	handle, chosen, cause := dart.DialTLS(host, port, insecure, name, protos)
	if cause != "" {
		return nil, errors.New("tls: dial " + addr + ": " + cause)
	}
	if name == "" {
		name = host
	}
	return &Conn{
		handle: handle,
		state:  ConnectionState{NegotiatedProtocol: chosen, ServerName: name},
	}, nil
}

// DialWithDialer opens a TLS connection with the timeout of the dialer. Each
// other field of the dialer reaches no operation.
func DialWithDialer(dialer *net.Dialer, network, addr string, config *Config) (*Conn, error) {
	if dialer != nil && dialer.Timeout != 0 {
		return nil, errors.New("tls: a timeout of the dialer is outside the subset")
	}
	return Dial(network, addr, config)
}

// reject stops the program for a field that Dart cannot honor. A silent change
// of the meaning of such a field is outside the rule of the project.
func reject(config *Config) {
	if config == nil {
		return
	}
	if config.MinVersion != 0 {
		panic("crypto/tls: Dart holds the whole handshake, thus MinVersion reaches no operation")
	}
	if config.MaxVersion != 0 {
		panic("crypto/tls: Dart holds the whole handshake, thus MaxVersion reaches no operation")
	}
	if len(config.CipherSuites) != 0 {
		panic("crypto/tls: Dart holds the whole handshake, thus CipherSuites reaches no operation")
	}
}

func hostPort(addr string) (string, int, error) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "", 0, errors.New("tls: the address " + addr + " holds no port")
	}
	port, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		return "", 0, errors.New("tls: the address " + addr + " holds no port")
	}
	host := addr[:i]
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port, nil
}
