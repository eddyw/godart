//go:build dart

package httptest

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// A Server holds a listener on a port of the machine and answers each request
// with a handler.
//
// The real Server of Go waits for each request that stands open, and it gives
// the certificate of its TLS form as a value of x509. The surface holds no
// package crypto/x509, thus the overlay holds no method Certificate.
type Server struct {
	// URL holds the base address of the server, for example
	// `http://127.0.0.1:52341`. The first call of Client, Start, or StartTLS
	// writes it.
	URL string
	// Listener holds the listener of the server. A server of NewTestServer
	// that neither Start nor StartTLS ran holds none.
	Listener net.Listener
	// Config holds the server that answers each request.
	Config *http.Server

	// EnableHTTP2 asks StartTLS for h2 in the list of ALPN, and the client
	// of the server then also offers h2. Set it before StartTLS.
	EnableHTTP2 bool

	// TLS holds the settings of the TLS form. A start writes a fresh config,
	// and it keeps each field of a config that the caller set before.
	TLS *tls.Config

	t      testing.TB
	client *http.Client

	// once guards the start of a server of a test that Client reaches before
	// Start.
	once sync.Once

	mu      sync.Mutex
	started bool
	// conns holds each connection that a listener accepted, because
	// CloseClientConnections ends them. The server of the overlay reports no
	// state of a connection. A number names each entry, because a map of the
	// runtime reads the fields of a key, and a connection changes them.
	conns map[int]net.Conn
	seq   int
	// plain and secure hold the 2 listeners of a server of NewTestServer.
	plain  net.Listener
	secure net.Listener
}

// NewServer builds a server and starts it.
func NewServer(handler http.Handler) *Server {
	s := NewUnstartedServer(handler)
	s.Start()
	return s
}

// NewTLSServer builds a server of TLS and starts it.
func NewTLSServer(handler http.Handler) *Server {
	s := NewUnstartedServer(handler)
	s.StartTLS()
	return s
}

// NewUnstartedServer builds a server that holds a listener and answers no
// request. Start answers each request.
func NewUnstartedServer(handler http.Handler) *Server {
	return &Server{
		Listener: newLocalListener(),
		Config:   &http.Server{Handler: handler},
	}
}

// NewTestServer builds a server for a test, and the test closes it at its end.
// A nil handler answers the status 500.
//
// Go answers on a network in memory, and the client of the server then reaches
// it for each host. The surface holds no pipe of memory, thus the overlay
// binds 2 listeners of the loopback address, 1 plain and 1 of TLS, and the
// client of the server sends each request to them. URL then reads
// `http://example.com` and Listener stays nil, exactly as in Go.
//
// Go also sends a request of the host example.com to a server of a test that
// Start bound. The overlay reaches such a server through URL only.
//
// Go reports a panic of the handler to the test. The server of the overlay
// recovers no panic, thus a panic of the handler ends the program.
func NewTestServer(t testing.TB, handler http.Handler) *Server {
	s := &Server{
		t:      t,
		Config: &http.Server{Handler: testHandler{h: handler}},
	}
	t.Cleanup(s.Close)
	return s
}

// A testHandler answers the status 500 for a server of NewTestServer that
// holds no handler of its own.
type testHandler struct {
	h http.Handler
}

func (t testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	t.h.ServeHTTP(w, r)
}

// startCommon marks the start and binds the listener of a server of a test.
func (s *Server) startCommon(loopback bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		panic("httptest: server already started")
	}
	s.started = true
	if s.t == nil || !loopback {
		return
	}
	// Client must bind no listener of its own after a start of the loopback
	// form.
	s.once.Do(func() {})
	if s.Listener != nil {
		panic("httptest: Server.Listener is unexpectedly set")
	}
	s.Listener = newLocalListener()
}

// Start answers each request of the listener.
func (s *Server) Start() {
	s.startCommon(true)
	s.URL = "http://" + s.Listener.Addr().String()
	s.client = &http.Client{}
	go s.Config.Serve(s.track(s.Listener))
}

