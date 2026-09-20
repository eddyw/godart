//go:build dart

package http

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"net/http/internal/ascii"
	"net/url"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

// A CloseNotifier gives a channel that reports a client that left before the
// answer ended.
//
// No ResponseWriter of this overlay holds the interface, thus an assertion of
// it gives false. A handler must read that result, exactly as Go asks. The
// runtime reports a close of the peer at the next read alone, and the writer
// holds no read of its own.
//
// Deprecated: Read the context of the request. It ends when the client
// leaves.
type CloseNotifier interface {
	CloseNotify() <-chan bool
}

// A Pusher sends a push of HTTP/2 to the client.
//
// No ResponseWriter of this overlay holds the interface, thus an assertion of
// it gives false, and a handler then serves the resource on request. The
// HTTP/2 session of the runtime sends no push frame.
type Pusher interface {
	// Push starts a push of the target. A target that holds no scheme and no
	// host names a path of the same origin.
	Push(target string, opts *PushOptions) error
}

// A PushOptions holds the fields of a push.
type PushOptions struct {
	// Method names the method of the push. An empty value names GET, and GET
	// and HEAD are the 2 that a push allows.
	Method string

	// Header holds the fields of the request of the push. A field that names
	// the method, the scheme, the authority, or the path is not allowed.
	Header Header
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

// A ConnState names the state of a connection of a server. The hook
// Server.ConnState reads each change.
type ConnState int

const (
	// StateNew marks a connection that a client opened. Such a connection
	// moves to StateActive or to StateClosed.
	StateNew ConnState = iota
	// StateActive marks a connection that holds the bytes of a request. The
	// hook runs before the server reads the request.
	StateActive
	// StateIdle marks a connection that answered a request and that waits for
	// the next request of the same client.
	StateIdle
	// StateHijacked marks a connection that a handler took. It is a final
	// state, and no StateClosed follows it.
	StateHijacked
	// StateClosed marks a connection that ended. It is a final state.
	StateClosed
)

var stateName = map[ConnState]string{
	StateNew:      "new",
	StateActive:   "active",
	StateIdle:     "idle",
	StateHijacked: "hijacked",
	StateClosed:   "closed",
}

// String gives the name of the state.
func (c ConnState) String() string { return stateName[c] }

// A contextKey names a value that the server puts in the context of a request.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "net/http context value " + k.name }

// ServerContextKey names the *Server of the context of a request.
var ServerContextKey = &contextKey{"http-server"}

// LocalAddrContextKey names the local address of the connection of a request.
// The value holds the type net.Addr.
var LocalAddrContextKey = &contextKey{"local-addr"}

// DefaultMaxHeaderBytes is the limit of the head of a request. A Server with no
// MaxHeaderBytes takes this value.
const DefaultMaxHeaderBytes = 1 << 20 // 1 MB

// DefaultMaxHeaderValueCount is the count of the values of the head of a
// request that a server reads. A Server with no MaxHeaderValueCount takes this
// value.
const DefaultMaxHeaderValueCount = 500

// A Server holds the settings of a listener.
type Server struct {
	// Addr names the host and the port of the listener.
	Addr string
	// Handler answers each request. A nil handler reads DefaultServeMux.
	Handler Handler
	// DisableGeneralOptionsHandler stops the built in answer of the request
	// line "OPTIONS *". The handler of the server then answers such a request.
	DisableGeneralOptionsHandler bool
	// ReadTimeout is the whole time of the read of a request, from the accept
	// to the end of the body. A value of 0 holds no limit.
	ReadTimeout time.Duration
	// ReadHeaderTimeout is the time of the read of the head of a request. A
	// value of 0 reads ReadTimeout.
	ReadHeaderTimeout time.Duration
	// WriteTimeout is the time of the write of an answer, from the end of the
	// head of the request. A value of 0 holds no limit.
	WriteTimeout time.Duration
	// IdleTimeout is the time that a connection waits for the next request. A
	// value of 0 reads ReadTimeout.
	IdleTimeout time.Duration
	// MaxHeaderBytes limits the bytes of the request line and of the head of a
	// request. A value of 0 reads DefaultMaxHeaderBytes. The body stays
	// outside the count.
	MaxHeaderBytes int
	// MaxHeaderValueCount limits the count of the values of the head of a
	// request. A value of 0 reads DefaultMaxHeaderValueCount. 2 values that a
	// comma divides in 1 line count 1 time, and 2 lines of 1 name count 2
	// times.
	MaxHeaderValueCount int
	// ConnState runs at each change of the state of a connection.
	ConnState func(net.Conn, ConnState)
	// ErrorLog receives the log of an accept failure and of a panic of a
	// handler. A nil field writes to the logger of the package log.
	ErrorLog *log.Logger
	// BaseContext gives the context of each connection of 1 listener. A nil
	// function reads context.Background.
	BaseContext func(net.Listener) context.Context
	// ConnContext changes the context of 1 connection. The result must hold no
	// nil value.
	ConnContext func(ctx context.Context, c net.Conn) context.Context
	// TLSConfig holds the settings of the TLS listener that ListenAndServeTLS
	// opens. ListenAndServeTLS reads the certificate of the 2 file names of
	// its own, or the field Certificates of this config.
	//
	// Read the package crypto/tls for each field that Dart honors. A field
	// that would change the handshake stops the program at the listen.
	TLSConfig *tls.Config
	// TLSNextProto names 1 handler for each protocol of ALPN outside
	// http/1.1 and h2. The server calls the handler of the protocol that the
	// handshake took, and the handler then owns the connection.
	//
	// A nil map leaves each protocol to the server.
	TLSNextProto map[string]func(*Server, *tls.Conn, Handler)
	// Protocols names the set of protocols that the server accepts. A nil
	// field holds HTTP1 and HTTP2.
	//
	// The server carries HTTP/2 over TLS alone. A set that holds
	// UnencryptedHTTP2 gives an error at the listen, because no path of the
	// runtime carries HTTP/2 over a plain socket.
	Protocols *Protocols

	// down marks a server that Close or Shutdown ended.
	down atomic.Bool

	mu          sync.Mutex
	listeners   map[*net.Listener]struct{}
	conns       map[*serverConn]struct{}
	onShutdown  []func()
	noKeepAlive bool
}

// ListenAndServe listens on the address and answers each request.
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// Serve answers each connection of the listener with the handler. A nil handler
// reads DefaultServeMux.
func Serve(l net.Listener, handler Handler) error {
	server := &Server{Handler: handler}
	return server.Serve(l)
}

// ListenAndServe listens on the address of the server.
func (s *Server) ListenAndServe() error {
	if s.shuttingDown() {
		return ErrServerClosed
	}
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
// Close or after Shutdown.
func (s *Server) Serve(l net.Listener) error {
	if err := s.checkProtocols(); err != nil {
		return err
	}
	origin := l
	var tracked net.Listener = &onceCloseListener{Listener: l}
	defer tracked.Close()
	if !s.trackListener(&tracked, true) {
		return ErrServerClosed
	}
	defer s.trackListener(&tracked, false)

	base := context.Background()
	if s.BaseContext != nil {
		base = s.BaseContext(origin)
		if base == nil {
			panic("BaseContext returned a nil context")
		}
	}
	base = context.WithValue(base, ServerContextKey, s)
	for {
		conn, err := tracked.Accept()
		if err != nil {
			if s.shuttingDown() {
				return ErrServerClosed
			}
			return err
		}
		ctx := base
		if hook := s.ConnContext; hook != nil {
			ctx = hook(ctx, conn)
			if ctx == nil {
				panic("ConnContext returned a nil context")
			}
		}
		c := &serverConn{server: s, conn: conn}
		s.trackConn(c, true)
		s.setState(c, StateNew)
		go c.serve(ctx)
	}
}

// Close ends each listener and each connection of the server. A handler that
// runs loses its connection. Read Shutdown for the end that waits.
func (s *Server) Close() error {
	s.down.Store(true)
	s.mu.Lock()
	err := s.closeListenersLocked()
	for c := range s.conns {
		c.conn.Close()
		delete(s.conns, c)
	}
	s.mu.Unlock()
	return err
}

// shutdownPollInterval is the wait between 2 tests of Shutdown.
const shutdownPollInterval = 10 * time.Millisecond

// Shutdown ends each listener, and it then waits for each connection to reach
// the idle state. It gives the error of the context when the context ends
// first, and Serve then gives ErrServerClosed.
func (s *Server) Shutdown(ctx context.Context) error {
	s.down.Store(true)
	s.mu.Lock()
	err := s.closeListenersLocked()
	for _, f := range s.onShutdown {
		go f()
	}
	s.mu.Unlock()
	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if s.closeIdleConns() {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// RegisterOnShutdown adds a function that Shutdown calls on a goroutine of its
// own. A server of a web socket registers the end of each such connection here.
func (s *Server) RegisterOnShutdown(f func()) {
	s.mu.Lock()
	s.onShutdown = append(s.onShutdown, f)
	s.mu.Unlock()
}

// SetKeepAlivesEnabled holds or drops the connection after an answer. A false
// value also ends each connection that stands idle.
func (s *Server) SetKeepAlivesEnabled(v bool) {
	s.mu.Lock()
	s.noKeepAlive = !v
	s.mu.Unlock()
	if v {
		return
	}
	s.closeIdleConns()
}

// closeIdleConns ends each connection that stands idle, and it reports whether
// the server holds no connection any more.
func (s *Server) closeIdleConns() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	quiet := true
	now := time.Now().Unix()
	for c := range s.conns {
		state, stamp := c.getState()
		// A connection that stood new for more than 5 seconds sent no request,
		// thus it counts as idle. Go holds the same rule.
		if state == StateNew && stamp < now-5 {
			state = StateIdle
		}
		if state != StateIdle {
			quiet = false
			continue
		}
		c.conn.Close()
		delete(s.conns, c)
	}
	return quiet
}

func (s *Server) closeListenersLocked() error {
	var err error
	for l := range s.listeners {
		if cause := (*l).Close(); cause != nil && err == nil {
			err = cause
		}
	}
	return err
}

// trackListener adds a listener to the server, or it drops one. It gives false
// for an add on a server that already ended.
func (s *Server) trackListener(l *net.Listener, add bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listeners == nil {
		s.listeners = map[*net.Listener]struct{}{}
	}
	if !add {
		delete(s.listeners, l)
		return true
	}
	if s.shuttingDown() {
		return false
	}
	s.listeners[l] = struct{}{}
	return true
}

func (s *Server) trackConn(c *serverConn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conns == nil {
		s.conns = map[*serverConn]struct{}{}
	}
	if add {
		s.conns[c] = struct{}{}
		return
	}
	delete(s.conns, c)
}

// setState writes the state of a connection and it calls the hook. A connection
// that a handler took, and a connection that ended, each leave the table of the
// server, thus Shutdown waits for neither one.
func (s *Server) setState(c *serverConn, state ConnState) {
	c.mu.Lock()
	c.state = state
	c.stamp = time.Now().Unix()
	c.mu.Unlock()
	if state == StateHijacked || state == StateClosed {
		s.trackConn(c, false)
	}
	if hook := s.ConnState; hook != nil {
		hook(c.conn, state)
	}
}

func (s *Server) shuttingDown() bool { return s.down.Load() }

func (s *Server) doKeepAlives() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.noKeepAlive && !s.shuttingDown()
}

func (s *Server) idleTimeout() time.Duration {
	if s.IdleTimeout != 0 {
		return s.IdleTimeout
	}
	return s.ReadTimeout
}

func (s *Server) readHeaderTimeout() time.Duration {
	if s.ReadHeaderTimeout != 0 {
		return s.ReadHeaderTimeout
	}
	return s.ReadTimeout
}

func (s *Server) maxHeaderBytes() int {
	if s.MaxHeaderBytes > 0 {
		return s.MaxHeaderBytes
	}
	return DefaultMaxHeaderBytes
}

// readLimit gives the bytes that the request line and the head of a request may
// hold together. Go adds 4096 bytes to MaxHeaderBytes for the buffer, thus the
// overlay adds the same count.
func (s *Server) readLimit() int { return s.maxHeaderBytes() + 4096 }

func (s *Server) maxHeaderValueCount() int {
	if s.MaxHeaderValueCount > 0 {
		return s.MaxHeaderValueCount
	}
	return DefaultMaxHeaderValueCount
}

// logf writes a line to ErrorLog, or to the logger of the package log.
// protocols gives the set of protocols of the server. A nil field holds
// HTTP1 and HTTP2, exactly as in Go.
func (s *Server) protocols() Protocols {
	if s.Protocols != nil {
		return *s.Protocols
	}
	var p Protocols
	p.SetHTTP1(true)
	p.SetHTTP2(true)
	return p
}

// checkProtocols rejects a set of protocols that the server cannot honor.
func (s *Server) checkProtocols() error {
	if s.Protocols == nil {
		return nil
	}
	p := *s.Protocols
	if p.UnencryptedHTTP2() {
		return errors.New("http: the server carries HTTP/2 over TLS alone, " +
			"thus Protocols must hold no UnencryptedHTTP2")
	}
	if !p.HTTP1() && !p.HTTP2() {
		return errors.New("http: Protocols holds no protocol that the server carries")
	}
	return nil
}

// alpn gives the list of ALPN of a TLS listener of the server. The list holds
// the protocol of each handler of TLSNextProto first, exactly as in Go.
func (s *Server) alpn() []string {
	var out []string
	for proto := range s.TLSNextProto {
		if proto == "http/1.1" || proto == "h2" {
			continue
		}
		out = append(out, proto)
	}
	slices.Sort(out)
	p := s.protocols()
	if p.HTTP2() {
		out = append(out, "h2")
	}
	if p.HTTP1() {
		out = append(out, "http/1.1")
	}
	return out
}

func (s *Server) logf(format string, args ...any) {
	if s != nil && s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}

// An onceCloseListener ends its listener 1 time only. Serve and Close each
// reach the same listener.
type onceCloseListener struct {
	net.Listener
	once sync.Once
	err  error
}

func (l *onceCloseListener) Close() error {
	l.once.Do(func() { l.err = l.Listener.Close() })
	return l.err
}

// A serverConn holds 1 connection of a server. The server tracks it, thus
// Close and Shutdown each reach it.
type serverConn struct {
	server *Server
	conn   net.Conn

	mu sync.Mutex
	// state holds the state of the connection, and stamp holds the second of
	// its last change.
	state ConnState
	stamp int64
}

func (c *serverConn) getState() (ConnState, int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.stamp
}

// serve answers each request of 1 connection.
func (c *serverConn) serve(ctx context.Context) {
	s := c.server
	// A handler that hijacks the connection owns it, thus the server closes it
	// no more.
	taken := false
	defer func() {
		if taken {
			return
		}
		c.conn.Close()
		s.setState(c, StateClosed)
	}()
	// A TLS connection whose ALPN took h2 leaves the loop of HTTP/1.1, and the
	// HTTP/2 session of the runtime drives its socket. A protocol outside
	// http/1.1 and h2 reaches the handler of TLSNextProto.
	proto := ""
	if tc, ok := c.conn.(h2Conn); ok {
		proto = tc.ConnectionState().NegotiatedProtocol
	}
	if proto != "" && proto != "http/1.1" && proto != "h2" {
		if next, ok := s.TLSNextProto[proto]; ok {
			tc, isTLS := c.conn.(*tls.Conn)
			if !isTLS {
				s.logf("http: the connection of the protocol %s holds no *tls.Conn", proto)
				return
			}
			s.setState(c, StateActive)
			taken = true
			next(s, tc, serverHandler{s})
			return
		}
	}
	if proto == "h2" {
		if !s.protocols().HTTP2() {
			s.logf("http: the server takes no connection of h2, because Protocols holds no HTTP2")
			return
		}
		tc, _ := c.conn.(h2Conn)
		s.setState(c, StateActive)
		s.serveH2(ctx, tc.Handle())
		return
	}
	if !s.protocols().HTTP1() {
		s.logf("http: the server takes no connection of HTTP/1, because Protocols holds no HTTP1")
		return
	}
	ctx = context.WithValue(ctx, LocalAddrContextKey, c.conn.LocalAddr())
	peer := ""
	if addr := c.conn.RemoteAddr(); addr != nil {
		peer = addr.String()
	}
	reader := bufio.NewReader(c.conn)
	first := true
	for {
		if !first {
			s.setState(c, StateIdle)
			if d := s.idleTimeout(); d > 0 {
				if c.conn.SetReadDeadline(time.Now().Add(d)) != nil {
					return
				}
			}
		}
		// The state moves to active at the first byte of the request, thus the
		// peek holds the wait for the client.
		if _, err := reader.Peek(1); err != nil {
			return
		}
		s.setState(c, StateActive)
		first = false
		start := time.Now()
		if d := s.readHeaderTimeout(); d > 0 {
			c.conn.SetReadDeadline(start.Add(d))
		} else {
			c.conn.SetReadDeadline(time.Time{})
		}
		if d := s.WriteTimeout; d > 0 {
			c.conn.SetWriteDeadline(start.Add(d))
		} else {
			c.conn.SetWriteDeadline(time.Time{})
		}
		req, err := readRequestLimit(reader, s.readLimit(), s.maxHeaderValueCount())
		if err != nil {
			if errors.Is(err, errHeaderTooLarge) {
				io.WriteString(c.conn, "HTTP/1.1 431 Request Header Fields Too Large\r\n"+
					"Content-Length: 0\r\nConnection: close\r\n\r\n")
			}
			return
		}
		if d := s.ReadTimeout; d > 0 {
			c.conn.SetReadDeadline(start.Add(d))
		}
		req.RemoteAddr = peer
		// A request that declares a trailer gives the status 501. The overlay
		// reads no trailer, thus a silent drop would lose a field.
		if req.Header.Get("Trailer") != "" {
			io.WriteString(c.conn, "HTTP/1.1 501 Not Implemented\r\n"+
				"Content-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}
		if tc, ok := c.conn.(h2Conn); ok {
			state := tc.ConnectionState()
			req.TLS = &state
		}
		reqCtx, cancel := context.WithCancel(ctx)
		req.ctx = reqCtx
		w := &response{conn: c.conn, reader: reader, header: Header{}, req: req}
		// A request that carries the field Expect with the value 100-continue
		// holds its body until the server approves it.
		if req.Body != nil && req.ProtoAtLeast(1, 1) &&
			ascii.EqualFold(req.Header.Get("Expect"), "100-continue") {
			req.Body = &continueBody{body: req.Body, res: w}
		}
		if !s.doKeepAlives() || req.Close {
			w.closeAfter = true
		}
		serverHandler{s}.ServeHTTP(w, req)
		if w.hijacked {
			cancel()
			taken = true
			s.setState(c, StateHijacked)
			return
		}
		w.finish()
		cancel()
		if w.closeAfter || !drain(req) {
			return
		}
	}
}

// A serverHandler picks the handler of a request. A nil Handler of the server
// reads DefaultServeMux, and the request line "OPTIONS *" reads a handler of
// the package.
type serverHandler struct {
	srv *Server
}

func (sh serverHandler) ServeHTTP(w ResponseWriter, req *Request) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
	if !sh.srv.DisableGeneralOptionsHandler && req.RequestURI == "*" &&
		req.Method == MethodOptions {
		handler = globalOptionsHandler{}
	}
	handler.ServeHTTP(w, req)
}

// A globalOptionsHandler answers the request line "OPTIONS *".
type globalOptionsHandler struct{}

func (globalOptionsHandler) ServeHTTP(w ResponseWriter, r *Request) {
	w.Header().Set("Content-Length", "0")
	if r.ContentLength != 0 {
		// A body of such a request holds no meaning. The read of 4 kilobytes
		// keeps the connection, and a longer body ends it.
		mb := MaxBytesReader(w, r.Body, 4<<10)
		io.Copy(io.Discard, mb)
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

// A continueBody answers the field Expect of a request. The first read of the
// body writes the line 100 Continue, thus the client sends the body.
//
// A handler that writes the answer with no read of the body sends no such line,
// exactly as in Go. The client then waits for its own ExpectContinueTimeout.
type continueBody struct {
	body io.ReadCloser
	res  *response
	done bool
}

func (b *continueBody) Read(p []byte) (int, error) {
	if !b.done {
		b.done = true
		if !b.res.wroteHead {
			if _, err := io.WriteString(b.res.conn, "HTTP/1.1 100 Continue\r\n\r\n"); err != nil {
				return 0, err
			}
		}
	}
	return b.body.Read(p)
}

func (b *continueBody) Close() error {
	b.done = true
	return b.body.Close()
}

// A response buffers a small body before it selects the wire framing.
type response struct {
	conn net.Conn
	// reader holds the buffer of the connection. A Hijack gives it to the
	// handler, because it can already hold bytes of the next request.
	reader        *bufio.Reader
	header        Header
	req           *Request
	body          []byte
	status        int
	wroteHead     bool
	closeAfter    bool
	flushed       bool
	writeErr      error
	contentLength int64
	written       int64
	// chunked marks an answer of the chunked form. Only HTTP/1.1 holds it.
	chunked bool
	// hijacked marks a connection that the handler took.
	hijacked bool
	// trailers names each field of the head Trailer that the answer writes
	// after the body.
	trailers []string
	// head holds the copy of the fields that WriteHeader took. An answer that
	// declares no trailer holds nil, and the write then reads the live map.
	head Header
}

// TrailerPrefix names a field that the server writes after the body. A
// handler adds the prefix in front of the name of the field, and the server
// removes the prefix at the write.
//
// The prefix reaches a field that the handler did not name in the head
// Trailer. The chunked form of HTTP/1.1 carries such a field, thus an answer
// of HTTP/1.0 writes none, exactly as in Go.
const TrailerPrefix = "Trailer:"

func (w *response) Header() Header { return w.header }

// requestTooLarge ends the connection after the answer. A MaxBytesReader of the
// body of the request calls it.
func (w *response) requestTooLarge() { w.closeAfter = true }

func (w *response) WriteHeader(code int) {
	if w.wroteHead {
		return
	}
	w.wroteHead = true
	w.status = code
	w.contentLength = -1
	if cl := w.header.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil && n >= 0 {
			w.contentLength = n
		} else {
			w.header.Del("Content-Length")
		}
	}
	w.declareTrailers()
}

// declareTrailers reads the head Trailer and keeps a copy of the fields.
//
// A handler writes the value of a declared field after WriteHeader, thus the
// copy holds the fields that the head of the answer takes. An answer that
// declares no trailer keeps no copy.
func (w *response) declareTrailers() {
	var names []string
	for _, v := range w.header.Values("Trailer") {
		for _, part := range strings.Split(v, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			key := CanonicalHeaderKey(name)
			switch key {
			case "Transfer-Encoding", "Trailer", "Content-Length":
				// RFC 9110, 6.5.1 forbids each of the 3 after the body.
				continue
			}
			names = append(names, key)
		}
	}
	if len(names) == 0 {
		return
	}
	w.trailers = names
	w.head = w.header.Clone()
}

// wire gives the fields that the head of the answer takes.
func (w *response) wire() Header {
	if w.head != nil {
		return w.head
	}
	return w.header
}

// finalTrailers gives each field that follows the body. A key with the prefix
// TrailerPrefix loses the prefix, and a name of the head Trailer keeps its
// value of the live map.
func (w *response) finalTrailers() Header {
	var t Header
	for k, vv := range w.header {
		name, found := strings.CutPrefix(k, TrailerPrefix)
		if !found {
			continue
		}
		if t == nil {
			t = Header{}
		}
		t[name] = vv
	}
	for _, k := range w.trailers {
		for _, v := range w.header.Values(k) {
			if t == nil {
				t = Header{}
			}
			t.Add(k, v)
		}
	}
	return t
}

// writeTrailers ends the chunked body. It writes each field that follows the
// body, and then the empty line.
func (w *response) writeTrailers(t Header) {
	io.WriteString(w.conn, "0\r\n")
	if len(t) > 0 {
		t.Write(w.conn)
	}
	io.WriteString(w.conn, "\r\n")
}

func (w *response) Write(p []byte) (int, error) {
	if w.hijacked {
		return 0, ErrHijacked
	}
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if len(p) == 0 {
		return 0, nil
	}
	w.written += int64(len(p))
	if w.contentLength != -1 && w.written > w.contentLength {
		return 0, ErrContentLength
	}
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	if w.flushed {
		return w.writeChunk(p)
	}
	if len(w.body)+len(p) > 2048 && (w.req == nil || w.req.Method != MethodHead) {
		n := 2048 - len(w.body)
		w.body = append(w.body, p[:n]...)
		w.Flush()
		if w.writeErr != nil {
			return n, w.writeErr
		}
		written, err := w.writeChunk(p[n:])
		return n + written, err
	}
	w.body = append(w.body, p...)
	return len(p), nil
}

// Flush sends buffered data and selects framing for subsequent writes.
func (w *response) Flush() {
	if w.hijacked {
		return
	}
	if !w.wroteHead {
		w.WriteHeader(StatusOK)
	}
	if !w.flushed {
		// The chunked form belongs to HTTP/1.1. An answer of HTTP/1.0 that
		// flushes writes the bytes raw, and the end of the connection then
		// names the end of the body.
		head := w.wire()
		w.chunked = head.Get("Content-Length") == "" && (w.req == nil || w.req.ProtoAtLeast(1, 1))
		if w.chunked {
			head.Set("Transfer-Encoding", "chunked")
		} else if head.Get("Content-Length") == "" {
			w.closeAfter = true
		}
		w.writeHead()
		w.flushed = true
		body := w.body
		w.body = nil
		if len(body) > 0 {
			_, w.writeErr = w.writeChunk(body)
		}
		return
	}
}

func (w *response) writeChunk(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if !w.chunked {
		return w.conn.Write(p)
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
	head := w.wire()
	if _, exists := head["Content-Type"]; !exists && head.Get("Content-Encoding") == "" && len(w.body) > 0 {
		head.Set("Content-Type", DetectContentType(w.body))
	}
	if w.closeAfter {
		head.Set("Connection", "close")
	} else if w.req != nil && !w.req.ProtoAtLeast(1, 1) {
		head.Set("Connection", "keep-alive")
	}
	// A field with the prefix TrailerPrefix follows the body, thus the head
	// holds none of them.
	var skip map[string]bool
	for k := range head {
		if strings.HasPrefix(k, TrailerPrefix) {
			if skip == nil {
				skip = map[string]bool{}
			}
			skip[k] = true
		}
	}
	head.WriteSubset(w.conn, skip)
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
	if (w.req == nil || w.req.Method != MethodHead) && w.contentLength != -1 && w.written != w.contentLength {
		w.closeAfter = true
	}
	trailers := w.finalTrailers()
	if w.flushed {
		if w.chunked {
			w.writeTrailers(trailers)
		}
		return
	}
	// A field that follows the body needs the chunked form of HTTP/1.1.
	if len(trailers) > 0 && (w.req == nil || w.req.ProtoAtLeast(1, 1)) {
		w.chunked = true
		head := w.wire()
		head.Set("Transfer-Encoding", "chunked")
		head.Del("Content-Length")
		w.writeHead()
		if w.req == nil || w.req.Method != MethodHead {
			w.writeChunk(w.body)
		}
		w.writeTrailers(trailers)
		return
	}
	if w.wire().Get("Content-Length") == "" {
		w.wire().Set("Content-Length", itoa(len(w.body)))
	}
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

// A redirectHandler answers each request with a redirect.
type redirectHandler struct {
	url  string
	code int
}

func (rh *redirectHandler) ServeHTTP(w ResponseWriter, r *Request) {
	Redirect(w, r, rh.url, rh.code)
}

// RedirectHandler gives a handler that answers each request with a redirect to
// the URL and with the status code. The code must name a redirect, for example
// StatusMovedPermanently or StatusFound.
func RedirectHandler(url string, code int) Handler {
	return &redirectHandler{url, code}
}

// AllowQuerySemicolons gives a handler that reads each semicolon of the raw
// query as an ampersand, and that then calls the handler.
//
// Go stopped the semicolon as a separator of a query in 1.17. This wrapper
// keeps the old form for a client that still writes it.
func AllowQuerySemicolons(h Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		if !strings.Contains(r.URL.RawQuery, ";") {
			h.ServeHTTP(w, r)
			return
		}
		r2 := new(Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.RawQuery = strings.ReplaceAll(r.URL.RawQuery, ";", "&")
		h.ServeHTTP(w, r2)
	})
}

// ErrHandlerTimeout is the error of a write on the ResponseWriter of a
// TimeoutHandler after the time ran out.
var ErrHandlerTimeout = errors.New("http: Handler timeout")

// TimeoutHandler gives a handler that runs the handler under a time limit.
//
// The new handler calls the handler each time, and it gives the answer of that
// handler when the handler ends inside the limit. A handler that takes longer
// gives the status 503, the text of the message, and the type text/html. Each
// later write of the handler gives ErrHandlerTimeout.
//
// The new handler holds no Flusher and no Hijacker, exactly as in Go.
func TimeoutHandler(h Handler, dt time.Duration, msg string) Handler {
	return &timeoutHandler{handler: h, body: msg, dt: dt}
}

type timeoutHandler struct {
	handler Handler
	body    string
	dt      time.Duration
}

func (h *timeoutHandler) errorBody() string {
	if h.body != "" {
		return h.body
	}
	return "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"
}

func (h *timeoutHandler) ServeHTTP(w ResponseWriter, r *Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.dt)
	defer cancel()
	r = r.WithContext(ctx)
	done := make(chan struct{})
	panics := make(chan any, 1)
	tw := &timeoutWriter{w: w, h: Header{}, req: r}
	go func() {
		defer func() {
			if p := recover(); p != nil {
				panics <- p
			}
		}()
		h.handler.ServeHTTP(tw, r)
		close(done)
	}()
	select {
	case p := <-panics:
		panic(p)
	case <-done:
		tw.mu.Lock()
		defer tw.mu.Unlock()
		dst := w.Header()
		for k, v := range tw.h {
			dst[k] = v
		}
		if !tw.wroteHeader {
			tw.code = StatusOK
		}
		w.WriteHeader(tw.code)
		w.Write(tw.body)
	case <-ctx.Done():
		tw.mu.Lock()
		defer tw.mu.Unlock()
		w.WriteHeader(StatusServiceUnavailable)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			io.WriteString(w, h.errorBody())
			tw.err = ErrHandlerTimeout
			return
		}
		tw.err = ctx.Err()
	}
}

// A timeoutWriter holds the answer of a handler that runs under a time limit.
// The bytes reach the real answer only when the handler ends inside the limit.
type timeoutWriter struct {
	w   ResponseWriter
	h   Header
	req *Request

	mu          sync.Mutex
	body        []byte
	code        int
	wroteHeader bool
	// err holds the error of each write after the time ran out.
	err error
}

func (tw *timeoutWriter) Header() Header { return tw.h }

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.err != nil {
		return 0, tw.err
	}
	if !tw.wroteHeader {
		tw.writeHeaderLocked(StatusOK)
	}
	tw.body = append(tw.body, p...)
	return len(p), nil
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.writeHeaderLocked(code)
}

func (tw *timeoutWriter) writeHeaderLocked(code int) {
	if code < 100 || code > 999 {
		panic("invalid WriteHeader code " + itoa(code))
	}
	if tw.err != nil || tw.wroteHeader {
		return
	}
	tw.wroteHeader = true
	tw.code = code
}

// ServeMux is an HTTP request multiplexer. It matches the URL of each incoming
// request against a list of registered patterns, and it calls the handler of
// the pattern that fits the URL best.
//
// A pattern holds an optional method, an optional host, and a path.
//
//	[METHOD ][HOST]/[PATH]
//
// A pattern with a method matches that method alone, and the method GET also
// matches HEAD. A pattern with no method matches each method.
//
// A path holds segments that a slash divides. A segment that stands between
// braces is a wildcard: "{name}" matches 1 segment, and "{name...}" at the end
// matches each segment that follows. The segment "{$}" at the end matches the
// end of the path alone. Request.PathValue gives the text of a wildcard.
//
// A path that ends with a slash matches each path under it. A request of the
// path of such a pattern with no slash takes a redirect to the path with the
// slash.
//
// The pattern that is more specific wins. 2 patterns where neither is more
// specific than the other conflict, and the registration of the second one
// stops the program.
//
// This type comes from the real source of Go 1.27. The mode of Go 1.21, which
// GODEBUG=httpmuxgo121=1 names, stays out.
type ServeMux struct {
	mu    sync.RWMutex
	tree  routingNode
	index routingIndex
}

// NewServeMux builds a ServeMux.
func NewServeMux() *ServeMux { return &ServeMux{} }

// DefaultServeMux is the ServeMux of Handle and HandleFunc.
var DefaultServeMux = &defaultServeMux

var defaultServeMux ServeMux

// cleanPath gives the canonical form of a path. It drops each element "." and
// each element "..".
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean drops the trailing slash of each path but the root, thus the
	// slash comes back here.
	if p[len(p)-1] == '/' && np != "/" {
		if len(p) == len(np)+1 && strings.HasPrefix(p, np) {
			np = p
		} else {
			np += "/"
		}
	}
	return np
}

// stripHostPort gives the host with no trailing port.
func stripHostPort(h string) string {
	if !strings.Contains(h, ":") {
		return h
	}
	host, _, err := net.SplitHostPort(h)
	if err != nil {
		return h
	}
	return host
}

// Handler gives the handler of a request and the pattern that matched it. It
// reads r.Method, r.Host, and r.URL.Path, and it always gives a handler.
//
// A path that is not canonical gives a handler of a redirect and the path that
// matches after the redirect. Handler changes no field of the request, thus
// Request.PathValue gives an empty string after this call.
func (mux *ServeMux) Handler(r *Request) (Handler, string) {
	h, p, _, _ := mux.findHandler(r)
	return h, p
}

// findHandler gives the handler of a request, the text of the pattern, the
// pattern, and the value of each wildcard.
func (mux *ServeMux) findHandler(r *Request) (h Handler, patStr string, _ *pattern, matches []string) {
	var n *routingNode
	host := r.URL.Host
	escapedPath := r.URL.EscapedPath()
	target := escapedPath
	// A request of the method CONNECT keeps its path.
	if r.Method == MethodConnect {
		_, _, u := mux.matchOrRedirect(host, r.Method, target, r.URL)
		if u != nil {
			return RedirectHandler(u.String(), StatusTemporaryRedirect), u.Path, nil, nil
		}
		// The match runs a second time with r.Host. A nil URL skips the
		// redirect of the trailing slash.
		n, matches, _ = mux.matchOrRedirect(r.Host, r.Method, target, nil)
	} else {
		host = stripHostPort(r.Host)
		target = cleanPath(target)

		var u *url.URL
		n, matches, u = mux.matchOrRedirect(host, r.Method, target, r.URL)
		if u != nil {
			return RedirectHandler(u.String(), StatusTemporaryRedirect), n.pattern.String(), nil, nil
		}
		if target != escapedPath {
			// The path took a clean, thus the answer names the new path.
			patStr := ""
			if n != nil {
				patStr = n.pattern.String()
			}
			u := urlFromEscaped(target, r.URL.RawQuery)
			return RedirectHandler(u.String(), StatusTemporaryRedirect), patStr, nil, nil
		}
	}
	if n == nil {
		// No pattern of the method matched. A pattern that matches the path
		// with another method gives the status 405 and the field Allow.
		allowed := mux.matchingMethods(host, target)
		if len(allowed) > 0 {
			return HandlerFunc(func(w ResponseWriter, r *Request) {
				w.Header().Set("Allow", strings.Join(allowed, ", "))
				Error(w, StatusText(StatusMethodNotAllowed), StatusMethodNotAllowed)
			}), "", nil, nil
		}
		return NotFoundHandler(), "", nil, nil
	}
	return n.handler, n.pattern.String(), n.pattern, matches
}

// matchOrRedirect reads the node of the tree that matches the host, the method,
// and the path.
//
// A URL that holds a value also gives the redirect of the trailing slash: a
// path that matches no pattern takes a second match with a slash at its end,
// and the third result then names the target of the redirect.
func (mux *ServeMux) matchOrRedirect(host, method, target string, u *url.URL) (_ *routingNode, matches []string, redirectTo *url.URL) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	n, matches := mux.tree.match(host, method, target)
	if !exactMatch(n, target) && u != nil && !strings.HasSuffix(target, "/") && target != "" {
		target += "/"
		n2, _ := mux.tree.match(host, method, target)
		if exactMatch(n2, target) {
			return n2, nil, urlFromEscaped(target, u.RawQuery)
		}
	}
	return n, matches, nil
}

