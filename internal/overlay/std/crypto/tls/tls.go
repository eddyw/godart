//go:build dart

// Package tls gives a TLS client and a TLS server over the secure socket of
// Dart.
//
// Dart holds the whole handshake. It honors InsecureSkipVerify, ServerName,
// NextProtos, and the first certificate of a server config, and it honors no
// version and no cipher suite. A value of MinVersion, MaxVersion, or
// CipherSuites would reach no operation, thus a dial or a listen with such a
// value stops the program.
//
// A Certificate holds the DER chain, parsed key, and leaf certificate.
// The listener encodes the chain and key as PEM for Dart.
package tls

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
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

	// Deprecated: SSLv3 carries a break of the cryptography, and no package
	// of the surface takes it.
	VersionSSL30 = 0x0300
)

// A Config holds the settings of a TLS connection.
type Config struct {
	// InsecureSkipVerify drops the check of the certificate of the server.
	InsecureSkipVerify bool
	// ServerName names the host of the check and of the extension of the name
	// of the server. An empty name reads the host of the address.
	ServerName string
	// NextProtos holds each protocol of ALPN that the client offers, or that
	// the server accepts.
	NextProtos []string
	// Certificates holds the certificate chain of a server. Listen reads the
	// first one, and a client dial with one stops the program, because the
	// secure socket of Dart takes no client certificate on a dial.
	Certificates []Certificate
	// MinVersion, MaxVersion, and CipherSuites each reach no operation. A dial
	// with a value stops the program.
	MinVersion   uint16
	MaxVersion   uint16
	CipherSuites []uint16
	// PreferServerCipherSuites reaches no operation here, and Go itself
	// ignores it too, thus the field carries no lie.
	//
	// Deprecated: PreferServerCipherSuites is ignored.
	PreferServerCipherSuites bool
}

// A Certificate holds a DER chain and its private key.
type Certificate struct {
	// Certificate holds the leaf first, followed by its intermediate certificates.
	Certificate [][]byte
	PrivateKey  crypto.PrivateKey
	Leaf        *x509.Certificate
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
	// HandshakeComplete reports a handshake that ran to its end. Dart gives a
	// connection only after the handshake, thus each Conn of this package
	// holds the value true. A zero ConnectionState holds false, exactly as in
	// Go.
	HandshakeComplete bool
	// NegotiatedProtocolIsMutual always holds true. Go 1.14 and later set the
	// same value, and the handshake of Go fails when the 2 sides agree on no
	// protocol.
	//
	// Deprecated: this value is always true.
	NegotiatedProtocolIsMutual bool
}

// A Conn is a TLS connection. It holds each method of net.Conn.
type Conn struct {
	handle int
	state  ConnectionState
	read   int64
	write  int64
}

// LocalAddr gives the address of this side of the connection.
func (c *Conn) LocalAddr() net.Addr { return connAddr(c.handle, true) }

// RemoteAddr gives the address of the peer.
func (c *Conn) RemoteAddr() net.Addr { return connAddr(c.handle, false) }

// connAddr reads the 2 addresses of the socket of a TLS connection.
func connAddr(handle int, local bool) net.Addr {
	lHost, lPort, rHost, rPort, cause := dart.ConnAddrs(handle)
	if cause != "" {
		lHost, lPort, rHost, rPort = "", 0, "", 0
	}
	if local {
		return &net.TCPAddr{IP: net.ParseIP(lHost), Port: lPort}
	}
	return &net.TCPAddr{IP: net.ParseIP(rHost), Port: rPort}
}

