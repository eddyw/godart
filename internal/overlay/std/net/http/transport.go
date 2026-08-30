//go:build dart

package http

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"sync"
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
	// MaxIdleConnsPerHost limits the connections that stand open for 1 host. A
	// value of 0 reads the default of 2.
	MaxIdleConnsPerHost int

	mu   sync.Mutex
	idle map[string][]net.Conn
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
		conn, reused, err := t.getConn(key, req)
		if err != nil {
			return nil, err
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
	if err := req.write(conn); err != nil {
		return nil, err
	}
	res, err := readResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
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
		c, err := tls.Dial("tcp", host+":"+port, t.TLSClientConfig)
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
