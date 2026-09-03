//go:build dart

package http

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"maps"
	"net"
	"net/url"
	"os"
	"runtime"
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
// The real Transport of Go holds a vendored HTTP/2 stack and a set of fields of
// that stack. The overlay holds the fields below, and each other field stays
// out. docs/parity.md holds the cause of each one.
type Transport struct {
	// TLSClientConfig holds the settings of a TLS connection. Read the package
	// crypto/tls for the fields that Dart honors.
	TLSClientConfig *tls.Config
	// DisableKeepAlives ends each connection after 1 answer.
	DisableKeepAlives bool
	// DisableCompression stops the field Accept-Encoding that the transport
	// adds.
	DisableCompression bool
	// MaxIdleConns limits the connections that stand open with no request over
	// each host. A value of 0 holds no limit.
	MaxIdleConns int
	// MaxIdleConnsPerHost limits the connections that stand open for 1 host. A
	// value of 0 reads DefaultMaxIdleConnsPerHost.
	MaxIdleConnsPerHost int
	// MaxConnsPerHost limits the connections of 1 host, and it counts the
	// connection of a request that runs. A request that reaches the limit
	// waits for a free connection. A value of 0 holds no limit.
	MaxConnsPerHost int
	// IdleConnTimeout ends a connection that stands open with no request for
	// that time. A value of 0 holds no limit.
	IdleConnTimeout time.Duration
	// ResponseHeaderTimeout limits the wait for the head of the answer, from
	// the end of the write of the request. A value of 0 holds no limit.
	ResponseHeaderTimeout time.Duration
	// ExpectContinueTimeout limits the wait for the answer 100 of a request
	// that carries the field Expect with the value 100-continue. A value of 0
	// sends the body at once.
	ExpectContinueTimeout time.Duration
	// MaxResponseHeaderBytes limits the bytes of the status line and of the
	// head of an answer. A value of 0 reads 10 megabytes, exactly as in Go.
	MaxResponseHeaderBytes int64
	// WriteBufferSize is the size of the buffer of the write of a request. A
	// value of 0 reads 4096.
	WriteBufferSize int
	// ReadBufferSize is the size of the buffer of the read of an answer. A
	// value of 0 reads 4096.
	ReadBufferSize int
	// ForceAttemptHTTP2 offers h2 on a dial with a TLSClientConfig of its
	// own. A transport with no TLSClientConfig offers h2 on each https dial,
	// exactly as in Go.
	ForceAttemptHTTP2 bool
	// Proxy gives the proxy of a request. A nil function and a nil result each
	// give a direct dial. Read ProxyFromEnvironment and ProxyURL.
	Proxy func(*Request) (*url.URL, error)
	// ProxyConnectHeader holds the fields that a CONNECT request to a proxy
	// carries. GetProxyConnectHeader wins over this field.
	ProxyConnectHeader Header
	// GetProxyConnectHeader gives the fields of 1 CONNECT request. An error
	// ends the request.
	GetProxyConnectHeader func(ctx context.Context, proxyURL *url.URL, target string) (Header, error)
	// Dial opens a plain connection. DialContext wins over it.
	//
	// A request of the scheme https reaches no such function, because the
	// secure socket of Dart takes no socket that stands open. Such a request
	// gives an error, and DialTLSContext holds the way. Go marks the field as
	// deprecated.
	Dial func(network, addr string) (net.Conn, error)
	// DialContext opens a plain connection. It wins over Dial, and a request
	// of the scheme https reaches it no time.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	// DialTLS opens a connection whose handshake already ran.
	// DialTLSContext wins over it.
	DialTLS func(network, addr string) (net.Conn, error)
	// DialTLSContext opens a connection whose handshake already ran. A result
	// that also holds ConnectionState and Handle can carry HTTP/2.
	DialTLSContext func(ctx context.Context, network, addr string) (net.Conn, error)

	mu sync.Mutex
	// idle holds the connections that stand open with no request, and
	// idleCount holds the count of them over each host.
	idle      map[string][]*persistConn
	idleCount int
	// stamp counts the entries of the idle pool, thus MaxIdleConns can name
	// the oldest connection.
	stamp int64
	// open holds the count of the connections of each host, and wait holds the
	// requests that stand in line for a free connection.
	open map[string]int
	wait map[string][]chan struct{}
	// h2 holds the HTTP/2 session of each host. 1 session carries each
	// request of its host, because HTTP/2 multiplexes the streams.
	h2 map[string]int
	// h2conn holds the connection of each HTTP/2 session. The end of a session
	// gives the place of the host back.
	h2conn map[string]*persistConn
	// alt holds the RoundTripper of each scheme that RegisterProtocol named.
	alt map[string]RoundTripper
	// running holds the connection of each request that stands open.
	// CancelRequest reads it.
	running map[*Request]*persistConn
}