// StartTLS answers each request of the listener over TLS, with the test
// certificate of the package.
//
// Go wraps the listener that the server already holds. The secure socket of
// Dart binds a socket of its own, thus the plain listener closes, and the TLS
// listener takes a fresh port of the loopback address.
func (s *Server) StartTLS() {
	s.startCommon(true)
	s.initTLS()
	s.Listener.Close()
	l, err := tls.Listen("tcp", "127.0.0.1:0", s.TLS)
	if err != nil {
		panic("httptest: failed to listen on a port: " + err.Error())
	}
	s.Listener = l
	s.URL = "https://" + l.Addr().String()
	s.client = &http.Client{Transport: s.transport()}
	go s.Config.Serve(s.track(l))
}

// initTLS writes the config of the TLS form. A config that the caller set
// keeps each field that it holds.
func (s *Server) initTLS() {
	cert, err := tls.X509KeyPair([]byte(testCert), []byte(testKey()))
	if err != nil {
		panic("httptest: " + err.Error())
	}
	s.TLS = s.TLS.Clone()
	if s.TLS == nil {
		s.TLS = &tls.Config{}
	}
	if s.TLS.NextProtos == nil {
		protos := []string{"http/1.1"}
		if s.EnableHTTP2 {
			protos = []string{"h2", "http/1.1"}
		}
		s.TLS.NextProtos = protos
	}
	if len(s.TLS.Certificates) == 0 {
		s.TLS.Certificates = []tls.Certificate{cert}
	}
}

// transport gives the transport of the client of a TLS server.
//
// The certificate is self signed, and the surface holds no root pool, thus the
// client of the server drops the check. Go gives its client the certificate as
// a root.
func (s *Server) transport() *http.Transport {
	return &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		ForceAttemptHTTP2: s.EnableHTTP2,
	}
}

// startFake binds the 2 listeners that stand for the network in memory of Go.
func (s *Server) startFake() {
	s.startCommon(false)
	s.initTLS()
	plain := newLocalListener()
	secure, err := tls.Listen("tcp", "127.0.0.1:0", s.TLS)
	if err != nil {
		panic("httptest: failed to listen on a port: " + err.Error())
	}
	s.mu.Lock()
	s.plain, s.secure = plain, secure
	s.mu.Unlock()
	s.URL = "http://example.com"
	s.client = &http.Client{Transport: &fakeTransport{
		plain:  plain.Addr().String(),
		secure: secure.Addr().String(),
		inner:  s.transport(),
	}}
	go s.Config.Serve(s.track(plain))
	go s.Config.Serve(s.track(secure))
}

// A fakeTransport sends each request to a listener of a server of
// NewTestServer, whatever host the URL names. The field Host of the request
// keeps the host of the caller, thus the handler reads the name that Go gives
// it over the network in memory.
type fakeTransport struct {
	plain  string
	secure string
	inner  *http.Transport
}

func (t *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := *req
	target := *req.URL
	if out.Host == "" {
		out.Host = target.Host
	}
	if target.Scheme == "https" {
		target.Host = t.secure
	} else {
		target.Host = t.plain
	}
	out.URL = &target
	res, err := t.inner.RoundTrip(&out)
	if res != nil {
		res.Request = req
	}
	return res, err
}

func (t *fakeTransport) CloseIdleConnections() { t.inner.CloseIdleConnections() }

