//go:build dart

// Package httptest holds a recorder of an answer and a server of a test.
//
// The real package of Go gives the certificate of a TLS server as a value of
// x509, and Close of it waits for each request that stands open. The surface
// holds no package crypto/x509, thus the overlay holds the names below.
package httptest

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
)

// NewRequest builds a request of a server, for a call of a handler.
func NewRequest(method, target string, body io.Reader) *http.Request {
	return NewRequestWithContext(context.Background(), method, target, body)
}

// NewRequestWithContext builds a request of a server that a context can stop.
//
// The target holds a path or an absolute URL. A target with no host takes the
// host `example.com`. An empty method reads GET. The function panics for a
// target that holds no valid form.
func NewRequestWithContext(ctx context.Context, method, target string, body io.Reader) *http.Request {
	if method == "" {
		method = http.MethodGet
	}
	// The line names HTTP/1.0, thus it needs no field Host.
	line := method + " " + target + " HTTP/1.0\r\n\r\n"
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(line)))
	if err != nil {
		panic("invalid NewRequest arguments; " + err.Error())
	}
	req = req.WithContext(ctx)
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = false
	req.Body = nil
	req.ContentLength = 0
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
		default:
			req.ContentLength = -1
		}
		if rc, ok := body.(io.ReadCloser); ok {
			req.Body = rc
		} else {
			req.Body = io.NopCloser(body)
		}
	}
	// The block 192.0.2.0/24 is TEST-NET of RFC 5737. It stands in an example
	// only.
	req.RemoteAddr = "192.0.2.1:1234"
	if req.Host == "" {
		req.Host = "example.com"
	}
	return req
}
