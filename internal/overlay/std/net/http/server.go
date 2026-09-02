//go:build dart

package http

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall/dart"
	"time"
)

// A Handler answers 1 request.
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

// A HandlerFunc is a function that holds the Handler interface.
type HandlerFunc func(ResponseWriter, *Request)

// ServeHTTP calls the function.
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }

// A ResponseWriter writes the answer of a request.
type ResponseWriter interface {
	// Header gives the fields of the answer. A write of a field after
	// WriteHeader reaches no wire.
	Header() Header
	// Write writes the body. The first call writes the status 200 when
	// WriteHeader ran no time.
	Write([]byte) (int, error)
	// WriteHeader writes the status line and the fields.
	WriteHeader(statusCode int)
}

// A Flusher writes the buffer of an answer to the wire.
type Flusher interface {
	Flush()
}

// A Hijacker gives the connection of a request to the handler. The handler
// then owns the connection, and the server writes no answer and no close of
// its own.
//
// The ResponseWriter of the HTTP/1.1 server holds the interface. The answer of
// an HTTP/2 stream carries no connection of its own, thus it holds none.
type Hijacker interface {
	// Hijack gives the connection and a buffered reader and writer of it. The
	// reader can already hold bytes that the client sent.
	Hijack() (net.Conn, *bufio.ReadWriter, error)
}

// ErrHijacked reports a write on an answer whose connection the handler took.
var ErrHijacked = errors.New("http: connection has been hijacked")

// A Server holds the settings of a listener.
type Server struct {
	// Addr names the host and the port of the listener.
	Addr string
	// Handler answers each request. A nil handler reads DefaultServeMux.
	Handler Handler
	// ErrorLog receives the log of an accept or a handshake failure in Go.
	// The overlay server writes no such log, thus the field keeps the value
	// only.
	ErrorLog *log.Logger

	mu       sync.Mutex
	listener net.Listener
	closed   bool
}

// ListenAndServe listens on the address and answers each request.
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// ListenAndServe listens on the address of the server.
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":80"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

// Serve answers each connection of the listener. It gives ErrServerClosed after
// Close.
func (s *Server) Serve(l net.Listener) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		l.Close()
		return ErrServerClosed
	}
	s.listener = l
	s.mu.Unlock()
	for {
		conn, err := l.Accept()
		if err != nil {
			s.mu.Lock()
			done := s.closed
			s.mu.Unlock()
			if done {
				return ErrServerClosed
			}
			return err
		}
		go s.handle(conn)
	}
}

// Close ends the listener. Each connection that stands open keeps its answer.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}

// handle answers each request of 1 connection.
func (s *Server) handle(conn net.Conn) {
	// A TLS connection whose ALPN took h2 leaves the table of the
	// connections, and the HTTP/2 session of the runtime drives its socket.
	if tc, ok := conn.(h2Conn); ok &&
		tc.ConnectionState().NegotiatedProtocol == "h2" {
		s.serveH2(tc.Handle())
		return
	}
	// A handler that hijacks the connection owns it, thus the server closes
	// it no more.
	taken := false
	defer func() {
		if !taken {
			conn.Close()
		}
	}()
	peer := ""
	if addr := conn.RemoteAddr(); addr != nil {
		peer = addr.String()
	}
	reader := bufio.NewReader(conn)
	for {
		req, err := readRequest(reader)
		if err != nil {
			return
		}
		req.RemoteAddr = peer
		w := &response{conn: conn, reader: reader, header: Header{}, req: req}
		handler := s.Handler
		if handler == nil {
			handler = DefaultServeMux
		}
		handler.ServeHTTP(w, req)
		if w.hijacked {
			taken = true
			return
		}
		w.finish()
		if req.Close || w.closeAfter || !drain(req) {
			return
		}
	}
}

// maxPostHandlerReadBytes is the count of bytes of a body that a handler left
// behind that the server reads. A body that holds more ends the connection.
const maxPostHandlerReadBytes = 256 << 10

