//go:build dart

package http

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
)

// CrossOriginProtection rejects each cross-origin browser request that is not
// safe. It gives protection against Cross-Site Request Forgery, or CSRF.
//
// The check reads the header Sec-Fetch-Site, which each browser sends since
// 2023. When that header is absent, the check compares the host of the header
// Origin with the header Host.
//
// The methods GET, HEAD, and OPTIONS are safe, and the check always lets them
// through. An application must change no state for a request of a safe method.
//
// A request with no Sec-Fetch-Site header and no Origin header is same-origin,
// or it comes from no browser. The check lets such a request through.
//
// The zero value is ready for use. It holds no trusted origin and no bypass
// pattern.
type CrossOriginProtection struct {
	bypass    atomic.Pointer[ServeMux]
	trustedMu sync.RWMutex
	trusted   map[string]bool
	deny      atomic.Pointer[Handler]
}

// NewCrossOriginProtection gives a new CrossOriginProtection.
func NewCrossOriginProtection() *CrossOriginProtection {
	return &CrossOriginProtection{}
}

// AddTrustedOrigin lets through each request with an Origin header that is
// equal to the given value.
//
// An Origin header holds the form "scheme://host[:port]".
//
// A call of AddTrustedOrigin runs beside another method and beside the work of
// a request. It applies to each later request.
func (c *CrossOriginProtection) AddTrustedOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin %q: %w", origin, err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("invalid origin %q: scheme is required", origin)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid origin %q: host is required", origin)
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid origin %q: path, query, and fragment are not allowed", origin)
	}
	c.trustedMu.Lock()
	defer c.trustedMu.Unlock()
	if c.trusted == nil {
		c.trusted = make(map[string]bool)
	}
	c.trusted[origin] = true
	return nil
}

// A noopHandler answers no request. It marks a bypass pattern.
type noopHandler struct{}

// ServeHTTP writes no answer.
func (noopHandler) ServeHTTP(ResponseWriter, *Request) {}

// sentinelHandler is the handler of each bypass pattern. The check compares a
// handler of the bypass mux with this value.
var sentinelHandler Handler = &noopHandler{}

// AddInsecureBypassPattern lets through each request that matches the pattern.
//
// The syntax and the precedence of the pattern are those of ServeMux. Only a
// request that matches the pattern itself comes through. A request that
// ServeMux sends to a pattern through a redirect does not come through. Such a
// redirect follows a clean of the path or a trailing slash that ServeMux adds.
//
// AddInsecureBypassPattern stops the program when the pattern conflicts with a
// pattern that is already registered. It also stops the program when the
// syntax of the pattern is bad, such as a bad wildcard.
//
// A call of AddInsecureBypassPattern runs beside another method and beside the
// work of a request. It applies to each later request.
func (c *CrossOriginProtection) AddInsecureBypassPattern(pattern string) {
	var bypass *ServeMux

	// The field c.bypass takes its value at the first call.
	for {
		bypass = c.bypass.Load()
		if bypass != nil {
			break
		}
		bypass = NewServeMux()
		if c.bypass.CompareAndSwap(nil, bypass) {
			break
		}
	}

	bypass.Handle(pattern, sentinelHandler)
}

// SetDenyHandler writes the handler that answers a rejected request. The
// default answer holds the status 403 Forbidden.
//
// A call of SetDenyHandler runs beside another method and beside the work of a
// request. It applies to each later request.
//
// Check calls no deny handler.
func (c *CrossOriginProtection) SetDenyHandler(h Handler) {
	if h == nil {
		c.deny.Store(nil)
		return
	}
	c.deny.Store(&h)
}

// Check applies the cross-origin checks to a request. It gives an error when
// the caller must reject the request.
func (c *CrossOriginProtection) Check(req *Request) error {
	switch req.Method {
	case "GET", "HEAD", "OPTIONS":
		// A safe method always comes through.
		return nil
	}

	switch req.Header.Get("Sec-Fetch-Site") {
	case "":
		// The request holds no Sec-Fetch-Site header. The check of the Origin
		// header follows.
	case "same-origin", "none":
		return nil
	default:
		if c.isRequestExempt(req) {
			return nil
		}
		return errCrossOriginRequest
	}

	origin := req.Header.Get("Origin")
	if origin == "" {
		// The request holds no Sec-Fetch-Site header and no Origin header.
		// The request is same-origin, or it comes from no browser.
		return nil
	}

	if o, err := url.Parse(origin); err == nil && o.Host == req.Host {
		// The Origin header matches the Host header. The Host header holds no
		// scheme, thus this test does not see a cross-origin request from HTTP
		// to HTTPS. The check lets the request through, because each modern
		// browser sends Sec-Fetch-Site since 2023. An old browser is a clear
		// trade of security already. A site limits this hole with HTTP Strict
		// Transport Security, or HSTS.
		return nil
	}

	if c.isRequestExempt(req) {
		return nil
	}
	return errCrossOriginRequestFromOldBrowser
}

var (
	errCrossOriginRequest               = errors.New("cross-origin request detected from Sec-Fetch-Site header")
	errCrossOriginRequestFromOldBrowser = errors.New("cross-origin request detected, and/or browser is out of date: " +
		"Sec-Fetch-Site is missing, and Origin does not match Host")
)

// isRequestExempt reads the bypasses that need a lock. The caller holds this
// check to the last moment.
func (c *CrossOriginProtection) isRequestExempt(req *Request) bool {
	if bypass := c.bypass.Load(); bypass != nil {
		if h, _ := bypass.Handler(req); h == sentinelHandler {
			// The request matches a bypass pattern.
			return true
		}
	}

	c.trustedMu.RLock()
	defer c.trustedMu.RUnlock()
	origin := req.Header.Get("Origin")
	// The request matches a trusted origin.
	return origin != "" && c.trusted[origin]
}

// Handler gives a handler that applies the cross-origin checks before it calls
// the handler h.
//
// A request that fails a check takes the status 403 Forbidden, or it takes the
// handler of CrossOriginProtection.SetDenyHandler.
func (c *CrossOriginProtection) Handler(h Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		if err := c.Check(r); err != nil {
			if deny := c.deny.Load(); deny != nil {
				(*deny).ServeHTTP(w, r)
				return
			}
			Error(w, err.Error(), StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}
