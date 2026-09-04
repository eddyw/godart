//go:build dart

package http

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"net/http/httptrace"
	"net/http/internal/ascii"
	"net/textproto"
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
	// RemoteAddr holds the address of the peer, in the form host:port. A
	// server writes it, and a client leaves it empty.
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
	// GetBody gives a fresh copy of the body. A client reads it for a redirect
	// and for a second try of a request. NewRequest fills it for a body of the
	// 3 types that name their length.
	GetBody func() (io.ReadCloser, error)
	// Pattern holds the text of the pattern of the ServeMux that matched the
	// request. It stays empty for a request that no ServeMux matched.
	Pattern string
	// Response holds the answer that gave the redirect of this request. Only a
	// redirect of a Client writes it.
	Response *Response
	// TransferEncoding lists the encoding of the body, from the outermost to
	// the innermost. The overlay writes the 1 element "chunked", or nothing.
	TransferEncoding []string
	// Trailer holds the fields that follow the body. The overlay reads no
	// trailer and writes none, thus this map stays nil on a server. A request
	// that declares a Trailer field gives the status 501, thus no trailer is
	// lost with no report.
	Trailer Header
	// TLS holds the state of the secure connection that carried the request. A
	// server writes it, and a plain connection leaves it nil.
	//
	// The ConnectionState of the overlay holds 3 fields. Read docs/parity.md.
	TLS *tls.ConnectionState

	// Cancel names a channel whose close ends the request. The transport
	// closes the connection of the request at that point, exactly as
	// Transport.CancelRequest does. Go marks the field as deprecated: a
	// context of the request holds the same power.
	//
	// A server request holds no such channel.
	Cancel <-chan struct{}

	ctx context.Context
	// deadline holds the end that the Timeout of a client gives.
	deadline time.Time
	// pat holds the pattern that matched the request, and matches holds the
	// text of each named wildcard of it, in the order of the pattern.
	pat     *pattern
	matches []string
	// otherValues holds each value that SetPathValue wrote for a name that the
	// pattern does not hold.
	otherValues map[string]string
	// origin names the request that a copy of withDeadline came from.
	// Transport.CancelRequest reads it, thus the caller can name either 1.
	origin *Request
	// extra holds each field that the transport adds. It follows the header of
	// the caller on the wire, exactly as in Go.
	extra Header
}

// withDeadline gives a copy of the request with the end of its wait.
func (r *Request) withDeadline(t time.Time) *Request {
	out := *r
	out.deadline = t
	out.origin = r
	if r.origin != nil {
		out.origin = r.origin
	}
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
	// Go names no host here. The transport reports a URL with no host, and a
	// RoundTripper of RegisterProtocol reads such a URL, for example the
	// scheme file.
	req := &Request{
		Method:     method,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     Header{},
		Host:       u.Host,
		ctx:        ctx,
	}
	if body != nil {
		rc, ok := body.(io.ReadCloser)
		if !ok {
			rc = io.NopCloser(body)
		}
		req.Body = rc
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
			buf := v.Bytes()
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(buf)), nil
			}
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		default:
			// A body of another type names no length, thus the request holds
			// the value -1 and the transport writes the chunked form.
		}
		if req.GetBody != nil && req.ContentLength == 0 {
			req.Body = NoBody
			req.GetBody = func() (io.ReadCloser, error) { return NoBody, nil }
		}
	}
	return req, nil
}

// outgoingLength gives the count of bytes of the body that the wire names. A
// value below 0 marks a body whose length no field can name, thus the request
// takes the chunked form.
//
// A ContentLength of 0 next to a body that holds a value names no length, and
// Go holds the same rule.
func (r *Request) outgoingLength() int64 {
	if r.Body == nil || r.Body == NoBody {
		return 0
	}
	if r.ContentLength != 0 {
		return r.ContentLength
	}
	return -1
}

