//go:build dart

package http

import (
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"syscall/dart"
)

// errFetchClosed is the cause that a read of a closed body gives.
var errFetchClosed = errors.New("net/http: reader is closed")

// fetchRoundTrip sends 1 request through the fetch API of the host.
//
// A browser holds no socket, thus the web target reaches the network through
// this path. The host follows a redirect on its own, thus CheckRedirect of a
// Client never runs there, and the host also decodes the gzip form. Go gives
// the same 2 answers under GOOS=js.
func (t *Transport) fetchRoundTrip(req *Request) (*Response, error) {
	var body []byte
	if req.Body != nil {
		read, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		body = read
	}
	fields := make([]string, 0, 2*len(req.Header))
	for name, values := range req.Header {
		for _, value := range values {
			fields = append(fields, name, value)
		}
	}
	handle, code, head, moved, cause := dart.FetchStart(
		req.Method, req.URL.String(), fields, body, fetchDeadline(req))
	if cause != "" {
		return nil, errors.New("net/http: fetch() failed: " + cause)
	}
	header := Header{}
	for i := 0; i+1 < len(head); i += 2 {
		key := CanonicalHeaderKey(head[i])
		header[key] = append(header[key], head[i+1])
	}
	length := int64(-1)
	if text := header.Get("Content-Length"); text != "" {
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil || n < 0 {
			dart.FetchClose(handle)
			return nil, errors.New(
				"net/http: invalid Content-Length header: " + strconv.Quote(text))
		}
		length = n
	}
	uncompressed := false
	if strings.EqualFold(header.Get("Content-Encoding"), "gzip") {
		header.Del("Content-Encoding")
		header.Del("Content-Length")
		length = -1
		uncompressed = true
	}
	if moved != "" {
		if target, err := url.Parse(moved); err == nil {
			req = req.WithContext(req.Context())
			req.URL = target
		}
	}
	return &Response{
		Status:        strconv.Itoa(code) + " " + StatusText(code),
		StatusCode:    code,
		Header:        header,
		ContentLength: length,
		Uncompressed:  uncompressed,
		Body:          &fetchBody{handle: handle},
		Request:       req,
	}, nil
}

// fetchDeadline gives the end of the request, in nanoseconds after the start
// of the Unix epoch. A value of 0 holds no limit.
func fetchDeadline(req *Request) int64 {
	if deadline, ok := req.Context().Deadline(); ok {
		return deadline.UnixNano()
	}
	if !req.deadline.IsZero() {
		return req.deadline.UnixNano()
	}
	return 0
}

// fetchBody reads the body of an answer of the fetch API.
type fetchBody struct {
	handle int
	closed bool
}

func (b *fetchBody) Read(p []byte) (int, error) {
	if b.closed {
		return 0, errFetchClosed
	}
	n, cause := dart.FetchRead(b.handle, p)
	if cause != "" {
		return n, errors.New(cause)
	}
	if n == 0 && len(p) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (b *fetchBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	if cause := dart.FetchClose(b.handle); cause != "" {
		return errors.New(cause)
	}
	return nil
}
