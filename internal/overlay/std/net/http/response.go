//go:build dart

package http

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// A Response holds the answer of a server.
type Response struct {
	Status     string
	StatusCode int
	Proto      string
	ProtoMajor int
	ProtoMinor int
	Header     Header
	Body       io.ReadCloser
	// ContentLength holds the count of bytes of the body. A value of -1 marks a
	// length that the header does not name.
	ContentLength int64
	// Close marks a connection that ends after the body.
	Close bool
	// Request holds the request that the response answers.
	Request *Request
	// Uncompressed marks a body that the transport read back from the gzip
	// form.
	Uncompressed bool
	// TransferEncoding lists the encoding of the body, from the outermost to
	// the innermost. The overlay writes the 1 element "chunked", or nothing.
	TransferEncoding []string
	// Trailer holds the fields that follow the body. The overlay reads no
	// trailer, thus this map stays nil.
	Trailer Header
	// TLS holds the state of the secure connection that carried the answer. A
	// plain connection leaves it nil.
	TLS *tls.ConnectionState

	// bodyEnds marks a body whose end the header names. A body without such an
	// end reaches the end of the connection, thus the connection holds no
	// second answer.
	bodyEnds bool
}

// ReadResponse reads 1 response from the reader. The request holds the request
// that the response answers, and a nil request names a GET. The caller closes
// the body of the result.
func ReadResponse(r *bufio.Reader, req *Request) (*Response, error) {
	return readResponseLimit(r, req, 0)
}

// readResponse reads 1 response from a connection.
func readResponse(r *bufio.Reader, req *Request) (*Response, error) {
	return readResponseLimit(r, req, 0)
}

// readResponseLimit reads 1 response. A limit above 0 names the count of bytes
// that the status line and the head hold together.
func readResponseLimit(r *bufio.Reader, req *Request, limit int) (*Response, error) {
	left := -1
	if limit > 0 {
		left = limit
	}
	line, err := readLineLimit(r, &left)
	if err != nil {
		return nil, err
	}
	proto, rest, ok := strings.Cut(line, " ")
	if !ok {
		return nil, errors.New("malformed HTTP response " + strconv.Quote(line))
	}
	major, minor, ok := ParseHTTPVersion(proto)
	if !ok {
		return nil, errors.New("malformed HTTP version " + strconv.Quote(proto))
	}
	code, text, _ := strings.Cut(rest, " ")
	status, err := strconv.Atoi(code)
	if err != nil {
		return nil, errors.New("malformed HTTP status " + strconv.Quote(code))
	}
	header, err := readHeaderLimit(r, &left, 0)
	if err != nil {
		return nil, err
	}
	res := &Response{
		Status:     strings.TrimSpace(rest),
		StatusCode: status,
		Proto:      proto,
		ProtoMajor: major,
		ProtoMinor: minor,
		Header:     header,
		Request:    req,
		Close:      shouldClose(major, minor, header, true),
	}
	_ = text
	withBody := req == nil || req.Method != MethodHead
	if status == StatusNoContent || status == StatusNotModified || status < 200 {
		withBody = false
	}
	body, length, ends, err := readBody(r, header, withBody, true)
	if err != nil {
		return nil, err
	}
	res.Body = body
	res.ContentLength = length
	if strings.EqualFold(header.Get("Transfer-Encoding"), "chunked") {
		res.TransferEncoding = []string{"chunked"}
		header.Del("Transfer-Encoding")
		if withBody {
			header.Del("Content-Length")
		}
	}
	if req != nil && req.Method == MethodHead {
		res.ContentLength = -1
		if text := header.Get("Content-Length"); text != "" {
			n, err := strconv.ParseInt(text, 10, 64)
			if err != nil || n < 0 {
				return nil, errors.New("bad Content-Length " + strconv.Quote(text))
			}
			res.ContentLength = n
		}
	}
	res.bodyEnds = ends
	return res, nil
}

// errHeaderTooLarge reports a head of a message that holds more bytes than the
// limit. The server answers such a request with the status 431.
var errHeaderTooLarge = errors.New("http: request header fields too large")

// readLine reads 1 line and drops the end of the line.
func readLine(r *bufio.Reader) (string, error) {
	left := -1
	return readLineLimit(r, &left)
}