// sendsZeroLength reports a method whose request carries the field
// Content-Length with the value 0 when the body holds no byte.
func sendsZeroLength(method string) bool {
	switch method {
	case MethodPost, MethodPut, MethodPatch:
		return true
	}
	return false
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

// Write writes the request in the form of the wire. The request line holds the
// path of the URL, and the field Host holds its host.
//
// The call reads and closes the body of the request.
func (r *Request) Write(w io.Writer) error { return r.write(w, false) }

// WriteProxy writes the request in the form that an HTTP proxy reads. The
// request line then holds the whole URL.
//
// The call reads and closes the body of the request.
func (r *Request) WriteProxy(w io.Writer) error { return r.write(w, true) }

// write writes the head and the body of the request.
func (r *Request) write(w io.Writer, absolute bool) error {
	if err := r.writeHead(w, absolute, nil); err != nil {
		return err
	}
	return r.writeBody(w)
}

// writeHead writes the request line and each field. The order of the fields is
// the order of Go: the request line, Host, User-Agent, the length of the body,
// and then each other field in order.
//
// A request that rides an HTTP proxy holds the whole URL in the request line,
// and each other request holds the path.
func (r *Request) writeHead(w io.Writer, absolute bool, trace *httptrace.ClientTrace) error {
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
	wroteField(trace, "Host", host)
	// A header that holds the field User-Agent with an empty value writes no
	// such line, exactly as in Go.
	agent := defaultUserAgent
	if r.Header.has("User-Agent") {
		agent = r.Header.Get("User-Agent")
	}
	if agent != "" {
		if _, err := io.WriteString(w, "User-Agent: "+agent+"\r\n"); err != nil {
			return err
		}
		wroteField(trace, "User-Agent", agent)
	}
	key, value := "", ""
	switch length := r.outgoingLength(); {
	case length < 0:
		key, value = "Transfer-Encoding", "chunked"
	case length > 0:
		key, value = "Content-Length", strconv.FormatInt(length, 10)
	case sendsZeroLength(r.Method):
		key, value = "Content-Length", "0"
	}
	if key != "" {
		if _, err := io.WriteString(w, key+": "+value+"\r\n"); err != nil {
			return err
		}
		wroteField(trace, key, value)
	}
	if err := r.Header.writeSubset(w, reqWriteExclude, trace); err != nil {
		return err
	}
	if err := r.extra.writeSubset(w, reqWriteExclude, trace); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	if trace != nil && trace.WroteHeaders != nil {
		trace.WroteHeaders()
	}
	return nil
}

// wroteField reports 1 field of the head of a request to a client trace.
func wroteField(trace *httptrace.ClientTrace, key, value string) {
	if trace != nil && trace.WroteHeaderField != nil {
		trace.WroteHeaderField(key, []string{value})
	}
}

// reqWriteExclude names each field that writeHead already wrote.
var reqWriteExclude = map[string]bool{
	"Host":              true,
	"User-Agent":        true,
	"Content-Length":    true,
	"Transfer-Encoding": true,
	"Trailer":           true,
}

// writeBody writes the body of the request, and it closes the body. A body
// whose length no field names takes the chunked form.
func (r *Request) writeBody(w io.Writer) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	if r.outgoingLength() < 0 {
		return writeChunked(w, r.Body)
	}
	_, err := io.Copy(w, r.Body)
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
func ReadRequest(b *bufio.Reader) (*Request, error) { return readRequestLimit(b, 0, 0) }

// readRequestLimit reads 1 request from a connection. A limit above 0 names the
// count of bytes that the request line and the head hold together, and a
// maxValues above 0 names the count of the lines of the head. A head that holds
// more gives errHeaderTooLarge.
func readRequestLimit(r *bufio.Reader, limit, maxValues int) (*Request, error) {
	left := -1
	if limit > 0 {
		left = limit
	}
	line, err := readLineLimit(r, &left)
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
	major, minor, ok := ParseHTTPVersion(proto)
	if !ok {
		return nil, errors.New("malformed HTTP version " + strconv.Quote(proto))
	}
	u, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, err
	}
	header, err := readHeaderLimit(r, &left, maxValues)
	if err != nil {
		return nil, err
	}
	// RFC 7230, section 5.3 holds the 2 forms equal: a request line with the
	// whole URL, and a request line with the path next to a field Host. The
	// URL wins, exactly as in Go, and the field Host then reaches no reader.
	host := u.Host
	if host == "" {
		host = header.Get("Host")
	}
	header.Del("Host")
	req := &Request{
		Method:     method,
		URL:        u,
		Proto:      proto,
		ProtoMajor: major,
		ProtoMinor: minor,
		Header:     header,
		Host:       host,
		RequestURI: target,
	}
	req.Close = shouldClose(major, minor, header, false)
	body, length, _, err := readBody(r, header, method != MethodHead, false)
	if err != nil {
		return nil, err
	}
	req.Body = body
	req.ContentLength = length
	if length == -1 {
		req.TransferEncoding = []string{"chunked"}
	}
	return req, nil
}