// DefaultTransport is the transport of DefaultClient.
var DefaultTransport RoundTripper = &Transport{}

// DefaultMaxIdleConnsPerHost is the count of connections that stand open with
// no request for 1 host. A Transport with no MaxIdleConnsPerHost reads it.
const DefaultMaxIdleConnsPerHost = 2

// defaultBufferSize is the size of the buffer of a read and of a write of a
// connection. WriteBufferSize and ReadBufferSize each replace it.
const defaultBufferSize = 4096

// A persistConn holds 1 connection of the pool of a transport.
//
// The buffer of the read stands over the whole life of the connection, because
// the read of 1 answer can already hold bytes of the next answer.
type persistConn struct {
	t      *Transport
	key    string
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	once   sync.Once
	// timer ends the connection after IdleConnTimeout.
	timer *time.Timer
	// order names the place of the connection in the idle pool.
	order int64
}

// close ends the connection and gives its place of the host back. It runs 1
// time only, thus the count of the host stays right.
func (pc *persistConn) close() {
	pc.once.Do(func() {
		pc.conn.Close()
		pc.t.release(pc.key)
	})
}

// RegisterProtocol registers a RoundTripper for the scheme. A request of that
// scheme then reaches the RoundTripper, and a result of ErrSkipAltProtocol
// gives the request back to the transport.
//
// A scheme that already holds a RoundTripper stops the program.
func (t *Transport) RegisterProtocol(scheme string, rt RoundTripper) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.alt == nil {
		t.alt = map[string]RoundTripper{}
	}
	if _, ok := t.alt[scheme]; ok {
		panic("protocol " + scheme + " already registered")
	}
	t.alt[scheme] = rt
}

// altOf gives the RoundTripper of a scheme, or nil.
func (t *Transport) altOf(scheme string) RoundTripper {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.alt[scheme]
}

// Clone gives a copy of the transport with each exported field. The copy holds
// an empty pool, and it keeps each RoundTripper of RegisterProtocol.
func (t *Transport) Clone() *Transport {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := &Transport{
		TLSClientConfig:        t.TLSClientConfig.Clone(),
		DisableKeepAlives:      t.DisableKeepAlives,
		DisableCompression:     t.DisableCompression,
		MaxIdleConns:           t.MaxIdleConns,
		MaxIdleConnsPerHost:    t.MaxIdleConnsPerHost,
		MaxConnsPerHost:        t.MaxConnsPerHost,
		IdleConnTimeout:        t.IdleConnTimeout,
		ResponseHeaderTimeout:  t.ResponseHeaderTimeout,
		ExpectContinueTimeout:  t.ExpectContinueTimeout,
		MaxResponseHeaderBytes: t.MaxResponseHeaderBytes,
		WriteBufferSize:        t.WriteBufferSize,
		ReadBufferSize:         t.ReadBufferSize,
		ForceAttemptHTTP2:      t.ForceAttemptHTTP2,
		Proxy:                  t.Proxy,
		ProxyConnectHeader:     t.ProxyConnectHeader.Clone(),
		GetProxyConnectHeader:  t.GetProxyConnectHeader,
		Dial:                   t.Dial,
		DialContext:            t.DialContext,
		DialTLS:                t.DialTLS,
		DialTLSContext:         t.DialTLSContext,
	}
	if t.alt != nil {
		out.alt = maps.Clone(t.alt)
	}
	return out
}