// urlFromEscaped builds a URL of an escaped path and of a raw query. The 2
// fields Path and RawPath stay equal, thus URL.String writes no second escape.
func urlFromEscaped(escaped, rawQuery string) *url.URL {
	unescaped, err := url.PathUnescape(escaped)
	if err != nil {
		unescaped = escaped
	}
	return &url.URL{
		Path:     unescaped,
		RawPath:  escaped,
		RawQuery: rawQuery,
	}
}

// exactMatch reports whether the pattern of the node matches the whole path. A
// nil node gives false.
//
// A match where a multi wildcard takes a text that is not empty is not exact.
// Each other match is exact.
func exactMatch(n *routingNode, target string) bool {
	if n == nil {
		return false
	}
	if !n.pattern.lastSegment().multi {
		return true
	}
	if len(target) > 0 && target[len(target)-1] != '/' {
		return false
	}
	return len(n.pattern.segments) == strings.Count(target, "/")
}

// matchingMethods gives each method that matches the host and the path, in
// order.
func (mux *ServeMux) matchingMethods(host, target string) []string {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	ms := map[string]bool{}
	mux.tree.matchingMethods(host, target, ms)
	// matchOrRedirect adds a trailing slash when no pattern matches.
	if !strings.HasSuffix(target, "/") {
		mux.tree.matchingMethods(host, target+"/", ms)
	}
	return slices.Sorted(maps.Keys(ms))
}