// ParseHTTPVersion reads the version of a request line or of a status line.
// The text "HTTP/1.0" gives (1, 0, true). A text with no minor number, such as
// "HTTP/2", is not valid.
func ParseHTTPVersion(vers string) (major, minor int, ok bool) {
	switch vers {
	case "HTTP/1.1":
		return 1, 1, true
	case "HTTP/1.0":
		return 1, 0, true
	}
	if !strings.HasPrefix(vers, "HTTP/") {
		return 0, 0, false
	}
	if len(vers) != len("HTTP/X.Y") {
		return 0, 0, false
	}
	if vers[6] != '.' {
		return 0, 0, false
	}
	maj, err := strconv.ParseUint(vers[5:6], 10, 0)
	if err != nil {
		return 0, 0, false
	}
	min, err := strconv.ParseUint(vers[7:8], 10, 0)
	if err != nil {
		return 0, 0, false
	}
	return int(maj), int(min), true
}

// shouldClose reports whether the connection ends after the message. A message
// of HTTP/1.0 keeps the connection only for a field Connection that names
// keep-alive.
func shouldClose(major, minor int, header Header, removeCloseHeader bool) bool {
	if major < 1 {
		return true
	}
	values := header.Values("Connection")
	hasClose := headerHasToken(values, "close")
	if major == 1 && minor == 0 {
		return hasClose || !headerHasToken(values, "keep-alive")
	}
	if hasClose && removeCloseHeader {
		header.Del("Connection")
	}
	return hasClose
}

// headerHasToken reports whether 1 value of the list holds the token. The
// comparison ignores the case, and a comma divides the tokens of 1 value.
func headerHasToken(values []string, token string) bool {
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

// ProtoAtLeast reports whether the version of the protocol of the request
// reaches the major and the minor number.
func (r *Request) ProtoAtLeast(major, minor int) bool {
	return r.ProtoMajor > major ||
		r.ProtoMajor == major && r.ProtoMinor >= minor
}

// PathValue gives the text of the wildcard of the name in the pattern that
// matched the request. It gives an empty string for a request that no pattern
// matched, and for a name that the pattern does not hold.
//
// The text holds no escape. The pattern "/b/{bucket}" against the path
// "/b/a%2fb" thus gives "a/b".
func (r *Request) PathValue(name string) string {
	if i := r.patIndex(name); i >= 0 {
		return r.matches[i]
	}
	return r.otherValues[name]
}

// SetPathValue writes the value of a name, thus each PathValue of that name
// gives the value. It writes no escape.
func (r *Request) SetPathValue(name, value string) {
	if i := r.patIndex(name); i >= 0 {
		r.matches[i] = value
		return
	}
	if r.otherValues == nil {
		r.otherValues = map[string]string{}
	}
	r.otherValues[name] = value
}

// patIndex gives the place of the name in the list of the named wildcards of
// the pattern of the request, and it gives -1 for a name outside it.
func (r *Request) patIndex(name string) int {
	if r.pat == nil {
		return -1
	}
	i := 0
	for _, seg := range r.pat.segments {
		if seg.wild && seg.s != "" {
			if name == seg.s {
				return i
			}
			i++
		}
	}
	return -1
}

// Clone gives a deep copy of the request with the context. The context must
// hold a value. The copy of the field Body is shallow.
func (r *Request) Clone(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.ctx = ctx
	r2.URL = cloneURL(r.URL)
	r2.Header = r.Header.Clone()
	r2.Form = cloneURLValues(r.Form)
	r2.PostForm = cloneURLValues(r.PostForm)
	r2.MultipartForm = cloneMultipartForm(r.MultipartForm)
	if s := r.matches; s != nil {
		s2 := make([]string, len(s))
		copy(s2, s)
		r2.matches = s2
	}
	r2.otherValues = maps.Clone(r.otherValues)
	return r2
}

// cloneURL gives a copy of a URL. A nil URL gives nil.
func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	return u.Clone()
}

// cloneURLValues gives a copy of a set of values. A nil set gives nil.
func cloneURLValues(v url.Values) url.Values {
	if v == nil {
		return nil
	}
	return url.Values(Header(v).Clone())
}

// cloneMultipartForm gives a copy of a form. A nil form gives nil.
func cloneMultipartForm(f *multipart.Form) *multipart.Form {
	if f == nil {
		return nil
	}
	f2 := &multipart.Form{
		Value: (map[string][]string)(Header(f.Value).Clone()),
	}
	if f.File != nil {
		m := make(map[string][]*multipart.FileHeader, len(f.File))
		for k, vv := range f.File {
			vv2 := make([]*multipart.FileHeader, len(vv))
			for i, v := range vv {
				vv2[i] = cloneMultipartFileHeader(v)
			}
			m[k] = vv2
		}
		f2.File = m
	}
	return f2
}

