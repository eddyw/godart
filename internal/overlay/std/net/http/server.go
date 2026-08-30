//go:build dart

package http

import (
	"bufio"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// A Server holds the settings of a listener.
type Server struct {
	// Addr names the host and the port of the listener.
	Addr string
	// Handler answers each request. A nil handler reads DefaultServeMux.
	Handler Handler

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
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		req, err := readRequest(reader)
		if err != nil {
			return
		}
		w := &response{conn: conn, header: Header{}, req: req}
		handler := s.Handler
		if handler == nil {
			handler = DefaultServeMux
		}
		handler.ServeHTTP(w, req)
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
	conn       net.Conn
	header     Header
	req        *Request
	body       []byte
	status     int
	wroteHead  bool
	closeAfter bool
	flushed    bool
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
