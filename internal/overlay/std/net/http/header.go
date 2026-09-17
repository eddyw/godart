//go:build dart

package http

import (
	"io"
	"net/http/httptrace"
	"net/textproto"
	"sort"
	"strings"
)

// A Header holds the fields of a request or of a response. The key of each
// entry holds the canonical form, for example `Content-Type`.
type Header map[string][]string

// Add appends a value to the key.
func (h Header) Add(key, value string) {
	key = CanonicalHeaderKey(key)
	h[key] = append(h[key], value)
}

// Set replaces each value of the key with 1 value.
func (h Header) Set(key, value string) {
	h[CanonicalHeaderKey(key)] = []string{value}
}

// Get gives the first value of the key, and it gives an empty string for a key
// that the header does not hold.
func (h Header) Get(key string) string {
	if h == nil {
		return ""
	}
	v := h[CanonicalHeaderKey(key)]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// Values gives each value of the key.
func (h Header) Values(key string) []string {
	if h == nil {
		return nil
	}
	return h[CanonicalHeaderKey(key)]
}

// Del drops the key.
func (h Header) Del(key string) { delete(h, CanonicalHeaderKey(key)) }

// Clone gives a copy of the header. A nil header gives nil.
func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	out := make(Header, len(h))
	for k, v := range h {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// Write writes the header in the form of the wire. The keys stand in order,
// because a Go map holds no order.
func (h Header) Write(w io.Writer) error { return h.WriteSubset(w, nil) }

// WriteSubset writes the header in the form of the wire, and it drops each key
// that the map exclude holds with the value true. The keys stand in order,
// because a Go map holds no order.
func (h Header) WriteSubset(w io.Writer, exclude map[string]bool) error {
	return h.writeSubset(w, exclude, nil)
}

// writeSubset writes the header, and it reports each key to a client trace.
func (h Header) writeSubset(w io.Writer, exclude map[string]bool, trace *httptrace.ClientTrace) error {
	keys := make([]string, 0, len(h))
	for k := range h {
		if exclude[k] || !validHeaderKey(k) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var formatted []string
		for _, v := range h[k] {
			v = strings.ReplaceAll(strings.ReplaceAll(v, "\r", " "), "\n", " ")
			v = textproto.TrimString(v)
			if trace != nil && trace.WroteHeaderField != nil {
				formatted = append(formatted, v)
			}
			if _, err := io.WriteString(w, k+": "+v+"\r\n"); err != nil {
				return err
			}
		}
		if trace != nil && trace.WroteHeaderField != nil {
			trace.WroteHeaderField(k, formatted)
		}
	}
	return nil
}

// CanonicalHeaderKey gives the canonical form of a key: the first letter of
// each part is upper case, and each other letter is lower case. A key that
// holds a byte outside the set of a token keeps its form.
func CanonicalHeaderKey(s string) string {
	if !validHeaderKey(s) {
		return s
	}
	var b strings.Builder
	upper := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case upper && 'a' <= c && c <= 'z':
			c -= 'a' - 'A'
		case !upper && 'A' <= c && c <= 'Z':
			c += 'a' - 'A'
		}
		b.WriteByte(c)
		upper = c == '-'
	}
	return b.String()
}

func validHeaderKey(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !validTokenByte(s[i]) {
			return false
		}
	}
	return true
}

func validTokenByte(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}
