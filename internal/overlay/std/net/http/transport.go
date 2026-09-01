//go:build dart

package http

import (
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syscall/dart"
	"time"
)

// RoundTripper runs 1 request and gives its response.
type RoundTripper interface {
	RoundTrip(*Request) (*Response, error)
}

// A Transport holds a pool of connections that stand open. A Close of the body
// of an answer gives the connection back to the pool.
//
// The real Transport of Go holds HTTP/2, a proxy, and a set of timeouts. The
// overlay holds the fields below, and each other field stays out.
type Transport struct {
	// TLSClientConfig holds the settings of a TLS connection. Read the package
	// crypto/tls for the fields that Dart honors.
	TLSClientConfig *tls.Config
	// DisableKeepAlives ends each connection after 1 answer.
	DisableKeepAlives bool
	// DisableCompression stops the field Accept-Encoding that the transport
	// adds.
	DisableCompression bool
	// MaxIdleConnsPerHost limits the connections that stand open for 1 host. A
	// value of 0 reads the default of 2.
	MaxIdleConnsPerHost int
	// ForceAttemptHTTP2 offers h2 on a dial with a TLSClientConfig of its
	// own. A transport with no TLSClientConfig offers h2 on each https dial,
	// exactly as in Go.
	ForceAttemptHTTP2 bool

	mu   sync.Mutex
	idle map[string][]net.Conn
	// h2 holds the HTTP/2 session of each host. 1 session carries each
	// request of its host, because HTTP/2 multiplexes the streams.
	h2 map[string]int
}

// DefaultTransport is the transport of DefaultClient.
var DefaultTransport RoundTripper = &Transport{}

// defaultMaxIdlePerHost is the count of connections that stand open for 1 host.
const defaultMaxIdlePerHost = 2

// RoundTrip writes the request and reads the response. The body of the response
// holds the connection, thus a Close of the body ends it or gives it back to
// the pool.
func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	if req.URL == nil {
		return nil, errors.New("http: nil Request.URL")
	}
	key, err := t.connectKey(req)
	if err != nil {
		return nil, err
	}
	for {
		// An open HTTP/2 session of the host carries the request. A session
		// that broke leaves the pool, and a request with no body rides a
		// fresh dial, exactly as a reused connection does.
		if session, ok := t.h2Of(key); ok {
			res, err := t.h2Exchange(session, req)
			if err == nil {
				return res, nil
			}
			t.dropH2(key, session)
			if req.Body == nil {
				continue
			}
			return nil, err
		}
		conn, reused, err := t.getConn(key, req)
		if err != nil {
			return nil, err
		}
		if tc, ok := conn.(*tls.Conn); ok &&
			tc.ConnectionState().NegotiatedProtocol == "h2" {
			session, cause := dartH2Client(tc)
			if cause != nil {
				conn.Close()
				return nil, cause
			}
			t.putH2(key, session)
			res, err := t.h2Exchange(session, req)
			if err != nil {
				t.dropH2(key, session)
				return nil, err
			}
			return res, nil
		}
		res, err := t.exchange(conn, req, key)
		if err == nil {
			return res, nil
		}
		conn.Close()
		// A connection of the pool can hold an end that the server wrote. A
		// request with no body reaches the wire a second time with no change of
		// meaning.
		if reused && req.Body == nil {
			continue
		}
		return nil, err
	}
}

// exchange writes 1 request and reads 1 answer on the connection.
func (t *Transport) exchange(conn net.Conn, req *Request, key string) (*Response, error) {
	if deadline, ok := req.Context().Deadline(); ok {
		conn.SetDeadline(deadline)
	} else if !req.deadline.IsZero() {
		conn.SetDeadline(req.deadline)
	}
	wire, gzipped := t.gzipRequest(req)
	if err := wire.write(conn); err != nil {
		return nil, err
	}
	res, err := readResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
	}
	if gzipped {
		unzip(res)
	}
	res.Body = &connBody{
		body:     res.Body,
		conn:     conn,
		pool:     t,
		key:      key,
		reusable: !t.DisableKeepAlives && !res.Close && !req.Close && res.bodyEnds,
	}
	return res, nil
}

// gzipRequest gives the request that reaches the wire, and it asks for the gzip
// form of the body. Go adds the field to a copy, thus the request of the caller
// keeps its header.
func (t *Transport) gzipRequest(req *Request) (*Request, bool) {
	if t.DisableCompression || req.Method == MethodHead {
		return req, false
	}
	if req.Header.Get("Accept-Encoding") != "" || req.Header.Get("Range") != "" {
		return req, false
	}
	out := *req
	out.Header = req.Header.Clone()
	if out.Header == nil {
		out.Header = Header{}
	}
	out.Header.Set("Accept-Encoding", "gzip")
	return &out, true
}

