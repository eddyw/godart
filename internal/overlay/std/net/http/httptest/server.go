//go:build dart

package httptest

import (
	"net"
	"net/http"
)

// A Server holds a listener on a port of the machine and answers each request
// with a handler.
//
// The real Server of Go waits for each request that stands open, and it holds a
// TLS form. Close of the overlay ends the listener and each connection of the
// client that stands open with no request.
type Server struct {
	// URL holds the base address of the server, for example
	// `http://127.0.0.1:52341`.
	URL string
	// Listener holds the listener of the server.
	Listener net.Listener
	// Config holds the server that answers each request.
	Config *http.Server

	client *http.Client
}

// NewServer builds a server and starts it.
func NewServer(handler http.Handler) *Server {
	s := NewUnstartedServer(handler)
	s.Start()
	return s
}

// NewUnstartedServer builds a server that holds a listener and answers no
// request. Start answers each request.
func NewUnstartedServer(handler http.Handler) *Server {
	return &Server{
		Listener: newLocalListener(),
		Config:   &http.Server{Handler: handler},
	}
}

// Start answers each request of the listener.
func (s *Server) Start() {
	if s.URL != "" {
		panic("httptest: server already started")
	}
	s.URL = "http://" + s.Listener.Addr().String()
	s.client = &http.Client{}
	go s.Config.Serve(s.Listener)
}

// Close ends the listener and each connection of the client that stands open
// with no request.
func (s *Server) Close() {
	s.Config.Close()
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
}

// Client gives a client that reaches the server.
func (s *Server) Client() *http.Client {
	if s.client == nil {
		s.client = &http.Client{}
	}
	return s.client
}

// newLocalListener binds a port of the loopback address.
func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("httptest: failed to listen on a port: " + err.Error())
	}
	return l
}