// CancelRequest ends the connection of a request that stands open. Go marks the
// method as deprecated: a context of the request holds the same power, and it
// also reaches a request of HTTP/2.
//
// The request must be the request that RoundTrip read. A Client with a Timeout
// gives a copy of the request to the transport, and the method also reads that
// copy.
func (t *Transport) CancelRequest(req *Request) {
	t.mu.Lock()
	var pc *persistConn
	for k, v := range t.running {
		if k == req || k.origin == req {
			pc = v
			break
		}
	}
	t.mu.Unlock()
	if pc != nil {
		pc.close()
	}
}

// track writes the connection of a request that stands open, and a nil
// connection drops the entry.
func (t *Transport) track(req *Request, pc *persistConn) {
	t.mu.Lock()
	if pc == nil {
		delete(t.running, req)
		t.mu.Unlock()
		return
	}
	if t.running == nil {
		t.running = map[*Request]*persistConn{}
	}
	t.running[req] = pc
	t.mu.Unlock()
}

func (t *Transport) readBufferSize() int {
	if t.ReadBufferSize > 0 {
		return t.ReadBufferSize
	}
	return defaultBufferSize
}

func (t *Transport) writeBufferSize() int {
	if t.WriteBufferSize > 0 {
		return t.WriteBufferSize
	}
	return defaultBufferSize
}

// defaultMaxResponseHeaderBytes is the limit of the head of an answer that a
// Transport with no MaxResponseHeaderBytes reads. Go holds the same value.
const defaultMaxResponseHeaderBytes = 10 << 20

// headLimit gives the count of bytes that the head of an answer may hold.
func (t *Transport) headLimit() int {
	if t.MaxResponseHeaderBytes > 0 {
		return int(t.MaxResponseHeaderBytes)
	}
	return defaultMaxResponseHeaderBytes
}

// RoundTrip writes the request and reads the response. The body of the response
// holds the connection, thus a Close of the body ends it or gives it back to
// the pool.
func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	if req.URL == nil {
		return nil, errors.New("http: nil Request.URL")
	}
	// A RoundTripper of RegisterProtocol wins over the transport. A result of
	// ErrSkipAltProtocol gives the request back.
	if rt := t.altOf(req.URL.Scheme); rt != nil {
		res, err := rt.RoundTrip(req)
		if !errors.Is(err, ErrSkipAltProtocol) {
			return res, err
		}
	}
	// A RoundTripper of another scheme reads a URL with no host, thus the test
	// of the host stands after the dispatch, exactly as in Go.
	if req.URL.Host == "" {
		return nil, errors.New("http: no Host in request URL")
	}
	// A web target holds no socket. The value of GOOS names the target at run
	// time, because 1 build of the Go code serves each target. Read fetch.go.
	if runtime.GOOS == "js" {
		return t.fetchRoundTrip(req)
	}
	r, err := t.route(req)
	if err != nil {
		return nil, err
	}
	key := r.key
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
		pc, reused, err := t.getConn(req.Context(), r, req)
		if err != nil {
			return nil, err
		}
		if tc, ok := pc.conn.(h2Conn); ok &&
			tc.ConnectionState().NegotiatedProtocol == "h2" {
			session, cause := dartH2Client(tc.Handle())
			if cause != nil {
				pc.close()
				return nil, cause
			}
			t.putH2(key, session, pc)
			res, err := t.h2Exchange(session, req)
			if err != nil {
				t.dropH2(key, session)
				return nil, err
			}
			return res, nil
		}
		t.track(req, pc)
		res, err := t.exchange(pc, req, r)
		if err == nil {
			return res, nil
		}
		t.track(req, nil)
		pc.close()
		// A connection of the pool can hold an end that the server wrote. A
		// request with no body reaches the wire a second time with no change of
		// meaning.
		if reused && req.Body == nil {
			continue
		}
		return nil, err
	}
}

// setDeadline writes the deadline of the whole request on the connection.
func (t *Transport) setDeadline(pc *persistConn, req *Request) error {
	if deadline, ok := req.Context().Deadline(); ok {
		return pc.conn.SetDeadline(deadline)
	}
	if !req.deadline.IsZero() {
		return pc.conn.SetDeadline(req.deadline)
	}
	return pc.conn.SetDeadline(time.Time{})
}

