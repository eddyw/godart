//go:build dart

// Package http gives an HTTP/1.1 client and an HTTP/1.1 server over the net
// package.
//
// The real source of Go reads a vendored HTTP/2 stack, a cookie jar, a file
// transport, and each other part of the standard library. The overlay holds the
// client, the server, and the message format only. `docs/parity.md` lists each
// gap.
package http

import (
	"errors"
	"net/http/internal"
	"strconv"
)

// The methods of Go.
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

// A ProtocolError names an error of the format of HTTP. Go marks the type as
// deprecated, and 6 variables of the package still hold it.
type ProtocolError struct {
	ErrorString string
}

// Error gives the text of the error.
func (pe *ProtocolError) Error() string { return pe.ErrorString }

// Is reports that ErrNotSupported matches errors.ErrUnsupported.
func (pe *ProtocolError) Is(err error) bool {
	return pe == ErrNotSupported && err == errors.ErrUnsupported
}

// The 6 errors of the format that hold the type ProtocolError.
var (
	// ErrNotSupported reports an operation that the transport, the server, or
	// the answer of a request does not hold. It matches errors.ErrUnsupported.
	ErrNotSupported = &ProtocolError{"feature not supported"}

	// ErrUnexpectedTrailer reports a field Trailer of a message that holds no
	// chunked transfer encoding.
	ErrUnexpectedTrailer = &ProtocolError{"trailer header without chunked transfer encoding"}

	// ErrHeaderTooLong reports a head of a message that is too long.
	ErrHeaderTooLong = &ProtocolError{"header too long"}

	// ErrShortBody reports a body that ends before its length.
	ErrShortBody = &ProtocolError{"entity body too short"}

	// ErrMissingContentLength reports an answer of a HEAD request with no
	// field Content-Length.
	ErrMissingContentLength = &ProtocolError{"missing ContentLength in HEAD response"}
)

// The errors of the server and of the answer of a request.
var (
	// ErrBodyNotAllowed reports a write of a body on an answer that holds
	// none.
	ErrBodyNotAllowed = errors.New("http: request method or response status code does not allow body")

	// ErrContentLength reports a write of a body that is longer than the field
	// Content-Length of the answer.
	ErrContentLength = errors.New("http: wrote more than the declared Content-Length")

	// ErrWriteAfterFlush holds no meaning any more. Go keeps the name.
	ErrWriteAfterFlush = errors.New("unused")

	// ErrBodyReadAfterClose reports a read of the body of a request or of an
	// answer after the Close of that body.
	ErrBodyReadAfterClose = errors.New("http: invalid Read on closed Body")

	// ErrAbortHandler is a panic value of a handler. The server ends the
	// connection and writes no log for it.
	ErrAbortHandler = internal.ErrAbortHandler

	// ErrLineTooLong reports a line of the head or of a chunk that is too
	// long.
	ErrLineTooLong = internal.ErrLineTooLong

	// ErrSkipAltProtocol is the answer of a RoundTripper of
	// Transport.RegisterProtocol that gives the request back to the transport.
	ErrSkipAltProtocol = internal.ErrSkipAltProtocol

	// ErrSchemeMismatch reports a plain answer of HTTP on a connection of
	// HTTPS.
	ErrSchemeMismatch = errors.New("http: server gave HTTP response to HTTPS client")

	// ErrUseLastResponse stops the redirects of a client. CheckRedirect gives
	// it.
	ErrUseLastResponse = errors.New("net/http: use last response")

	// ErrServerClosed reports the end of a server after Close or after
	// Shutdown.
	ErrServerClosed = errors.New("http: Server closed")
)

// itoa gives the text of a whole number.
func itoa(n int) string { return strconv.Itoa(n) }

// isToken reports whether the text holds a valid token of RFC 2616, section
// 2.2. A name of a field of a header holds the same rule.
func isToken(v string) bool { return validHeaderKey(v) }

// stringContainsCTLByte reports whether the text holds a control character of
// ASCII.
func stringContainsCTLByte(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < ' ' || b == 0x7f {
			return true
		}
	}
	return false
}
