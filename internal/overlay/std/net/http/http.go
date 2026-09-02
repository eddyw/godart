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

// ErrBodyNotAllowed reports a body of a response that holds none.
var ErrBodyNotAllowed = errors.New("http: request method or response status code does not allow body")

// ErrNotSupported reports a call that the overlay does not hold.
var ErrNotSupported = errors.New("http: the operation is outside the subset")

// ErrUseLastResponse stops the redirects of a client. CheckRedirect gives it.
var ErrUseLastResponse = errors.New("net/http: use last response")

// ErrServerClosed reports the end of a server after Close.
var ErrServerClosed = errors.New("http: Server closed")

// itoa gives the text of a whole number.
func itoa(n int) string { return strconv.Itoa(n) }