// exchange writes 1 request and reads 1 answer on the connection.
func (t *Transport) exchange(pc *persistConn, req *Request, r route) (*Response, error) {
	if err := t.setDeadline(pc, req); err != nil {
		return nil, err
	}
	wire, gzipped := t.gzipRequest(req)
	if err := wire.writeHead(pc.writer, r.absolute(req)); err != nil {
		return nil, err
	}
	// A request that carries the field Expect with the value 100-continue
	// waits for the answer of the server before the body.
	var early *Response
	if wire.Body != nil && hasToken(wire.Header.Get("Expect"), "100-continue") {
		if err := pc.writer.Flush(); err != nil {
			return nil, err
		}
		res, err := t.awaitContinue(pc, req)
		if err != nil {
			return nil, err
		}
		early = res
	}
	if early == nil {
		if err := wire.writeBody(pc.writer); err != nil {
			return nil, err
		}
	}
	if err := pc.writer.Flush(); err != nil {
		return nil, err
	}
	res := early
	if res == nil {
		if d := t.ResponseHeaderTimeout; d > 0 {
			if err := pc.conn.SetReadDeadline(time.Now().Add(d)); err != nil {
				return nil, err
			}
		}
		out, err := readResponseLimit(pc.reader, req, t.headLimit())
		if err != nil {
			return nil, err
		}
		// An answer of the family 100 stands before the real answer.
		for out.StatusCode == StatusContinue {
			out, err = readResponseLimit(pc.reader, req, t.headLimit())
			if err != nil {
				return nil, err
			}
		}
		res = out
	}
	if err := t.setDeadline(pc, req); err != nil {
		return nil, err
	}
	if gzipped {
		unzip(res)
	}
	res.Body = &connBody{
		body:     res.Body,
		pc:       pc,
		req:      req,
		reusable: !t.DisableKeepAlives && !res.Close && !req.Close && res.bodyEnds,
	}
	return res, nil
}