// cloneMultipartFileHeader gives a copy of the head of 1 file of a form.
func cloneMultipartFileHeader(fh *multipart.FileHeader) *multipart.FileHeader {
	if fh == nil {
		return nil
	}
	fh2 := new(multipart.FileHeader)
	*fh2 = *fh
	fh2.Header = textproto.MIMEHeader(Header(fh.Header).Clone())
	return fh2
}

// BasicAuth gives the name and the secret of the field Authorization of the
// request, for a request that holds the basic form of RFC 2617.
func (r *Request) BasicAuth() (username, password string, ok bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", "", false
	}
	return parseBasicAuth(auth)
}

// parseBasicAuth reads the basic form of the field Authorization. The text
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" gives ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !ascii.EqualFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}
	username, password, ok = strings.Cut(string(c), ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

// basicAuth gives the value of the field Authorization of the basic form.
func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// SetBasicAuth writes the field Authorization of the basic form. The name and
// the secret reach the wire with no encryption, thus a request of the scheme
// https must carry them.
//
// The name must hold no colon.
func (r *Request) SetBasicAuth(username, password string) {
	r.Header.Set("Authorization", "Basic "+basicAuth(username, password))
}

// ErrNoCookie reports a name of a cookie that the request does not hold.
var ErrNoCookie = errors.New("http: named cookie not present")

// Cookies gives each cookie of the request.
func (r *Request) Cookies() []*Cookie { return readCookies(r.Header, "") }

// CookiesNamed gives each cookie of the name, and it gives an empty list for a
// name that the request does not hold.
func (r *Request) CookiesNamed(name string) []*Cookie {
	if name == "" {
		return []*Cookie{}
	}
	return readCookies(r.Header, name)
}

// Cookie gives the cookie of the name, and it gives ErrNoCookie for a name that
// the request does not hold.
func (r *Request) Cookie(name string) (*Cookie, error) {
	if name == "" {
		return nil, ErrNoCookie
	}
	for _, c := range readCookies(r.Header, name) {
		return c, nil
	}
	return nil, ErrNoCookie
}

// AddCookie adds a cookie to the field Cookie of the request. 1 field holds
// each cookie, and a semicolon divides the pairs.
func (r *Request) AddCookie(c *Cookie) {
	pair := sanitizeCookieName(c.Name) + "=" + sanitizeCookieValue(c.Value, c.Quoted)
	if old := r.Header.Get("Cookie"); old != "" {
		r.Header.Set("Cookie", old+"; "+pair)
		return
	}
	r.Header.Set("Cookie", pair)
}

// MaxBytesReader limits the bytes of a body, exactly as io.LimitReader does. A
// read past the limit gives an error of the type *MaxBytesError, and the Close
// of the result closes the body under it.
//
// It also asks the ResponseWriter to end the connection after the limit.
func MaxBytesReader(w ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser {
	if n < 0 {
		n = 0
	}
	return &maxBytesReader{w: w, r: r, i: n, n: n}
}

// MaxBytesHandler gives a handler that wraps the body of each request with a
// MaxBytesReader of the limit.
func MaxBytesHandler(h Handler, n int64) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		r2 := *r
		r2.Body = MaxBytesReader(w, r.Body, n)
		h.ServeHTTP(w, &r2)
	})
}

// A MaxBytesError reports a body that reached the limit of a MaxBytesReader.
type MaxBytesError struct {
	Limit int64
}

// Error gives the text of the error.
func (e *MaxBytesError) Error() string { return "http: request body too large" }

type maxBytesReader struct {
	w io.Writer
	// r holds the body under the limit.
	r io.ReadCloser
	// i holds the limit, and n holds the bytes that stand open.
	i   int64
	n   int64
	err error
}

func (l *maxBytesReader) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// A read of 1 byte past the count that stands open answers the question of
	// the limit, thus the call reads no more.
	if int64(len(p))-1 > l.n {
		p = p[:l.n+1]
	}
	n, err := l.r.Read(p)
	if int64(n) <= l.n {
		l.n -= int64(n)
		l.err = err
		return n, err
	}
	n = int(l.n)
	l.n = 0
	// The server ends the connection after a body that is too long. A client
	// holds no such answer, thus the test names the interface.
	type requestTooLarger interface {
		requestTooLarge()
	}
	if res, ok := l.w.(requestTooLarger); ok {
		res.requestTooLarge()
	}
	l.err = &MaxBytesError{l.i}
	return n, l.err
}

func (l *maxBytesReader) Close() error { return l.r.Close() }
