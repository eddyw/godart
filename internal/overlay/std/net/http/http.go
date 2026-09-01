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

// The status codes of Go.
const (
	StatusContinue           = 100
	StatusSwitchingProtocols = 101

	StatusOK                           = 200
	StatusCreated                      = 201
	StatusAccepted                     = 202
	StatusNoContent                    = 204
	StatusResetContent                 = 205
	StatusPartialContent               = 206
	StatusMultipleChoices              = 300
	StatusMovedPermanently             = 301
	StatusFound                        = 302
	StatusSeeOther                     = 303
	StatusNotModified                  = 304
	StatusPreconditionFailed           = 412
	StatusRequestedRangeNotSatisfiable = 416
	StatusTemporaryRedirect            = 307
	StatusPermanentRedirect            = 308
	StatusBadRequest                   = 400
	StatusUnauthorized                 = 401
	StatusPaymentRequired              = 402
	StatusForbidden                    = 403
	StatusNotFound                     = 404
	StatusMethodNotAllowed             = 405
	StatusNotAcceptable                = 406
	StatusRequestTimeout               = 408
	StatusConflict                     = 409
	StatusGone                         = 410
	StatusLengthRequired               = 411
	StatusRequestEntityTooLarge        = 413
	StatusRequestURITooLong            = 414
	StatusUnsupportedMediaType         = 415
	StatusTeapot                       = 418
	StatusTooManyRequests              = 429

	StatusInternalServerError     = 500
	StatusNotImplemented          = 501
	StatusBadGateway              = 502
	StatusServiceUnavailable      = 503
	StatusGatewayTimeout          = 504
	StatusHTTPVersionNotSupported = 505
)

var statusText = map[int]string{
	StatusContinue:           "Continue",
	StatusSwitchingProtocols: "Switching Protocols",

	StatusOK:                           "OK",
	StatusCreated:                      "Created",
	StatusAccepted:                     "Accepted",
	StatusNoContent:                    "No Content",
	StatusResetContent:                 "Reset Content",
	StatusPartialContent:               "Partial Content",
	StatusMultipleChoices:              "Multiple Choices",
	StatusMovedPermanently:             "Moved Permanently",
	StatusFound:                        "Found",
	StatusSeeOther:                     "See Other",
	StatusNotModified:                  "Not Modified",
	StatusPreconditionFailed:           "Precondition Failed",
	StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	StatusTemporaryRedirect:            "Temporary Redirect",
	StatusPermanentRedirect:            "Permanent Redirect",
	StatusBadRequest:                   "Bad Request",
	StatusUnauthorized:                 "Unauthorized",
	StatusPaymentRequired:              "Payment Required",
	StatusForbidden:                    "Forbidden",
	StatusNotFound:                     "Not Found",
	StatusMethodNotAllowed:             "Method Not Allowed",
	StatusNotAcceptable:                "Not Acceptable",
	StatusRequestTimeout:               "Request Timeout",
	StatusConflict:                     "Conflict",
	StatusGone:                         "Gone",
	StatusLengthRequired:               "Length Required",
	StatusRequestEntityTooLarge:        "Request Entity Too Large",
	StatusRequestURITooLong:            "Request URI Too Long",
	StatusUnsupportedMediaType:         "Unsupported Media Type",
	StatusTeapot:                       "I'm a teapot",
	StatusTooManyRequests:              "Too Many Requests",

	StatusInternalServerError:     "Internal Server Error",
	StatusNotImplemented:          "Not Implemented",
	StatusBadGateway:              "Bad Gateway",
	StatusServiceUnavailable:      "Service Unavailable",
	StatusGatewayTimeout:          "Gateway Timeout",
	StatusHTTPVersionNotSupported: "HTTP Version Not Supported",
}

// StatusText gives the text of a status code, and it gives an empty string for
// a code that the package does not hold.
func StatusText(code int) string { return statusText[code] }

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