func (c *Conn) Read(b []byte) (int, error) {
	n, cause := dart.ConnRead(c.handle, b)
	if cause != "" {
		return n, errors.New("read tls: " + cause)
	}
	if n == 0 && len(b) > 0 {
		return 0, io.EOF
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

// HandshakeContext does nothing. Dart runs the handshake inside the dial,
// thus each Conn of this package already holds a complete handshake.
//
// Go also gives nil for a complete handshake, and it reads the context no
// more at that point.
func (c *Conn) HandshakeContext(ctx context.Context) error { return nil }

// ConnectionState gives the result of the handshake.
func (c *Conn) ConnectionState() ConnectionState { return c.state }

// Handle gives the handle of the connection inside the runtime. The HTTP/2
// transport of net/http wraps the socket behind it. The real tls.Conn of Go
// holds no such method.
func (c *Conn) Handle() int { return c.handle }

func deadlineNanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// Listen opens a TLS listener. The network must be tcp, and the config must
// hold a certificate.
func Listen(network, laddr string, config *Config) (net.Listener, error) {
	if network != "tcp" {
		return nil, errors.New("tls: the network " + network + " is outside the subset")
	}
	reject(config)
	if config == nil || len(config.Certificates) == 0 {
		return nil, errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
	}
	host, port, err := hostPort(laddr)
	if err != nil {
		return nil, err
	}
	cert := config.Certificates[0]
	var chain []byte
	for _, der := range cert.Certificate {
		chain = append(chain, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, err
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	handle, taken, cause := dart.ListenTLS(host, port, chain, key,
		config.NextProtos)
	if cause != "" {
		return nil, errors.New("tls: listen " + laddr + ": " + cause)
	}
	return &listener{handle: handle, addr: listenAddr{host: host, port: taken}}, nil
}

// listener waits for a TLS connection of a client.
type listener struct {
	handle int
	addr   listenAddr
}

// Accept waits for 1 connection and runs the handshake.
func (l *listener) Accept() (net.Conn, error) {
	handle, proto, cause := dart.AcceptTLS(l.handle)
	if cause != "" {
		return nil, errors.New("tls: accept: " + cause)
	}
	return &Conn{handle: handle,
		state: ConnectionState{NegotiatedProtocol: proto,
			HandshakeComplete: true, NegotiatedProtocolIsMutual: true}}, nil
}

// Close ends the listener. Each connection of it stays open, exactly as in
// Go.
func (l *listener) Close() error {
	if cause := dart.ListenClose(l.handle); cause != "" {
		return errors.New("tls: close: " + cause)
	}
	return nil
}

// Addr gives the address of the listener.
func (l *listener) Addr() net.Addr { return l.addr }

// listenAddr holds the host and the port of a TLS listener. The package net
// keeps its own address type unexported, thus this package holds one too.
type listenAddr struct {
	host string
	port int
}

func (a listenAddr) Network() string { return "tcp" }

func (a listenAddr) String() string {
	return a.host + ":" + strconv.Itoa(a.port)
}

// Dial opens a TLS connection. The network must be tcp.
func Dial(network, addr string, config *Config) (*Conn, error) {
	if network != "tcp" {
		return nil, errors.New("tls: the network " + network + " is outside the subset")
	}
	reject(config)
	if config != nil && len(config.Certificates) != 0 {
		panic("crypto/tls: the secure socket of Dart takes no client certificate on a dial")
	}
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
		state: ConnectionState{NegotiatedProtocol: chosen, ServerName: name,
			HandshakeComplete: true, NegotiatedProtocolIsMutual: true},
	}, nil
}

// DialProxy opens a TLS connection through an HTTP proxy. proxyAddr names the
// proxy, addr names the target, and head holds the whole CONNECT request.
//
// Go holds no such function. Go writes the CONNECT request itself and then
// wraps the connection with tls.Client. The secure socket of Dart takes over a
// socket that no read touched, thus the runtime holds the whole exchange.
// docs/parity.md holds that gap.
func DialProxy(proxyAddr, addr, head string, config *Config) (*Conn, error) {
	reject(config)
	host, port, err := hostPort(proxyAddr)
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
	if name == "" {
		name, _, err = hostPort(addr)
		if err != nil {
			return nil, err
		}
	}
	handle, chosen, cause := dart.DialTLSProxy(host, port, head, insecure, name, protos)
	if cause != "" {
		return nil, errors.New("tls: dial " + addr + " through " + proxyAddr + ": " + cause)
	}
	return &Conn{
		handle: handle,
		state: ConnectionState{NegotiatedProtocol: chosen, ServerName: name,
			HandshakeComplete: true, NegotiatedProtocolIsMutual: true},
	}, nil
}

// DialWithDialer opens a TLS connection with the timeout, the deadline, and
// the cancel channel of the dialer.
//
// The dial and the handshake share the 1 limit, because the secure socket of
// Dart runs the 2 in 1 call. Go gives the handshake the limit alone.
func DialWithDialer(dialer *net.Dialer, network, addr string, config *Config) (*Conn, error) {
	return dial(context.Background(), dialer, network, addr, config)
}

// A Dialer opens a TLS connection with a context.
type Dialer struct {
	// NetDialer holds the dialer of the plain connection. A nil field holds
	// the zero net.Dialer.
	NetDialer *net.Dialer

	// Config holds the settings of the TLS connection. A nil field holds the
	// zero Config.
	Config *Config
}

// Dial opens a TLS connection with the settings of the dialer.
//
// The result holds a *Conn, exactly as in Go.
func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

// DialContext opens a TLS connection and ends the wait when the context ends.
//
// The dial runs on a goroutine, thus the select takes the first of the 2
// events. A dial that ends after the wait closes its own connection.
//
// The dial and the handshake share the 1 limit, because the secure socket of
// Dart runs the 2 in 1 call.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	c, err := dial(ctx, d.NetDialer, network, addr, d.Config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// dialResult holds the 2 results of a dial that runs on a goroutine.
type dialResult struct {
	conn  *Conn
	cause error
}

// dial opens a TLS connection under the limits of a net.Dialer and of a
// context.
func dial(ctx context.Context, dialer *net.Dialer, network, addr string, config *Config) (*Conn, error) {
	if dialer != nil {
		if dialer.Timeout > 0 {
			inner, cancel := context.WithTimeout(ctx, dialer.Timeout)
			defer cancel()
			ctx = inner
		}
		if !dialer.Deadline.IsZero() {
			inner, cancel := context.WithDeadline(ctx, dialer.Deadline)
			defer cancel()
			ctx = inner
		}
	}
	target, name, err := resolveTarget(ctx, dialer, addr, config)
	if err != nil {
		return nil, err
	}
	if name != "" {
		inner := config.Clone()
		if inner == nil {
			inner = &Config{}
		}
		inner.ServerName = name
		config = inner
	}
	var cancelChan <-chan struct{}
	if dialer != nil {
		cancelChan = dialer.Cancel
	}
	done := make(chan dialResult, 1)
	go func() {
		c, cause := Dial(network, target, config)
		done <- dialResult{conn: c, cause: cause}
	}()
	select {
	case r := <-done:
		return r.conn, r.cause
	case <-ctx.Done():
		go closeLate(done)
		return nil, ctx.Err()
	case <-cancelChan:
		go closeLate(done)
		return nil, errors.New("tls: the dial ended, because the cancel channel closed")
	}
}

// closeLate closes a connection of a dial that ended after the caller left.
func closeLate(done chan dialResult) {
	if r := <-done; r.conn != nil {
		r.conn.Close()
	}
}

// resolveTarget reads the host name of the address with the resolver of the
// dialer. It gives the address of the dial and the name of the handshake.
//
// A nil resolver leaves the name to the runtime, thus the address passes
// unchanged and the name of the handshake stays empty.
func resolveTarget(ctx context.Context, dialer *net.Dialer, addr string, config *Config) (string, string, error) {
	if dialer == nil || dialer.Resolver == nil {
		return addr, "", nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", "", err
	}
	if host == "" || net.ParseIP(host) != nil {
		return addr, "", nil
	}
	ips, err := dialer.Resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return "", "", err
	}
	if len(ips) == 0 {
		return "", "", errors.New("tls: the resolver gave no address of " + host)
	}
	// The dial takes the address, thus the handshake keeps the name of the
	// user. A ServerName of the config wins, exactly as in Go.
	name := host
	if config != nil && config.ServerName != "" {
		name = config.ServerName
	}
	return net.JoinHostPort(ips[0].String(), port), name, nil
}

// LoadX509KeyPair reads a PEM chain and a PEM key of the file system, and it
// builds a Certificate of the 2.
func LoadX509KeyPair(certFile, keyFile string) (Certificate, error) {
	certPEMBlock, err := os.ReadFile(certFile)
	if err != nil {
		return Certificate{}, err
	}
	keyPEMBlock, err := os.ReadFile(keyFile)
	if err != nil {
		return Certificate{}, err
	}
	return X509KeyPair(certPEMBlock, keyPEMBlock)
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
