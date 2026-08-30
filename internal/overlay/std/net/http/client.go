//go:build dart

package http

import (
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// A Client sends a request and reads the answer. It follows a redirect.
type Client struct {
	// Transport runs 1 request. A nil transport reads DefaultTransport.
	Transport RoundTripper
	// CheckRedirect decides each redirect. A nil function follows up to 10
	// redirects, exactly as in Go.
	CheckRedirect func(req *Request, via []*Request) error
	// Timeout ends a request that takes too long. A value of 0 holds no limit.
	Timeout time.Duration
}

// DefaultClient is the client of the top level functions.
var DefaultClient = &Client{}

// CloseIdleConnections ends each connection of the transport that stands open
// with no request.
func (c *Client) CloseIdleConnections() {
	transport := c.Transport
	if transport == nil {
		transport = DefaultTransport
	}
	if t, ok := transport.(interface{ CloseIdleConnections() }); ok {
		t.CloseIdleConnections()
	}
}

// Get sends a GET request.
func Get(url string) (*Response, error) { return DefaultClient.Get(url) }

// Head sends a HEAD request.
func Head(url string) (*Response, error) { return DefaultClient.Head(url) }

// Post sends a POST request with a body.
func Post(url, contentType string, body io.Reader) (*Response, error) {
	return DefaultClient.Post(url, contentType, body)
}

// PostForm sends a POST request with a body of the form type.
func PostForm(url string, data url.Values) (*Response, error) {
	return DefaultClient.PostForm(url, data)
}

// Get sends a GET request.
func (c *Client) Get(target string) (*Response, error) {
	req, err := NewRequest(MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Head sends a HEAD request.
func (c *Client) Head(target string) (*Response, error) {
	req, err := NewRequest(MethodHead, target, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post sends a POST request with a body.
func (c *Client) Post(target, contentType string, body io.Reader) (*Response, error) {
	req, err := NewRequest(MethodPost, target, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// PostForm sends a POST request with a body of the form type.
func (c *Client) PostForm(target string, data url.Values) (*Response, error) {
	return c.Post(target, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
}

// Do sends 1 request and gives the answer. It follows a redirect, and the
// caller closes the body of the answer.
func (c *Client) Do(req *Request) (*Response, error) {
	if req.URL == nil {
		return nil, &url.Error{Op: methodOp(req.Method), URL: "", Err: errors.New("http: nil Request.URL")}
	}
	transport := c.Transport
	if transport == nil {
		transport = DefaultTransport
	}
	if c.Timeout > 0 {
		if _, ok := req.Context().Deadline(); !ok {
			req = req.withDeadline(time.Now().Add(c.Timeout))
		}
	}
	var via []*Request
	for {
		res, err := transport.RoundTrip(req)
		if err != nil {
			return nil, &url.Error{Op: methodOp(req.Method), URL: req.URL.String(), Err: err}
		}
		next, err := c.redirect(req, res, via)
		if err != nil {
			res.Body.Close()
			return nil, &url.Error{Op: methodOp(req.Method), URL: req.URL.String(), Err: err}
		}
		if next == nil {
			return res, nil
		}
		res.Body.Close()
		via = append(via, req)
		req = next
	}
}

// redirect gives the next request of a redirect, and it gives nil for an answer
// that ends the chain.
func (c *Client) redirect(req *Request, res *Response, via []*Request) (*Request, error) {
	if !redirectCode(res.StatusCode) {
		return nil, nil
	}
	location := res.Header.Get("Location")
	if location == "" {
		return nil, nil
	}
	target, err := req.URL.Parse(location)
	if err != nil {
		return nil, errors.New("failed to parse Location header " + strconv.Quote(location))
	}
	method := req.Method
	if res.StatusCode == StatusSeeOther || res.StatusCode == StatusFound ||
		res.StatusCode == StatusMovedPermanently {
		if method != MethodHead {
			method = MethodGet
		}
	}
	next, err := NewRequestWithContext(req.Context(), method, target.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Header {
		if k == "Content-Length" || k == "Content-Type" {
			continue
		}
		next.Header[k] = append([]string(nil), v...)
	}
	check := c.CheckRedirect
	if check == nil {
		check = defaultCheckRedirect
	}
	if err := check(next, append(via, req)); err != nil {
		if errors.Is(err, ErrUseLastResponse) {
			return nil, nil
		}
		return nil, err
	}
	return next, nil
}

func defaultCheckRedirect(req *Request, via []*Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

func redirectCode(code int) bool {
	switch code {
	case StatusMovedPermanently, StatusFound, StatusSeeOther,
		StatusTemporaryRedirect, StatusPermanentRedirect:
		return true
	}
	return false
}

func methodOp(method string) string {
	if method == "" {
		return "Get"
	}
	return method[:1] + strings.ToLower(method[1:])
}