// drain reads the rest of a body that the handler left. The connection then
// stands at the start of the next request. It gives false for a body that holds
// more than maxPostHandlerReadBytes bytes.
func drain(req *Request) bool {
	if req.Body == nil {
		return true
	}
	n, err := io.Copy(io.Discard, io.LimitReader(req.Body, maxPostHandlerReadBytes+1))
	return err == nil && n <= maxPostHandlerReadBytes
}

// A response writes the answer of 1 request. It holds the body in memory, thus
// the answer names its length.
type response struct {
	conn net.Conn
	// reader holds the buffer of the connection. A Hijack gives it to the
	// handler, because it can already hold bytes of the next request.
	reader     *bufio.Reader
	header     Header
	req        *Request
	body       []byte
	status     int
	wroteHead  bool
	closeAfter bool
	flushed    bool
	// hijacked marks a connection that the handler took.
	hijacked bool
}

func (w *response) Header() Header { return w.header }

func (w *response) WriteHeader(code int) {
	if w.wroteHead {
		return
	}
	w.wroteHead = true
	w.status = code
}

func (w *response) Write(p []byte) (int, error) {
	if w.hijacked {
		return 0, ErrHijacked
	}
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if w.flushed {
		return w.writeChunk(p)
	}
	w.body = append(w.body, p...)
	return len(p), nil
}

// Flush writes the head and the body that stands open, and it moves the answer
// to the chunked form.
func (w *response) Flush() {
	if w.hijacked {
		return
	}
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if !w.flushed {
		w.header.Set("Transfer-Encoding", "chunked")
		w.header.Del("Content-Length")
		w.writeHead()
		w.flushed = true
		body := w.body
		w.body = nil
		if len(body) > 0 {
			w.writeChunk(body)
		}
		return
	}
}

func (w *response) writeChunk(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	head := strconv.FormatInt(int64(len(p)), 16) + "\r\n"
	if _, err := io.WriteString(w.conn, head); err != nil {
		return 0, err
	}
	if _, err := w.conn.Write(p); err != nil {
		return 0, err
	}
	_, err := io.WriteString(w.conn, "\r\n")
	return len(p), err
}

// writeHead writes the status line and each field.
func (w *response) writeHead() {
	text := StatusText(w.status)
	if text == "" {
		text = "status code " + itoa(w.status)
	}
	io.WriteString(w.conn, "HTTP/1.1 "+itoa(w.status)+" "+text+"\r\n")
	// Go reads the first bytes of the body and names the type. The overlay
	// names text only. Read docs/parity.md.
	if w.header.Get("Content-Type") == "" && len(w.body) > 0 {
		w.header.Set("Content-Type", "text/plain; charset=utf-8")
	}
	if w.req != nil && w.req.Close {
		w.header.Set("Connection", "close")
	}
	w.header.Write(w.conn)
	io.WriteString(w.conn, "\r\n")
}

// finish writes the answer that stands open.
func (w *response) finish() {
	if w.hijacked {
		return
	}
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if w.flushed {
		io.WriteString(w.conn, "0\r\n\r\n")
		return
	}
	w.header.Set("Content-Length", itoa(len(w.body)))
	w.writeHead()
	if w.req != nil && w.req.Method == MethodHead {
		return
	}
	if len(w.body) > 0 {
		w.conn.Write(w.body)
	}
}

// Hijack gives the connection of the request to the handler. A second call
// gives ErrHijacked.
func (w *response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.hijacked {
		return nil, nil, ErrHijacked
	}
	if w.flushed {
		return nil, nil, errors.New("http: the answer already reached the wire")
	}
	w.hijacked = true
	reader := w.reader
	if reader == nil {
		reader = bufio.NewReader(w.conn)
	}
	return w.conn, bufio.NewReadWriter(reader, bufio.NewWriter(w.conn)), nil
}

