//go:build dart

package httptest

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// A ResponseRecorder holds the answer that a handler writes. A test reads the
// fields of it, or it reads the Response that Result gives.
type ResponseRecorder struct {
	// Code holds the status that WriteHeader wrote.
	Code int
	// HeaderMap holds each field that the handler set.
	HeaderMap http.Header
	// Body holds each byte that the handler wrote. A nil buffer drops them.
	Body *bytes.Buffer
	// Flushed marks a handler that called Flush.
	Flushed bool

	result      *http.Response
	snapHeader  http.Header
	wroteHeader bool
}

// DefaultRemoteAddr is the address that a request of a recorder names.
const DefaultRemoteAddr = "1.2.3.4"

// NewRecorder builds a recorder with the status 200 and an empty body.
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		HeaderMap: http.Header{},
		Body:      &bytes.Buffer{},
		Code:      200,
	}
}

// Header gives the fields of the answer.
func (rw *ResponseRecorder) Header() http.Header {
	if rw.HeaderMap == nil {
		rw.HeaderMap = http.Header{}
	}
	return rw.HeaderMap
}

// Write writes the bytes to the body.
func (rw *ResponseRecorder) Write(buf []byte) (int, error) {
	rw.writeHeader()
	if rw.Body != nil {
		rw.Body.Write(buf)
	}
	if !bodyAllowedForStatus(rw.Code) {
		return 0, http.ErrBodyNotAllowed
	}
	return len(buf), nil
}

// WriteString writes the text to the body.
func (rw *ResponseRecorder) WriteString(str string) (int, error) {
	rw.writeHeader()
	if rw.Body != nil {
		rw.Body.WriteString(str)
	}
	if !bodyAllowedForStatus(rw.Code) {
		return 0, http.ErrBodyNotAllowed
	}
	return len(str), nil
}

// writeHeader writes the status 200 for the first write, and it names the type
// of the body.
//
// Go reads the first 512 bytes of the body and names the type. The overlay
// names text only, exactly as the server of the overlay does.
func (rw *ResponseRecorder) writeHeader() {
	if rw.wroteHeader {
		return
	}
	m := rw.Header()
	_, hasType := m["Content-Type"]
	hasTE := m.Get("Transfer-Encoding") != ""
	if !hasType && !hasTE {
		m.Set("Content-Type", "text/plain; charset=utf-8")
	}
	rw.WriteHeader(200)
}

// bodyAllowedForStatus reports whether the status holds a body.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

// WriteHeader holds the status of the answer. A second call performs no
// operation.
func (rw *ResponseRecorder) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	if code < 100 || code > 999 {
		panic("invalid WriteHeader code " + strconv.Itoa(code))
	}
	rw.Code = code
	rw.wroteHeader = true
	if rw.HeaderMap == nil {
		rw.HeaderMap = http.Header{}
	}
	rw.snapHeader = rw.HeaderMap.Clone()
}

// Flush marks the answer as flushed.
func (rw *ResponseRecorder) Flush() {
	if !rw.wroteHeader {
		rw.WriteHeader(200)
	}
	rw.Flushed = true
}

// Result gives the answer of the handler. The header holds the fields of the
// first write, or the fields of this call for a handler that wrote none.
func (rw *ResponseRecorder) Result() *http.Response {
	if rw.result != nil {
		return rw.result
	}
	if rw.snapHeader == nil {
		rw.snapHeader = rw.HeaderMap.Clone()
	}
	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: rw.Code,
		Header:     rw.snapHeader,
	}
	rw.result = res
	if res.StatusCode == 0 {
		res.StatusCode = 200
	}
	res.Status = pad3(res.StatusCode) + " " + http.StatusText(res.StatusCode)
	if rw.Body != nil {
		res.Body = io.NopCloser(bytes.NewReader(rw.Body.Bytes()))
	} else {
		res.Body = io.NopCloser(strings.NewReader(""))
	}
	res.ContentLength = parseContentLength(res.Header.Get("Content-Length"))
	return res
}

// pad3 writes a status with a minimum of 3 digits, exactly as the verb %03d
// of Go does.
func pad3(code int) string {
	text := strconv.Itoa(code)
	for len(text) < 3 {
		text = "0" + text
	}
	return text
}

// parseContentLength reads the field Content-Length. A field that holds no
// number gives -1.
func parseContentLength(text string) int64 {
	text = strings.TrimSpace(text)
	if text == "" {
		return -1
	}
	n, err := strconv.ParseUint(text, 10, 63)
	if err != nil {
		return -1
	}
	return int64(n)
}
