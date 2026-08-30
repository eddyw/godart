//go:build dart

package http

import (
	"bufio"
	"errors"
	"io"
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

	// bodyEnds marks a body whose end the header names. A body without such an
	// end reaches the end of the connection, thus the connection holds no
	// second answer.
	bodyEnds bool
}

// readResponse reads 1 response from a connection.
func readResponse(r *bufio.Reader, req *Request) (*Response, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	proto, rest, ok := strings.Cut(line, " ")
	if !ok {
		return nil, errors.New("malformed HTTP response " + strconv.Quote(line))
	}
	code, text, _ := strings.Cut(rest, " ")
	status, err := strconv.Atoi(code)
	if err != nil {
		return nil, errors.New("malformed HTTP status " + strconv.Quote(code))
	}
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	res := &Response{
		Status:     strings.TrimSpace(rest),
		StatusCode: status,
		Proto:      proto,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Request:    req,
		Close:      strings.EqualFold(header.Get("Connection"), "close"),
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
	res.bodyEnds = ends
	return res, nil
}

// readLine reads 1 line and drops the end of the line.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// readHeader reads each field up to the empty line.
func readHeader(r *bufio.Reader) (Header, error) {
	out := Header{}
	for {
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		if line == "" {
			return out, nil
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, errors.New("malformed header line " + strconv.Quote(line))
		}
		out.Add(strings.TrimSpace(key), strings.TrimSpace(value))
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
