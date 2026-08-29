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
)

// RoundTripper runs 1 request and gives its response.
type RoundTripper interface {
	RoundTrip(*Request) (*Response, error)
}

// A Transport opens 1 connection for each request and closes it after the
// body.
//
// The real Transport of Go holds a pool of connections, HTTP/2, a proxy, and a
// set of timeouts. The overlay holds the fields below, and each other field
// stays out.
type Transport struct {
	// TLSClientConfig holds the settings of a TLS connection. Read the package
	// crypto/tls for the fields that Dart honors.
	TLSClientConfig *tls.Config
	// DisableKeepAlives stays true. The overlay opens 1 connection for each
	// request.
	DisableKeepAlives bool
}

// DefaultTransport is the transport of DefaultClient.
var DefaultTransport RoundTripper = &Transport{DisableKeepAlives: true}

// RoundTrip opens a connection, writes the request, and reads the response.
// The body of the response holds the connection, thus a Close of the body ends
// it.
func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	if req.URL == nil {
		return nil, errors.New("http: nil Request.URL")
	}
	conn, err := t.dial(req)
	if err != nil {
		return nil, err
	}
	if deadline, ok := req.Context().Deadline(); ok {
		conn.SetDeadline(deadline)
	} else if !req.deadline.IsZero() {
		conn.SetDeadline(req.deadline)
	}
	if err := req.write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	res, err := readResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, err
	}
	res.Body = &connBody{body: res.Body, conn: conn}
	return res, nil
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

// A connBody holds the connection of a response. A Close of the body ends the
// connection, because the overlay holds no pool.
type connBody struct {
	body io.ReadCloser
	conn net.Conn
	done bool
}

func (c *connBody) Read(p []byte) (int, error) {
	n, err := c.body.Read(p)
	if err == io.EOF {
		c.Close()
	}
	return n, err
}

func (c *connBody) Close() error {
	if c.done {
		return nil
	}
	c.done = true
	c.body.Close()
	return c.conn.Close()
}