// testCert holds a self signed certificate of the loopback address and of
// localhost, valid to the year 2126. The pair serves tests only.
const testCert = `-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIJAK9noFnHmJSfMA0GCSqGSIb3DQEBCwUAMCUxDzANBgNV
BAoMBmdvZGFydDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI2MDgzMTIyMDExNloY
DzIxMjYwODA3MjIwMTE2WjAlMQ8wDQYDVQQKDAZnb2RhcnQxEjAQBgNVBAMMCWxv
Y2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM6gZHPdERlf
6HZ4pCMFHGcLZ6RgAMKgbt6yciWue2qvRBqOaUJLQA0OFp4DCR5nQX0WjwutL2Y9
vytkOibGZpUwhWJB/m/OlErH8pk1TRyqc/QcXkAEqiRoc2iYQ3KQbig/m7ZTMEKb
TZgzvBddVcCA3mLI3HpLA8x8Ipsnxtb5kT6uct0PlwPu8T5jTlyVvMupQjtbuiMC
b26BX7Vlq/nRsfzfp07+Z9rrwc2pCdVdyvl7CqA8aiE5ElX/QpaWqKEG8IM0W41W
ODwBymimb+0KkQOOv+1+N8dsIYQZenKFJakIsHUEXUraCdojvLvKg56+B1hCBKyn
LPzB7/3PrusCAwEAAaMwMC4wLAYDVR0RBCUwI4IJbG9jYWxob3N0hwR/AAABhxAA
AAAAAAAAAAAAAAAAAAABMA0GCSqGSIb3DQEBCwUAA4IBAQATCypJ9ejqEvqUFIAc
fFUcW4ryKAmrHtNpectlRu9pf3EW/fbU9Z2fKzjA8oALL8qu0XX2uwGZ6RWYtt85
nSjOq1cDODcefcvpxTe+qB9y4Pg0xEnAtkjDBufa6TvBHTLvE463tx1+S5SFVGcS
tBILsYVDSdsD1G+9gp5Wuf/TnZN0odhGolhH4q5ufhO6kEtGo+6VNBM4+CFf8497
48VnoleUYEhW/N+/vRcuSKlEidNFjTO1s43Dw57apQDSdpaovljuYC/AMzcnjgrQ
6sUwRDUAUHgngWQREik06DARuX4/dgoBFqKlHFk/PH4K/ioQMWi3mBsTyA0PSNAM
BvWv
-----END CERTIFICATE-----`

// testKey gives the private key of testCert. The text holds the word TESTING,
// thus a scanner of secrets does not read a live key, exactly as the test
// certificate of Go does.
func testKey() string {
	return strings.ReplaceAll(testKeyMarked, "TESTING KEY", "PRIVATE KEY")
}

const testKeyMarked = `-----BEGIN TESTING KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDOoGRz3REZX+h2
eKQjBRxnC2ekYADCoG7esnIlrntqr0QajmlCS0ANDhaeAwkeZ0F9Fo8LrS9mPb8r
ZDomxmaVMIViQf5vzpRKx/KZNU0cqnP0HF5ABKokaHNomENykG4oP5u2UzBCm02Y
M7wXXVXAgN5iyNx6SwPMfCKbJ8bW+ZE+rnLdD5cD7vE+Y05clbzLqUI7W7ojAm9u
gV+1Zav50bH836dO/mfa68HNqQnVXcr5ewqgPGohORJV/0KWlqihBvCDNFuNVjg8
Acpopm/tCpEDjr/tfjfHbCGEGXpyhSWpCLB1BF1K2gnaI7y7yoOevgdYQgSspyz8
we/9z67rAgMBAAECggEAUso7VZ1HI6alCwsNoSaEGdmc/mSlCVak9cAEMor10mpF
XGqhTDnAgTqa31cSCUeQGuG2ERSkQFAPUACJuE2YfsCBQTx2CTB5hHkXAEI2En1N
lorjTL1RuuLLEyanJ7ktGh1ac7RjStzVO13HSIlSOZ6o/GXEyY9TB1TZvNc0OZ/u
9OiJYlyr6j1ToUePBnzpX+x9mNp5x3nlqQrKZQnR9sOi1DMNgR4SrCHsI/X2SWcR
00asTxZrDZF+0YwpybUcqteJBeJ0GSXtF4CSwFLNLeGDcIBhn78N7L6dWPitYjnF
AmV+KuU4cMWKjRuApWc5nmDRG5AZmuZOH7WIsF4b0QKBgQDzPUYgPS48LxI2jgNC
6N17EK9y1VQbxGblD5w0kxBJM0F2SYv2T0B/bKg2vAboyJRJJDLkjLZ4gxBPYmzW
ZLKhKh1nsDqJsUvJsJIuODtbuvui3392o3CswANlobVcz1/VLxHjU6oJznno0yAS
AeeIDmc/L5ln8+8QvxuqTq1agwKBgQDZd2eyKoVUuD8WrMuh1Js3OPbF60xkVYFQ
c8Xyo93uTFV4T9bRoJrmtACugf1vqaLNHyA9fuvZYb+zvhbDN0/rCWi4xAuhpQlT
7PV4wJxk9az/fnbb3WSsZuBOE0DFpLa/i6xVJ6MecP+hQAuxWL0eAFTqQdueFiAm
3fxLOPfNeQKBgD4clG6hENBfeLgjlWOdkQje3t5k6SBaC6hMTygBsbPvebTZ3Pqd
jCu9tEqdwunoZi0zKCfCYsntNdn/mSk5938+ANo8XwZQFrk+9GUHCOlImv8hMSw7
WgdS1Ms7+tLjRWZrqZayA4aTif/RAax3fgb5HpY88mBgItjU5ByeEJg/AoGAOn85
IJ75gKGTLulKWXzUR9WF5BV4ViD+CQMrgyp7P2e/Mf7f/UGjiV1wNYD4CHpsNZsi
1Af5L19w7su6WcbeM44zYkgyDMhVKwM5wMKnDy1a3NtK47e5JfsCaV4adU/qbLoI
oFtqMH0a9mhA/PKu5IwYS29twSeYfHwIKr/nQJkCgYEAyMxUkCT2C/qDfqeCLv4/
irn4zBlfRSM6My2WEK+eTAM0Z6LvkwwgvPLxLuzZJGfBc9CXzEYl3/8SswXxFYyV
GCgH6kOX2blj7DqiYOqmYsU47ll2Wn19QGp01CZ4++uJhGQ5ONHMdrBv9aBDKU8Q
BChx2JXwiXwYE525nl9qsTw=
-----END TESTING KEY-----`

