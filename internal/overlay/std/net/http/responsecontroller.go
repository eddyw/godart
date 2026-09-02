//go:build dart

package http

import (
	"bufio"
	"net"
	"time"
)

// A ResponseController lets a handler drive the answer of 1 request. A handler
// that ends must touch it no more.
//
// The controller reads the methods of the ResponseWriter that the handler
// holds. A writer that wraps another one gives the inner writer through an
// Unwrap method, and the controller walks the chain. A writer that holds no
// method of an operation gives an error that matches ErrNotSupported.
type ResponseController struct {
	rw ResponseWriter
}

// NewResponseController builds a ResponseController of a ResponseWriter.
func NewResponseController(rw ResponseWriter) *ResponseController {
	return &ResponseController{rw: rw}
}

// rwUnwrapper names a ResponseWriter that wraps another one.
type rwUnwrapper interface {
	Unwrap() ResponseWriter
}

// flushErrorWriter names a writer whose flush gives an error.
type flushErrorWriter interface {
	FlushError() error
}

// readDeadlineWriter names a writer that holds the read deadline.
type readDeadlineWriter interface {
	SetReadDeadline(deadline time.Time) error
}

// writeDeadlineWriter names a writer that holds the write deadline.
type writeDeadlineWriter interface {
	SetWriteDeadline(deadline time.Time) error
}

// fullDuplexWriter names a writer that reads the body of the request and
// writes the answer at the same time.
type fullDuplexWriter interface {
	EnableFullDuplex() error
}

// notSupported wraps ErrNotSupported. Go gives an error that matches
// ErrNotSupported and that is not equal to it, and the overlay does the same.
type notSupported struct{}

func (notSupported) Error() string { return ErrNotSupported.Error() }

func (notSupported) Unwrap() error { return ErrNotSupported }

// errNotSupported gives the error of an operation that the writer holds no
// method of.
func errNotSupported() error { return notSupported{} }

// Flush writes the bytes of the answer that stand in the buffer.
func (c *ResponseController) Flush() error {
	rw := c.rw
	for {
		switch t := rw.(type) {
		case flushErrorWriter:
			return t.FlushError()
		case Flusher:
			t.Flush()
			return nil
		case rwUnwrapper:
			rw = t.Unwrap()
		default:
			return errNotSupported()
		}
	}
}

// Hijack gives the connection of the request to the caller. The caller then
// owns the connection, and the server writes no answer of its own. Read the
// Hijacker interface.
func (c *ResponseController) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw := c.rw
	for {
		switch t := rw.(type) {
		case Hijacker:
			return t.Hijack()
		case rwUnwrapper:
			rw = t.Unwrap()
		default:
			return nil, nil, errNotSupported()
		}
	}
}

// SetReadDeadline writes the deadline of the read of the whole request. The
// zero Time holds no deadline.
func (c *ResponseController) SetReadDeadline(deadline time.Time) error {
	rw := c.rw
	for {
		switch t := rw.(type) {
		case readDeadlineWriter:
			return t.SetReadDeadline(deadline)
		case rwUnwrapper:
			rw = t.Unwrap()
		default:
			return errNotSupported()
		}
	}
}

// SetWriteDeadline writes the deadline of the write of the answer. The zero
// Time holds no deadline.
func (c *ResponseController) SetWriteDeadline(deadline time.Time) error {
	rw := c.rw
	for {
		switch t := rw.(type) {
		case writeDeadlineWriter:
			return t.SetWriteDeadline(deadline)
		case rwUnwrapper:
			rw = t.Unwrap()
		default:
			return errNotSupported()
		}
	}
}

// EnableFullDuplex lets the handler read the body of the request while it
// writes the answer.
func (c *ResponseController) EnableFullDuplex() error {
	rw := c.rw
	for {
		switch t := rw.(type) {
		case fullDuplexWriter:
			return t.EnableFullDuplex()
		case rwUnwrapper:
			rw = t.Unwrap()
		default:
			return errNotSupported()
		}
	}
}
