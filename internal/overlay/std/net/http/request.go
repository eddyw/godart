//go:build dart

package http

import (
	"bufio"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// A Request holds 1 HTTP request. A client builds one, and a server reads one.
type Request struct {
	Method string
	URL    *url.URL
	Proto  string
	// ProtoMajor and ProtoMinor name the version of the protocol. The overlay
	// holds HTTP/1.1 only.
	ProtoMajor int
	ProtoMinor int
	Header     Header
	Body       io.ReadCloser
	// ContentLength holds the count of bytes of the body. A value of -1 marks a
	// length that the header does not name.
	ContentLength int64
	// Host holds the host of the request. An empty host reads the URL.
	Host string
	// RemoteAddr stays empty. The Conn of the net overlay names no peer.
	RemoteAddr string
	// RequestURI holds the text of the request line that a server read.
	RequestURI string
	// Close asks for the end of the connection after the answer.
	Close bool
	// Form holds the values of the query of the URL and of the body of a form.
	// ParseForm fills it.
	Form url.Values
	// PostForm holds the values of the body of a form alone.
	PostForm url.Values
	// MultipartForm holds the parts of a body of the type multipart/form-data.
	// ParseMultipartForm fills it.
	MultipartForm *multipart.Form

	ctx context.Context
	// deadline holds the end that the Timeout of a client gives.
	deadline time.Time
}

// withDeadline gives a copy of the request with the end of its wait.
func (r *Request) withDeadline(t time.Time) *Request {
	out := *r
	out.deadline = t
	return &out
}

// NewRequest builds a request. The body reaches the wire when it holds a value.
func NewRequest(method, target string, body io.Reader) (*Request, error) {
	return NewRequestWithContext(context.Background(), method, target, body)
}

// NewRequestWithContext builds a request that a context can stop.
func NewRequestWithContext(ctx context.Context, method, target string, body io.Reader) (*Request, error) {
	if method == "" {
		method = MethodGet
	}
	if !validMethod(method) {
		return nil, errors.New("net/http: invalid method " + strconv.Quote(method))
	}
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, errors.New("http: no Host in request URL " + strconv.Quote(target))
	}
	req := &Request{
		Method:        method,
		URL:           u,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        Header{},
		Host:          u.Host,
		ContentLength: -1,
		ctx:           ctx,
	}
	if body != nil {
		rc, ok := body.(io.ReadCloser)
		if !ok {
			rc = io.NopCloser(body)
		}
		req.Body = rc
		if n, ok := knownLength(body); ok {
			req.ContentLength = n
		}
	} else {
		req.ContentLength = 0
	}
	return req, nil
}

// knownLength gives the count of bytes of a body whose type names it.
func knownLength(body io.Reader) (int64, bool) {
	if v, ok := body.(interface{ Len() int }); ok {
		return int64(v.Len()), true
	}
	return 0, false
}

// Context gives the context of the request.
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// WithContext gives a copy of the request with the context.
func (r *Request) WithContext(ctx context.Context) *Request {
	out := *r
	out.ctx = ctx
	return &out
}

// UserAgent gives the value of the field User-Agent.
func (r *Request) UserAgent() string { return r.Header.Get("User-Agent") }

// Referer gives the value of the field Referer.
func (r *Request) Referer() string { return r.Header.Get("Referer") }

// write writes the request in the form of the wire.
//
// A request that rides an HTTP proxy holds the whole URL in the request line,
// and each other request holds the path.
func (r *Request) write(w io.Writer, absolute bool) error {
	host := r.Host
	if host == "" && r.URL != nil {
		host = r.URL.Host
	}
	path := "/"
	if r.URL != nil {
		path = r.URL.RequestURI()
		if absolute {
			path = r.URL.Scheme + "://" + r.URL.Host + path
		}
	}
	head := r.Method + " " + path + " HTTP/1.1\r\n" + "Host: " + host + "\r\n"
	if _, err := io.WriteString(w, head); err != nil {
		return err
	}
	header := r.Header.Clone()
	if header == nil {
		header = Header{}
	}
	if header.Get("User-Agent") == "" {
		header.Set("User-Agent", defaultUserAgent)
	}
	if r.Body != nil && r.ContentLength >= 0 {
		header.Set("Content-Length", strconv.FormatInt(r.ContentLength, 10))
	}
	header.Del("Host")
	if err := header.Write(w); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	if r.Body == nil {
		return nil
	}
	_, err := io.Copy(w, r.Body)
	r.Body.Close()
	return err
}

const defaultUserAgent = "Go-http-client/1.1"

func validMethod(m string) bool {
	if m == "" {
		return false
	}
	for i := 0; i < len(m); i++ {
		if !validTokenByte(m[i]) {
			return false
		}
	}
	return true
}

// ReadRequest reads 1 request from the reader.
func ReadRequest(b *bufio.Reader) (*Request, error) { return readRequest(b) }

// readRequest reads 1 request from a connection. A server calls it.
func readRequest(r *bufio.Reader) (*Request, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	method, rest, ok := strings.Cut(line, " ")
	if !ok {
		return nil, errors.New("malformed HTTP request " + strconv.Quote(line))
	}
	target, proto, ok := strings.Cut(rest, " ")
	if !ok {
		return nil, errors.New("malformed HTTP request " + strconv.Quote(line))
	}
	u, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, err
	}
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	// Go holds the host of a request in the field Host, and not in the header.
	host := header.Get("Host")
	header.Del("Host")
	req := &Request{
		Method:     method,
		URL:        u,
		Proto:      proto,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Host:       host,
		RequestURI: target,
		Close:      strings.EqualFold(header.Get("Connection"), "close"),
	}
	if u.Host == "" {
		req.URL.Host = req.Host
	}
	body, length, _, err := readBody(r, header, method != MethodHead, false)
	if err != nil {
		return nil, err
	}
	req.Body = body
	req.ContentLength = length
	return req, nil
}
