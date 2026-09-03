//go:build dart

package http

import (
	"crypto/tls"
	"errors"
	"slices"
)

// This file holds the TLS listener of the server.
//
// `crypto/tls` opens a secure listener of its own over the secure socket of
// Dart, thus `ListenAndServeTLS` works. `ServeTLS` and `Server.ServeTLS` each
// take a listener that stands open and wrap it, and the secure socket of Dart
// takes over no socket that another call opened. Thus the overlay holds
// neither of the 2, and `tls.NewListener` stays out for the same cause.

// ListenAndServeTLS listens on the address and answers each request over TLS.
// The 2 file names hold the certificate chain and the private key, each in
// the PEM form.
//
// A chain of more than 1 certificate holds the certificate of the server
// first, and then each certificate of the authority above it.
func ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServeTLS(certFile, keyFile)
}

// ListenAndServeTLS listens on the address of the server and answers each
// request over TLS. An empty Addr names the port 443.
//
// The 2 file names hold the certificate chain and the private key, each in
// the PEM form. An empty pair of names reads the field Certificates of
// TLSConfig.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	if s.shuttingDown() {
		return ErrServerClosed
	}
	if err := s.checkProtocols(); err != nil {
		return err
	}
	addr := s.Addr
	if addr == "" {
		addr = ":443"
	}
	config := s.TLSConfig.Clone()
	if config == nil {
		config = &tls.Config{}
	}
	if len(config.Certificates) == 0 || certFile != "" || keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
		config.Certificates = []tls.Certificate{cert}
	}
	config.NextProtos = alpnList(config.NextProtos, s.alpn())
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

// alpnList joins the list of the config and the list of the server. A
// protocol of the config keeps its place, and each protocol of the server
// that the config misses follows it.
func alpnList(config, server []string) []string {
	out := slices.Clone(config)
	for _, proto := range server {
		if !slices.Contains(out, proto) {
			out = append(out, proto)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// errNoServeTLS names the cause that keeps ServeTLS out of the overlay. No
// code reads it, and the text stands here for a reader of the source.
var errNoServeTLS = errors.New("http: the secure socket of Dart takes over no " +
	"socket that another call opened, thus ServeTLS stays out")