// Close ends each listener and each connection of the client that stands open
// with no request.
func (s *Server) Close() {
	s.mu.Lock()
	plain, secure := s.plain, s.secure
	s.mu.Unlock()
	// The server of the overlay keeps 1 listener only, thus the 2 listeners
	// of a server of NewTestServer end here.
	if plain != nil {
		plain.Close()
	}
	if secure != nil {
		secure.Close()
	}
	s.Config.Close()
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
}

// CloseClientConnections ends each connection that a client of the server
// holds.
func (s *Server) CloseClientConnections() {
	s.mu.Lock()
	open := make([]net.Conn, 0, len(s.conns))
	for _, c := range s.conns {
		open = append(open, c)
	}
	s.mu.Unlock()
	for _, c := range open {
		c.Close()
	}
}

// Client gives a client that reaches the server. A server of a test binds its
// listeners here, because such a server starts at the first use.
func (s *Server) Client() *http.Client {
	if s.t != nil {
		s.once.Do(s.startFake)
	}
	if s.client == nil {
		s.client = &http.Client{}
	}
	return s.client
}

// newLocalListener binds a port of the loopback address.
func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("httptest: failed to listen on a port: " + err.Error())
	}
	return l
}

// track counts each connection of the listener, and it gives the listener that
// the server reads.
func (s *Server) track(l net.Listener) net.Listener {
	return &trackListener{Listener: l, server: s}
}

func (s *Server) claim() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *Server) remember(id int, c net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conns == nil {
		s.conns = map[int]net.Conn{}
	}
	s.conns[id] = c
}

func (s *Server) forget(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, id)
}

// A trackListener holds each connection that it accepted in the server, and
// CloseClientConnections then reaches them.
type trackListener struct {
	net.Listener
	server *Server
}

func (l *trackListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	id := l.server.claim()
	var out net.Conn
	if secure, ok := c.(*tls.Conn); ok {
		out = &trackTLSConn{
			trackConn: trackConn{Conn: c, server: l.server, id: id},
			secure:    secure,
		}
	} else {
		out = &trackConn{Conn: c, server: l.server, id: id}
	}
	l.server.remember(id, out)
	return out, nil
}

// A trackConn drops itself from the server at its end. The Close runs 1 time
// only, because the handler of the server and CloseClientConnections each
// reach it.
type trackConn struct {
	net.Conn
	server *Server
	id     int
	once   sync.Once
}

func (c *trackConn) Close() error {
	var err error
	c.once.Do(func() {
		c.server.forget(c.id)
		err = c.Conn.Close()
	})
	return err
}

// A trackTLSConn also holds the 2 methods that the server reads for a
// connection of HTTP/2. A wrapper without them would hide the h2 form.
type trackTLSConn struct {
	trackConn
	secure *tls.Conn
}

func (c *trackTLSConn) ConnectionState() tls.ConnectionState {
	return c.secure.ConnectionState()
}

func (c *trackTLSConn) Handle() int { return c.secure.Handle() }