// readLineLimit reads 1 line and drops the end of the line. The count that left
// names holds the bytes that the head may still hold, and a negative count
// holds no limit. Each read takes its bytes from that count.
func readLineLimit(r *bufio.Reader, left *int) (string, error) {
	var out []byte
	for {
		part, err := r.ReadSlice('\n')
		if *left >= 0 {
			*left -= len(part)
			if *left < 0 {
				return "", errHeaderTooLarge
			}
		}
		out = append(out, part...)
		if err == nil {
			break
		}
		if err != bufio.ErrBufferFull {
			return "", err
		}
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// readHeader reads each field up to the empty line.
func readHeader(r *bufio.Reader) (Header, error) {
	left := -1
	return readHeaderLimit(r, &left, 0)
}

// readHeaderLimit reads each field up to the empty line, under the limit of the
// bytes that left holds and under the limit of the count of the lines. A
// maxValues of 0 holds no limit of the count.
func readHeaderLimit(r *bufio.Reader, left *int, maxValues int) (Header, error) {
	out := Header{}
	count := 0
	for {
		line, err := readLineLimit(r, left)
		if err != nil {
			return nil, err
		}
		if line == "" {
			return out, nil
		}
		count++
		if maxValues > 0 && count > maxValues {
			return nil, errHeaderTooLarge
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, errors.New("malformed header line " + strconv.Quote(line))
		}
		out.Add(strings.TrimSpace(key), strings.TrimSpace(value))
	}
}

// ProtoAtLeast reports whether the version of the protocol of the response
// reaches the major and the minor number.
func (r *Response) ProtoAtLeast(major, minor int) bool {
	return r.ProtoMajor > major ||
		r.ProtoMajor == major && r.ProtoMinor >= minor
}

// Cookies gives each cookie of the field Set-Cookie of the response.
func (r *Response) Cookies() []*Cookie { return readSetCookies(r.Header) }

// ErrNoLocation reports an answer with no field Location. Response.Location
// gives it.
var ErrNoLocation = errors.New("http: no Location header in response")

// Location gives the URL of the field Location of the answer. A relative URL
// reads the URL of the request of the answer. An answer with no such field
// gives ErrNoLocation.
func (r *Response) Location() (*url.URL, error) {
	lv := r.Header.Get("Location")
	if lv == "" {
		return nil, ErrNoLocation
	}
	if r.Request != nil && r.Request.URL != nil {
		return r.Request.URL.Parse(lv)
	}
	return url.Parse(lv)
}

// Write writes the response in the form of the wire.
//
// The status line reads Status, and an empty Status reads StatusText of the
// code. A ContentLength of 0 or more names the length of the body. A negative
// ContentLength with a body writes the chunked form for HTTP/1.1, and the end
// of the connection then names the end of a body of HTTP/1.0.
//
// The call reads and closes the body of the response.
func (r *Response) Write(w io.Writer) error {
	text := r.Status
	if text == "" {
		text = StatusText(r.StatusCode)
		if text == "" {
			text = "status code " + itoa(r.StatusCode)
		}
	} else {
		// A Status of the form "200 OK" next to a StatusCode of 200 writes the
		// number 1 time only.
		text = strings.TrimPrefix(text, itoa(r.StatusCode)+" ")
	}
	major, minor := r.ProtoMajor, r.ProtoMinor
	if major == 0 {
		major, minor = 1, 1
	}
	head := "HTTP/" + itoa(major) + "." + itoa(minor) + " " +
		statusDigits(r.StatusCode) + " " + text + "\r\n"
	if _, err := io.WriteString(w, head); err != nil {
		return err
	}
	header := r.Header.Clone()
	if header == nil {
		header = Header{}
	}
	header.Del("Content-Length")
	header.Del("Transfer-Encoding")
	chunked := false
	switch {
	case r.ContentLength >= 0:
		header.Set("Content-Length", strconv.FormatInt(r.ContentLength, 10))
	case r.Body != nil && (major > 1 || major == 1 && minor >= 1):
		header.Set("Transfer-Encoding", "chunked")
		chunked = true
	}
	if r.Close {
		header.Set("Connection", "close")
	}
	if err := header.Write(w); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	if !chunked {
		_, err := io.Copy(w, r.Body)
		return err
	}
	return writeChunked(w, r.Body)
}

// statusDigits writes a status code with 3 digits at least.
func statusDigits(code int) string {
	out := itoa(code)
	for len(out) < 3 {
		out = "0" + out
	}
	return out
}

// writeChunked writes a body in the chunked form.
func writeChunked(w io.Writer, body io.Reader) error {
	buf := make([]byte, 4096)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			head := strconv.FormatInt(int64(n), 16) + "\r\n"
			if _, e := io.WriteString(w, head); e != nil {
				return e
			}
			if _, e := w.Write(buf[:n]); e != nil {
				return e
			}
			if _, e := io.WriteString(w, "\r\n"); e != nil {
				return e
			}
		}
		if err == io.EOF {
			_, e := io.WriteString(w, "0\r\n\r\n")
			return e
		}
		if err != nil {
			return err
		}
	}
}

// readBody gives the reader of a body. The header names the length or the
// chunked form. A message with neither holds no body, and toEnd marks a message
// whose body reaches the end of the connection. The third result marks a body
// whose end the header names.
func readBody(r *bufio.Reader, header Header, withBody, toEnd bool) (io.ReadCloser, int64, bool, error) {
	if !withBody {
		return io.NopCloser(strings.NewReader("")), 0, true, nil
	}
	if strings.EqualFold(header.Get("Transfer-Encoding"), "chunked") {
		return io.NopCloser(&chunkedReader{r: r}), -1, true, nil
	}
	if text := header.Get("Content-Length"); text != "" {
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil || n < 0 {
			return nil, 0, false, errors.New("bad Content-Length " + strconv.Quote(text))
		}
		return io.NopCloser(io.LimitReader(r, n)), n, true, nil
	}
	if !toEnd {
		return io.NopCloser(strings.NewReader("")), 0, true, nil
	}
	return io.NopCloser(r), -1, false, nil
}

// A chunkedReader reads a body of the chunked form.
type chunkedReader struct {
	r    *bufio.Reader
	left int64
	done bool
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if c.done {
		return 0, io.EOF
	}
	if c.left == 0 {
		line, err := readLine(c.r)
		if err != nil {
			return 0, err
		}
		size, _, _ := strings.Cut(line, ";")
		n, err := strconv.ParseInt(strings.TrimSpace(size), 16, 64)
		if err != nil || n < 0 {
			return 0, errors.New("bad chunk size " + strconv.Quote(line))
		}
		if n == 0 {
			c.done = true
			// The trailer stands after the last chunk, and the empty line ends
			// it.
			for {
				text, err := readLine(c.r)
				if err != nil || text == "" {
					return 0, io.EOF
				}
			}
		}
		c.left = n
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	if c.left == 0 && err == nil {
		if _, err := readLine(c.r); err != nil {
			return n, err
		}
	}
	return n, err
}