// SetReadDeadline writes the read deadline of the connection.
func (w *response) SetReadDeadline(deadline time.Time) error {
	return w.conn.SetReadDeadline(deadline)
}

// SetWriteDeadline writes the write deadline of the connection.
func (w *response) SetWriteDeadline(deadline time.Time) error {
	return w.conn.SetWriteDeadline(deadline)
}

// EnableFullDuplex lets the handler read the body of the request while it
// writes the answer. The overlay server reads the rest of a body after the
// handler ends, thus each handler already holds the 2 directions, and the call
// changes nothing.
func (w *response) EnableFullDuplex() error { return nil }

// Error writes a status code and a text.
func Error(w ResponseWriter, text string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	io.WriteString(w, text+"\n")
}

// NotFound writes the status 404.
func NotFound(w ResponseWriter, r *Request) { Error(w, "404 page not found", StatusNotFound) }

// NotFoundHandler gives a handler that writes the status 404.
func NotFoundHandler() Handler { return HandlerFunc(NotFound) }

// Redirect writes a redirect to the target.
func Redirect(w ResponseWriter, r *Request, target string, code int) {
	h := w.Header()
	h.Set("Location", target)
	if r.Method == MethodGet {
		h.Set("Content-Type", "text/html; charset=utf-8")
	}
	w.WriteHeader(code)
	if r.Method == MethodGet {
		io.WriteString(w, "<a href=\""+htmlEscape(target)+"\">"+StatusText(code)+"</a>.\n\n")
	}
}

// htmlEscape writes the entity of each byte that HTML holds apart.
func htmlEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&#34;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// A ServeMux holds a pattern for each handler. A pattern that ends with a slash
// names a subtree.
type ServeMux struct {
	mu       sync.Mutex
	patterns map[string]Handler
}

// NewServeMux builds a ServeMux.
func NewServeMux() *ServeMux { return &ServeMux{patterns: map[string]Handler{}} }

// DefaultServeMux is the ServeMux of Handle and HandleFunc.
var DefaultServeMux = NewServeMux()

// Handle registers a handler for the pattern.
func Handle(pattern string, handler Handler) { DefaultServeMux.Handle(pattern, handler) }

// HandleFunc registers a function for the pattern.
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}

// Handle registers a handler for the pattern.
func (m *ServeMux) Handle(pattern string, handler Handler) {
	if pattern == "" {
		panic("http: invalid pattern")
	}
	if handler == nil {
		panic("http: nil handler")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.patterns == nil {
		m.patterns = map[string]Handler{}
	}
	if _, ok := m.patterns[pattern]; ok {
		panic("http: multiple registrations for " + pattern)
	}
	m.patterns[pattern] = handler
}

// HandleFunc registers a function for the pattern.
func (m *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	m.Handle(pattern, HandlerFunc(handler))
}

// Handler gives the handler of a request and the pattern that it matched.
func (m *ServeMux) Handler(r *Request) (Handler, string) {
	path := "/"
	if r.URL != nil {
		path = r.URL.Path
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// The longest pattern wins, exactly as in Go.
	keys := make([]string, 0, len(m.patterns))
	for k := range m.patterns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best := ""
	for _, k := range keys {
		if !matchPattern(k, path) {
			continue
		}
		if len(k) > len(best) {
			best = k
		}
	}
	if best == "" {
		return NotFoundHandler(), ""
	}
	return m.patterns[best], best
}

// ServeHTTP gives the request to the handler of its path.
func (m *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	handler, _ := m.Handler(r)
	handler.ServeHTTP(w, r)
}

// matchPattern reports whether the path matches the pattern. A pattern that
// ends with a slash names a subtree.
func matchPattern(pattern, path string) bool {
	if !strings.HasSuffix(pattern, "/") {
		return pattern == path
	}
	return strings.HasPrefix(path, pattern)
}

// StripPrefix gives a handler that drops the prefix of the path.
func StripPrefix(prefix string, h Handler) Handler {
	if prefix == "" {
		return h
	}
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		rest, ok := strings.CutPrefix(r.URL.Path, prefix)
		if !ok {
			NotFound(w, r)
			return
		}
		out := *r
		u := *r.URL
		u.Path = rest
		out.URL = &u
		h.ServeHTTP(w, &out)
	})
}