// awaitContinue waits for the answer 100 of the server. A nil result asks for
// the body of the request, and an answer that holds a value ends the exchange
// with no body.
//
// An ExpectContinueTimeout of 0 sends the body at once, exactly as in Go.
func (t *Transport) awaitContinue(pc *persistConn, req *Request) (*Response, error) {
	d := t.ExpectContinueTimeout
	if d <= 0 {
		return nil, nil
	}
	if err := pc.conn.SetReadDeadline(time.Now().Add(d)); err != nil {
		return nil, err
	}
	res, err := readResponseLimit(pc.reader, req, t.headLimit())
	if err != nil {
		// The wait ran out, thus the body reaches the wire with no approval.
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil, nil
		}
		return nil, err
	}
	if res.StatusCode == StatusContinue {
		return nil, nil
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
// A route names the connection that 1 request needs.
type route struct {
	// key names the pool entry. 2 requests share a connection only when the 2
	// keys are equal.
	key string
	// proxy holds the proxy of the request, or nil.
	proxy *url.URL
}

// absolute reports whether the request line holds the whole URL. An http
// request through a proxy holds it, and each other request holds the path.
func (r route) absolute(req *Request) bool {
	return r.proxy != nil && req.URL.Scheme == "http"
}

func (t *Transport) route(req *Request) (route, error) {
	switch req.URL.Scheme {
	case "http", "https":
	default:
		return route{}, errors.New("http: the scheme " + strconv.Quote(req.URL.Scheme) +
			" is outside the subset")
	}
	var proxy *url.URL
	if t.Proxy != nil {
		out, err := t.Proxy(req)
		if err != nil {
			return route{}, err
		}
		proxy = out
	}
	host, port := hostPort(req.URL)
	key := req.URL.Scheme + "|" + host + ":" + port
	if proxy != nil {
		key += "|proxy " + proxy.String()
	}
	return route{key: key, proxy: proxy}, nil
}

// ProxyURL gives a Proxy function that always names 1 proxy.
func ProxyURL(fixedURL *url.URL) func(*Request) (*url.URL, error) {
	return func(*Request) (*url.URL, error) { return fixedURL, nil }
}

// ProxyFromEnvironment gives the proxy of a request from the environment. It
// reads HTTP_PROXY, HTTPS_PROXY, and NO_PROXY, and the lower case name of each
// one.
//
// A request of the scheme https reads HTTPS_PROXY, and each other request reads
// HTTP_PROXY. A host of NO_PROXY gives no proxy, and so does an empty value.
func ProxyFromEnvironment(req *Request) (*url.URL, error) {
	name := "HTTP_PROXY"
	if req.URL.Scheme == "https" {
		name = "HTTPS_PROXY"
	}
	value := envAny(name)
	if value == "" {
		return nil, nil
	}
	host, _ := hostPort(req.URL)
	if noProxy(envAny("NO_PROXY"), host) {
		return nil, nil
	}
	out, err := url.Parse(value)
	if err != nil || out.Host == "" {
		out, err = url.Parse("http://" + value)
		if err != nil {
			return nil, err
		}
	}
	switch out.Scheme {
	case "http", "https", "socks5":
	default:
		return nil, errors.New("http: the proxy scheme " + strconv.Quote(out.Scheme) +
			" is outside the subset")
	}
	return out, nil
}

// envAny reads the upper case name of a variable, and then the lower case name.
func envAny(name string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return os.Getenv(strings.ToLower(name))
}

// noProxy reports a host that the value of NO_PROXY names. A value of `*` names
// each host, and a name that starts with a point matches each host under it.
func noProxy(list, host string) bool {
	if list == "" {
		return false
	}
	if list == "*" {
		return true
	}
	host = strings.ToLower(host)
	for _, part := range strings.Split(list, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if part[0] == '.' {
			if strings.HasSuffix(host, part) {
				return true
			}
			continue
		}
		if host == part || strings.HasSuffix(host, "."+part) {
			return true
		}
	}
	return false
}

// getConn gives a connection of the pool, or a new one. The second result marks
// a connection that already stood open.
//
// A transport with a MaxConnsPerHost that the host reached waits for a free
// connection, or for the end of the context.
func (t *Transport) getConn(ctx context.Context, r route, req *Request) (*persistConn, bool, error) {
	for {
		if pc := t.takeIdle(r.key); pc != nil {
			return pc, true, nil
		}
		taken, wait := t.acquire(r.key)
		if taken {
			break
		}
		select {
		case <-wait:
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}
	conn, err := t.dial(ctx, r, req)
	if err != nil {
		t.release(r.key)
		return nil, false, err
	}
	pc := &persistConn{t: t, key: r.key, conn: conn}
	pc.reader = bufio.NewReaderSize(conn, t.readBufferSize())
	pc.writer = bufio.NewWriterSize(conn, t.writeBufferSize())
	return pc, false, nil
}

// acquire takes a place of the host. It gives false and a channel for a host
// that reached MaxConnsPerHost, and a close of that channel names a free place.
func (t *Transport) acquire(key string) (bool, chan struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.MaxConnsPerHost > 0 && t.open[key] >= t.MaxConnsPerHost {
		ch := make(chan struct{})
		if t.wait == nil {
			t.wait = map[string][]chan struct{}{}
		}
		t.wait[key] = append(t.wait[key], ch)
		return false, ch
	}
	if t.open == nil {
		t.open = map[string]int{}
	}
	t.open[key]++
	return true, nil
}

// release gives a place of the host back, and it wakes 1 request that waits.
func (t *Transport) release(key string) {
	t.mu.Lock()
	if n := t.open[key]; n > 0 {
		t.open[key] = n - 1
		if n == 1 {
			delete(t.open, key)
		}
	}
	ch := t.wakeLocked(key)
	t.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

// wakeLocked takes the first request that waits for a place of the host.
func (t *Transport) wakeLocked(key string) chan struct{} {
	list := t.wait[key]
	if len(list) == 0 {
		return nil
	}
	ch := list[0]
	if len(list) == 1 {
		delete(t.wait, key)
	} else {
		t.wait[key] = list[1:]
	}
	return ch
}

// takeIdle gives a connection of the pool of the host, or nil.
func (t *Transport) takeIdle(key string) *persistConn {
	if t.DisableKeepAlives {
		return nil
	}
	t.mu.Lock()
	list := t.idle[key]
	if len(list) == 0 {
		t.mu.Unlock()
		return nil
	}
	pc := list[len(list)-1]
	if len(list) == 1 {
		delete(t.idle, key)
	} else {
		t.idle[key] = list[:len(list)-1]
	}
	t.idleCount--
	timer := pc.timer
	pc.timer = nil
	t.mu.Unlock()
	if timer != nil {
		timer.Stop()
	}
	return pc
}

// putIdle gives a connection back to the pool. A pool of the host that is full,
// and a pool that reached MaxIdleConns, each end a connection.
func (t *Transport) putIdle(pc *persistConn) {
	if t.DisableKeepAlives {
		pc.close()
		return
	}
	if err := pc.conn.SetDeadline(time.Time{}); err != nil {
		pc.close()
		return
	}
	limit := t.MaxIdleConnsPerHost
	if limit <= 0 {
		limit = DefaultMaxIdleConnsPerHost
	}
	t.mu.Lock()
	if len(t.idle[pc.key]) >= limit {
		t.mu.Unlock()
		pc.close()
		return
	}
	var old *persistConn
	if t.MaxIdleConns > 0 && t.idleCount >= t.MaxIdleConns {
		old = t.dropOldestLocked()
	}
	if t.idle == nil {
		t.idle = map[string][]*persistConn{}
	}
	t.stamp++
	pc.order = t.stamp
	t.idle[pc.key] = append(t.idle[pc.key], pc)
	t.idleCount++
	if d := t.IdleConnTimeout; d > 0 {
		pc.timer = time.AfterFunc(d, func() { t.expire(pc) })
	}
	ch := t.wakeLocked(pc.key)
	t.mu.Unlock()
	if old != nil {
		old.close()
	}
	if ch != nil {
		close(ch)
	}
}

// dropOldestLocked takes the connection that stood in the pool the longest.
func (t *Transport) dropOldestLocked() *persistConn {
	var out *persistConn
	key := ""
	place := 0
	for k, list := range t.idle {
		for i, pc := range list {
			if out == nil || pc.order < out.order {
				out, key, place = pc, k, i
			}
		}
	}
	if out == nil {
		return nil
	}
	list := t.idle[key]
	t.idle[key] = append(list[:place], list[place+1:]...)
	if len(t.idle[key]) == 0 {
		delete(t.idle, key)
	}
	t.idleCount--
	if out.timer != nil {
		out.timer.Stop()
		out.timer = nil
	}
	return out
}

// expire ends a connection that stood idle for IdleConnTimeout.
func (t *Transport) expire(pc *persistConn) {
	t.mu.Lock()
	found := false
	list := t.idle[pc.key]
	for i, other := range list {
		if other != pc {
			continue
		}
		t.idle[pc.key] = append(list[:i], list[i+1:]...)
		if len(t.idle[pc.key]) == 0 {
			delete(t.idle, pc.key)
		}
		t.idleCount--
		found = true
		break
	}
	pc.timer = nil
	t.mu.Unlock()
	if found {
		pc.close()
	}
}

// CloseIdleConnections ends each connection that stands open with no request.
func (t *Transport) CloseIdleConnections() {
	t.mu.Lock()
	idle := t.idle
	t.idle = nil
	t.idleCount = 0
	t.mu.Unlock()
	for _, list := range idle {
		for _, pc := range list {
			if pc.timer != nil {
				pc.timer.Stop()
				pc.timer = nil
			}
			pc.close()
		}
	}
}

// dial opens a connection of the request.
func (t *Transport) dial(ctx context.Context, r route, req *Request) (net.Conn, error) {
	host, port := hostPort(req.URL)
	address := host + ":" + port
	switch req.URL.Scheme {
	case "http":
		target := address
		if r.proxy != nil {
			proxyHost, proxyPort := hostPort(r.proxy)
			target = proxyHost + ":" + proxyPort
		}
		if f := t.DialContext; f != nil {
			return f(ctx, "tcp", target)
		}
		if f := t.Dial; f != nil {
			return f("tcp", target)
		}
		return net.Dial("tcp", target)
	case "https":
		if f := t.DialTLSContext; f != nil {
			return f(ctx, "tcp", address)
		}
		if f := t.DialTLS; f != nil {
			return f("tcp", address)
		}
		if t.Dial != nil || t.DialContext != nil {
			return nil, errors.New("http: Dial and DialContext reach no request of " +
				"the scheme https, because the secure socket of Dart takes no socket " +
				"that stands open. Read DialTLSContext.")
		}
		config := t.TLSClientConfig
		if protos := t.protos(); len(protos) > 0 {
			config = config.Clone()
			if config == nil {
				config = &tls.Config{}
			}
			config.NextProtos = protos
		}
		if r.proxy != nil {
			proxyHost, proxyPort := hostPort(r.proxy)
			head, err := t.connectHead(ctx, address, r.proxy)
			if err != nil {
				return nil, err
			}
			return tls.DialProxy(proxyHost+":"+proxyPort, address, head, config)
		}
		c, err := tls.Dial("tcp", address, config)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	return nil, errors.New("http: the scheme " + strconv.Quote(req.URL.Scheme) +
		" is outside the subset")
}

// connectHead writes the CONNECT request that opens a tunnel through a proxy.
// ProxyConnectHeader and GetProxyConnectHeader each add fields to it.
func (t *Transport) connectHead(ctx context.Context, target string, proxy *url.URL) (string, error) {
	head := "CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n"
	if proxy.User != nil {
		name := proxy.User.Username()
		secret, _ := proxy.User.Password()
		head += "Proxy-Authorization: Basic " +
			base64.StdEncoding.EncodeToString([]byte(name+":"+secret)) + "\r\n"
	}
	extra := t.ProxyConnectHeader
	if t.GetProxyConnectHeader != nil {
		out, err := t.GetProxyConnectHeader(ctx, proxy, target)
		if err != nil {
			return "", err
		}
		extra = out
	}
	var b strings.Builder
	b.WriteString(head)
	if len(extra) > 0 {
		if err := extra.WriteSubset(&b, map[string]bool{"Host": true}); err != nil {
			return "", err
		}
	}
	if extra.Get("User-Agent") == "" {
		b.WriteString("User-Agent: " + defaultUserAgent + "\r\n")
	}
	b.WriteString("\r\n")
	return b.String(), nil
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
	pc       *persistConn
	req      *Request
	reusable bool
	eof      bool
	done     bool
}

func (c *connBody) Read(p []byte) (int, error) {
	if c.done {
		return 0, ErrBodyReadAfterClose
	}
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
	c.pc.t.track(c.req, nil)
	if c.reusable && c.eof {
		c.pc.t.putIdle(c.pc)
		return nil
	}
	c.pc.close()
	return nil
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

func (t *Transport) putH2(key string, session int, pc *persistConn) {
	t.mu.Lock()
	if t.h2 == nil {
		t.h2 = map[string]int{}
		t.h2conn = map[string]*persistConn{}
	}
	t.h2[key] = session
	t.h2conn[key] = pc
	t.mu.Unlock()
}

// dropH2 ends a session that broke. Another goroutine can hold a newer
// session of the key already, thus the drop compares the value.
func (t *Transport) dropH2(key string, session int) {
	t.mu.Lock()
	var pc *persistConn
	if t.h2[key] == session {
		delete(t.h2, key)
		pc = t.h2conn[key]
		delete(t.h2conn, key)
	}
	t.mu.Unlock()
	dart.H2Close(session)
	// The socket of the session belongs to the runtime, thus the close of the
	// connection gives the place of the host back and nothing else.
	if pc != nil {
		pc.close()
	}
}

// dartH2Client wraps a connection whose ALPN took h2.
func dartH2Client(handle int) (int, error) {
	session, cause := dart.H2Client(handle)
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