// unzip replaces the body of an answer of the gzip form. The length of the
// header names the bytes of the wire, thus it leaves the answer.
func unzip(res *Response) {
	if !strings.EqualFold(res.Header.Get("Content-Encoding"), "gzip") {
		return
	}
	res.Body = &gzipBody{source: res.Body}
	res.Header.Del("Content-Encoding")
	res.Header.Del("Content-Length")
	res.ContentLength = -1
	res.Uncompressed = true
}

// A gzipBody reads the gzip form of a body. The reader of the stream opens at
// the first read, because the header of the stream stands in the body.
type gzipBody struct {
	source io.ReadCloser
	reader *gzip.Reader
	err    error
}

func (g *gzipBody) Read(p []byte) (int, error) {
	if g.err != nil {
		return 0, g.err
	}
	if g.reader == nil {
		r, err := gzip.NewReader(g.source)
		if err != nil {
			g.err = err
			return 0, err
		}
		g.reader = r
	}
	return g.reader.Read(p)
}

func (g *gzipBody) Close() error { return g.source.Close() }

// connectKey names the pool of a request.
func (t *Transport) connectKey(req *Request) (string, error) {
	switch req.URL.Scheme {
	case "http", "https":
	default:
		return "", errors.New("http: the scheme " + strconv.Quote(req.URL.Scheme) +
			" is outside the subset")
	}
	host, port := hostPort(req.URL)
	return req.URL.Scheme + "|" + host + ":" + port, nil
}

// getConn gives a connection of the pool, or a new one. The second result marks
// a connection that already stood open.
func (t *Transport) getConn(key string, req *Request) (net.Conn, bool, error) {
	if conn := t.takeIdle(key); conn != nil {
		return conn, true, nil
	}
	conn, err := t.dial(req)
	if err != nil {
		return nil, false, err
	}
	return conn, false, nil
}

func (t *Transport) takeIdle(key string) net.Conn {
	if t.DisableKeepAlives {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	list := t.idle[key]
	if len(list) == 0 {
		return nil
	}
	conn := list[len(list)-1]
	t.idle[key] = list[:len(list)-1]
	return conn
}

// putIdle gives a connection back to the pool. A pool that is full ends the
// connection.
func (t *Transport) putIdle(key string, conn net.Conn) {
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return
	}
	limit := t.MaxIdleConnsPerHost
	if limit <= 0 {
		limit = defaultMaxIdlePerHost
	}
	t.mu.Lock()
	if len(t.idle[key]) >= limit {
		t.mu.Unlock()
		conn.Close()
		return
	}
	if t.idle == nil {
		t.idle = map[string][]net.Conn{}
	}
	t.idle[key] = append(t.idle[key], conn)
	t.mu.Unlock()
}

// CloseIdleConnections ends each connection that stands open with no request.
func (t *Transport) CloseIdleConnections() {
	t.mu.Lock()
	idle := t.idle
	t.idle = nil
	t.mu.Unlock()
	for _, list := range idle {
		for _, conn := range list {
			conn.Close()
		}
	}
}

func (t *Transport) dial(req *Request) (net.Conn, error) {
	host, port := hostPort(req.URL)
	switch req.URL.Scheme {
	case "http":
		return net.Dial("tcp", host+":"+port)
	case "https":
		config := t.TLSClientConfig
		if protos := t.protos(); len(protos) > 0 {
			config = config.Clone()
			if config == nil {
				config = &tls.Config{}
			}
			config.NextProtos = protos
		}
		c, err := tls.Dial("tcp", host+":"+port, config)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	return nil, errors.New("http: the scheme " + strconv.Quote(req.URL.Scheme) +
		" is outside the subset")
}

// hostPort splits the host of a URL. A URL with no port takes the port of its
// scheme.
func hostPort(u *url.URL) (string, string) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port
}

// A connBody holds the connection of a response. A Close of the body gives the
// connection back to the pool. A Close that stands before the end of the body
// ends the connection, exactly as in Go.
type connBody struct {
	body     io.ReadCloser
	conn     net.Conn
	pool     *Transport
	key      string
	reusable bool
	eof      bool
	done     bool
}

