//go:build dart

package http

import (
	"io"
	"strings"
)

// NoBody is a body with no byte. A Read of it gives io.EOF at once, and a Close
// gives nil. A request that carries NoBody names the length 0, thus the wire
// holds no body and no chunk.
//
// A caller that needs a body with no byte and a *Request that names it must use
// NoBody, because a nil Body of an outgoing request names no length.
var NoBody = noBody{}

type noBody struct{}

func (noBody) Read([]byte) (int, error)         { return 0, io.EOF }
func (noBody) Close() error                     { return nil }
func (noBody) WriteTo(io.Writer) (int64, error) { return 0, nil }

// hasToken reports whether the value holds the token. A comma divides the
// tokens, and the comparison ignores the case.
func hasToken(v, token string) bool {
	if v == "" {
		return false
	}
	for _, part := range strings.Split(v, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}
