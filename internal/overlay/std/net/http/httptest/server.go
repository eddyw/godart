//go:build dart

package httptest

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// A Server holds a listener on a port of the machine and answers each request
// with a handler.
//
// The real Server of Go waits for each request that stands open. The overlay
// does not.
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
	s.client = &http.Client{Transport: &http.Transport{}}
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

// Certificate gives the certificate of a TLS server. A server that StartTLS
// did not start gives nil.
func (s *Server) Certificate() *x509.Certificate {
	if s.TLS == nil || len(s.TLS.Certificates) == 0 {
		return nil
	}
	return s.TLS.Certificates[0].Leaf
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

// testCert holds the certificate of net/http/internal/testcert, which the
// httptest of Go also uses. It names 127.0.0.1, ::1, example.com, and
// *.example.com, and it is valid to the year 2084. A program that reads
// Server.Certificate thus finds the values of Go. The pair serves tests only.
const testCert = `-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIQEP/md970HysdBTpuzDOf0DANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAxcl69ROJdxjN+MJZnbFrYxyQooADCsJ6VDkuMyNQIix/Hk15Nk/u
FyBX1Me++aEpGmY3RIY4fUvELqT/srvAHsTXwVVSttMcY8pcAFmXSqo3x4MuUTG/
jCX3Vftj0r3EM5M8ImY1rzA/jqTTLJg00rD+DmuDABcqQvoXw/RV8w1yTRi5BPoH
DFD/AWTt/YgMvk1l2Yq/xI8VbMUIpjBoGXxWsSevQ5i2s1mk9/yZzu0Ysp1tTlzD
qOPa4ysFjBitdXiwfxjxtv5nXqOCP5rheKO0sWLk0fetMp1OV5JSJMAJw6c2ZMkl
U2WMqAEpRjdE/vHfIuNg+yGaRRqI07NZRQIDAQABo4GXMIGUMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBQR5QIzmacmw78ZI1C4MXw7Q0wJ1jA9BgNVHREENjA0ggtleGFtcGxlLmNv
bYINKi5leGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG
9w0BAQsFAAOCAQEACrRNgiioUDzxQftd0fwOa6iRRcPampZRDtuaF68yNHoNWbOu
LUwc05eOWxRq3iABGSk2xg+FXM3DDeW4HhAhCFptq7jbVZ+4Jj6HeJG9mYRatAxR
Y/dEpa0D0EHhDxxVg6UzKOXB355n0IetGE/aWvyTV9SiDs6QsaC57Q9qq1/mitx5
2GFBoapol9L5FxCc77bztzK8CpLujkBi25Vk6GAFbl27opLfpyxkM+rX/T6MXCPO
6/YBacNZ7ff1/57Etg4i5mNA6ubCpuc4Gi9oYqCNNohftr2lkJr7REdDR6OW0lsL
rF7r4gUnKeC7mYIH1zypY7laskopiLFAfe96Kg==
-----END CERTIFICATE-----`

// testKey gives the private key of testCert. The text holds the word TESTING,
// thus a scanner of secrets does not read a live key, exactly as the test
// certificate of Go does.
//
// Go labels the block RSA PRIVATE KEY, and the bytes hold the PKCS number 8
// form. X509KeyPair of Go reads each form and ignores the label, and the
// secure socket of Dart reads the label. The block below thus carries the
// label of the form that it holds.
func testKey() string {
	return strings.ReplaceAll(testKeyMarked, "TESTING KEY", "PRIVATE KEY")
}

const testKeyMarked = `-----BEGIN TESTING KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDFyXr1E4l3GM34
wlmdsWtjHJCigAMKwnpUOS4zI1AiLH8eTXk2T+4XIFfUx775oSkaZjdEhjh9S8Qu
pP+yu8AexNfBVVK20xxjylwAWZdKqjfHgy5RMb+MJfdV+2PSvcQzkzwiZjWvMD+O
pNMsmDTSsP4Oa4MAFypC+hfD9FXzDXJNGLkE+gcMUP8BZO39iAy+TWXZir/EjxVs
xQimMGgZfFaxJ69DmLazWaT3/JnO7RiynW1OXMOo49rjKwWMGK11eLB/GPG2/mde
o4I/muF4o7SxYuTR960ynU5XklIkwAnDpzZkySVTZYyoASlGN0T+8d8i42D7IZpF
GojTs1lFAgMBAAECggEAIYthUi1lFBDd5gG4Rzlu+BlBIn5JhcqkCqLEBiJIFfOr
/4yuMRrvS3bNzqWt6xJ9MSAC4ZlN/VobRLnxL/QNymoiGYUKCT3Ww8nvPpPzR9OE
sE68TUL9tJw/zZJcRMKwgvrGqSLimfq53MxxkE+kLdOc0v9C8YH8Re26mB5ZcWYa
7YFyZQpKsQYnsmu/05cMbpOQrQWhtmIqRoyn8mG/par2s3NzjtpSE9NINyz26uFc
k/3ovFJQIHkUmTS7KHD3BgY5vuCqP98HramYnOysJ0WoYgvSDNCWw3037s5CCwJT
gCKuM+Ow6liFrj83RrdKBpm5QUGjfNpYP31o+QNP4QKBgQDSrUQ2XdgtAnibAV7u
7kbxOxro0EhIKso0Y/6LbDQgcXgxLqltkmeqZgG8nC3Z793lhlSasz2snhzzooV5
5fTy1y8ikXqjhG0nNkInFyOhsI0auE28CFoDowaQd+5cmCatpN4Grqo5PNRXxm1w
HktfPEgoP11NNCFHvvN5fEKbbQKBgQDwVlOaV20IvW3IPq7cXZyiyabouFF9eTRo
VJka1Uv+JtyvL2P0NKkjYHOdN8gRblWqxQtJoTNk020rVA4UP1heiXALy50gvj/p
hMcybPTLYSPOhAGx838KIcvGR5oskP1aUCmFbFQzGELxhJ9diVVjxUtbG2DuwPKd
tD9TLxT2OQKBgQCcdlHSjp+dzdgERmBa0ludjGfPv9/uuNizUBAbO6D690psPFtY
JQMYaemgSd1DngEOFVWADt4e9M5Lose+YCoqr+UxpxmNlyv5kzJOFcFAs/4XeglB
PHKdgNW/NVKxMc6H54l9LPr+x05sYdGlEtqnP/3W5jhEvhJ5Vjc8YiyVgQKBgQCl
zwjyrGo+42GACy7cPYE5FeIfIDqoVByB9guC5bD98JXEDu/opQQjsgFRcBCJZhOY
M0UsURiB8ROaFu13rpQq9KrmmF0ZH+g8FSzQbzcbsTLg4VXCDXmR5esOKowFPypr
Sm667BfTAGP++D5ya7MLmCv6+RKQ5XD8uEQQAaV2kQKBgAD8qeJuWIXZT0VKkQrn
nIhgtzGERF/6sZdQGW2LxTbUDWG74AfFkkEbeBfwEkCZXY/xmnYqYABhvlSex8jU
supU6Eea21esIxIub2zv/Np0ojUb6rlqTPS4Ox1E27D787EJ3VOXpriSD10vyNnZ
jel6uj2FOP9g54s+GzlSVg/T
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
		s.client = &http.Client{Transport: &http.Transport{}}
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