func (c *connBody) Read(p []byte) (int, error) {
	n, err := c.body.Read(p)
	if err == io.EOF {
		c.eof = true
	}
	return n, err
}

func (c *connBody) Close() error {
	if c.done {
		return nil
	}
	c.done = true
	c.body.Close()
	if c.reusable && c.eof {
		c.pool.putIdle(c.key, c.conn)
		return nil
	}
	return c.conn.Close()
}

// protos gives the list of ALPN that a dial offers. A config with a list of
// its own wins. A transport with no config offers h2, exactly as the default
// transport of Go does, and ForceAttemptHTTP2 offers h2 next to a config.
func (t *Transport) protos() []string {
	if t.TLSClientConfig != nil && len(t.TLSClientConfig.NextProtos) > 0 {
		return t.TLSClientConfig.NextProtos
	}
	if t.TLSClientConfig == nil || t.ForceAttemptHTTP2 {
		return []string{"h2", "http/1.1"}
	}
	return nil
}

// h2Of gives the open HTTP/2 session of a host.
func (t *Transport) h2Of(key string) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	session, ok := t.h2[key]
	return session, ok
}

func (t *Transport) putH2(key string, session int) {
	t.mu.Lock()
	if t.h2 == nil {
		t.h2 = map[string]int{}
	}
	t.h2[key] = session
	t.mu.Unlock()
}

// dropH2 ends a session that broke. Another goroutine can hold a newer
// session of the key already, thus the drop compares the value.
func (t *Transport) dropH2(key string, session int) {
	t.mu.Lock()
	if t.h2[key] == session {
		delete(t.h2, key)
	}
	t.mu.Unlock()
	dart.H2Close(session)
}

// dartH2Client wraps a connection whose ALPN took h2.
func dartH2Client(conn *tls.Conn) (int, error) {
	session, cause := dart.H2Client(conn.Handle())
	if cause != "" {
		return 0, errors.New("http2: " + cause)
	}
	return session, nil
}

// h2Exchange runs 1 request over an HTTP/2 session. The body of the request
// reaches the wire as 1 piece, because the fetch of the runtime takes the
// bytes.
func (t *Transport) h2Exchange(session int, req *Request) (*Response, error) {
	wire, gzipped := t.gzipRequest(req)
	var body []byte
	if wire.Body != nil {
		b, err := io.ReadAll(wire.Body)
		wire.Body.Close()
		if err != nil {
			return nil, err
		}
		body = b
	}
	authority := wire.Host
	if authority == "" {
		authority = wire.URL.Host
	}
	stream, status, flat, cause := dart.H2Fetch(session, wire.Method,
		wire.URL.RequestURI(), authority, h2Fields(wire), body, len(body) > 0)
	if cause != "" {
		return nil, errors.New("http2: " + cause)
	}
	header := Header{}
	for i := 0; i+1 < len(flat); i += 2 {
		header.Add(flat[i], flat[i+1])
	}
	length := int64(-1)
	if v := header.Get("Content-Length"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			length = n
		}
	}
	res := &Response{
		Status:        itoa(status) + " " + StatusText(status),
		StatusCode:    status,
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		Header:        header,
		ContentLength: length,
		Body:          &h2Body{stream: stream},
		Request:       req,
		bodyEnds:      true,
	}
	if gzipped {
		unzip(res)
	}
	return res, nil
}

// h2Fields gives the flat header pairs of a request. The fields of the
// connection of HTTP/1.1 stay out, and the host rides the authority.
func h2Fields(req *Request) []string {
	out := []string{"user-agent", defaultUserAgent2}
	for name, values := range req.Header {
		switch name {
		case "Host", "Connection", "Keep-Alive", "Proxy-Connection",
			"Transfer-Encoding", "Upgrade", "Content-Length":
			continue
		case "User-Agent":
			out[1] = values[len(values)-1]
			continue
		}
		for _, v := range values {
			out = append(out, name, v)
		}
	}
	return out
}

const defaultUserAgent2 = "Go-http-client/2.0"

// h2Body reads the body of an answer of HTTP/2.
type h2Body struct {
	stream int
	done   bool
}

func (b *h2Body) Read(p []byte) (int, error) {
	if b.done {
		return 0, errors.New("http: read on closed response body")
	}
	n, cause := dart.H2Read(b.stream, p)
	if cause != "" {
		return n, errors.New(cause)
	}
	if n == 0 && len(p) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (b *h2Body) Close() error {
	if b.done {
		return nil
	}
	b.done = true
	dart.H2End(b.stream)
	return nil
}