// ServeHTTP gives the request to the handler of the pattern that matches its
// URL best.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	var h Handler
	h, r.Pattern, r.pat, r.matches = mux.findHandler(r)
	h.ServeHTTP(w, r)
}

// Handle registers the handler for the pattern. A pattern that conflicts with a
// registered pattern, and a pattern that holds a syntax error, each stop the
// program.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.register(pattern, handler)
}

// HandleFunc registers the function for the pattern. A pattern that conflicts
// with a registered pattern, and a pattern that holds a syntax error, each stop
// the program.
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	mux.register(pattern, HandlerFunc(handler))
}

// Handle registers the handler for the pattern in DefaultServeMux.
func Handle(pattern string, handler Handler) { DefaultServeMux.register(pattern, handler) }

// HandleFunc registers the function for the pattern in DefaultServeMux.
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.register(pattern, HandlerFunc(handler))
}

func (mux *ServeMux) register(pattern string, handler Handler) {
	if err := mux.registerErr(pattern, handler); err != nil {
		panic(err)
	}
}

func (mux *ServeMux) registerErr(patstr string, handler Handler) error {
	if patstr == "" {
		return errors.New("http: invalid pattern")
	}
	if handler == nil {
		return errors.New("http: nil handler")
	}
	if f, ok := handler.(HandlerFunc); ok && f == nil {
		return errors.New("http: nil handler")
	}

	pat, err := parsePattern(patstr)
	if err != nil {
		return fmt.Errorf("parsing %q: %w", patstr, err)
	}

	// The position of the caller gives a better message of a conflict. The
	// count of 3 skips register and the function that called it.
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		pat.loc = "unknown location"
	} else {
		pat.loc = fmt.Sprintf("%s:%d", file, line)
	}

	mux.mu.Lock()
	defer mux.mu.Unlock()
	if err := mux.index.possiblyConflictingPatterns(pat, func(pat2 *pattern) error {
		if pat.conflictsWith(pat2) {
			d := describeConflict(pat, pat2)
			return fmt.Errorf("pattern %q (registered at %s) conflicts with pattern %q (registered at %s):\n%s",
				pat, pat.loc, pat2, pat2.loc, d)
		}
		return nil
	}); err != nil {
		return err
	}
	mux.tree.addPattern(pat, handler)
	mux.index.addPattern(pat)
	return nil
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
func (s *Server) serveH2(ctx context.Context, conn int) {
	session, cause := dart.H2Server(conn)
	if cause != "" {
		return
	}
	defer dart.H2Close(session)
	for {
		stream, method, target, authority, flat, cause := dart.H2Next(session)
		if cause != "" {
			return
		}
		go s.serveH2Stream(ctx, stream, method, target, authority, flat)
	}
}

func (s *Server) serveH2Stream(ctx context.Context, stream int, method, path, authority string, flat []string) {
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
		ctx:           ctx,
	}
	w := &h2Response{stream: stream, header: Header{}, req: req}
	serverHandler{s}.ServeHTTP(w, req)
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
	if _, exists := w.header["Content-Type"]; !exists && w.header.Get("Content-Encoding") == "" && len(w.body) > 0 {
		w.header.Set("Content-Type", DetectContentType(w.body))
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
		if w.header.Get("Content-Length") == "" {
			w.header.Set("Content-Length", itoa(len(w.body)))
		}
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