// h2Conn names an accepted connection that can carry HTTP/2. The TLS
// listener gives one.
type h2Conn interface {
	ConnectionState() tls.ConnectionState
	Handle() int
}

// serveH2 answers each stream of 1 HTTP/2 connection. Each stream runs on a
// goroutine of its own, exactly as each request of HTTP/2 does in Go.
func (s *Server) serveH2(conn int) {
	session, cause := dart.H2Server(conn)
	if cause != "" {
		return
	}
	defer dart.H2Close(session)
	for {
		stream, method, path, authority, flat, cause := dart.H2Next(session)
		if cause != "" {
			return
		}
		go s.serveH2Stream(stream, method, path, authority, flat)
	}
}

func (s *Server) serveH2Stream(stream int, method, path, authority string, flat []string) {
	u, err := url.ParseRequestURI(path)
	if err != nil {
		u = &url.URL{Path: path}
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
	req := &Request{
		Method:        method,
		URL:           u,
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		Header:        header,
		Body:          &h2Body{stream: stream},
		ContentLength: length,
		Host:          authority,
		RequestURI:    path,
	}
	w := &h2Response{stream: stream, header: Header{}, req: req}
	handler := s.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
	handler.ServeHTTP(w, req)
	w.finish()
}

// An h2Response writes the answer of 1 stream. It holds the body until the
// end of the handler, exactly as the answer of HTTP/1.1 does, and a Flush
// writes the head and the bytes that stand open.
type h2Response struct {
	stream    int
	header    Header
	req       *Request
	body      []byte
	status    int
	wroteHead bool
	headSent  bool
	ended     bool
}

func (w *h2Response) Header() Header { return w.header }

func (w *h2Response) WriteHeader(code int) {
	if w.wroteHead {
		return
	}
	w.wroteHead = true
	w.status = code
}

func (w *h2Response) Write(p []byte) (int, error) {
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if w.headSent {
		if len(p) == 0 {
			return 0, nil
		}
		n, cause := dart.H2Write(w.stream, p)
		if cause != "" {
			return n, errors.New(cause)
		}
		return len(p), nil
	}
	w.body = append(w.body, p...)
	return len(p), nil
}

// Flush writes the head and the body that stands open.
func (w *h2Response) Flush() {
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if !w.headSent {
		w.sendHead(false)
		body := w.body
		w.body = nil
		if len(body) > 0 {
			dart.H2Write(w.stream, body)
		}
	}
}

func (w *h2Response) sendHead(endStream bool) {
	w.headSent = true
	if w.header.Get("Content-Type") == "" && len(w.body) > 0 {
		w.header.Set("Content-Type", "text/plain; charset=utf-8")
	}
	var flat []string
	for name, values := range w.header {
		for _, v := range values {
			flat = append(flat, name, v)
		}
	}
	dart.H2WriteHead(w.stream, w.status, flat, endStream)
	w.ended = endStream
}

// finish writes the answer that stands open, and it ends the stream.
func (w *h2Response) finish() {
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if !w.headSent {
		w.header.Set("Content-Length", itoa(len(w.body)))
		if w.req != nil && w.req.Method == MethodHead {
			w.sendHead(true)
			dart.H2End(w.stream)
			return
		}
		w.sendHead(len(w.body) == 0)
	}
	if w.ended {
		dart.H2End(w.stream)
		return
	}
	if len(w.body) > 0 {
		dart.H2Write(w.stream, w.body)
	}
	dart.H2End(w.stream)
}
