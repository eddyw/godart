//go:build dart

package multipart

import (
	"errors"
	"net/textproto"
	"strings"
)

// readMIMEHeader reads the header of 1 part.
//
// Go reaches the function of the same name inside net/textproto with a
// go:linkname directive, and the subset holds no such directive. This copy
// reads the lines through the public API of net/textproto, and it holds the 2
// limits of Go.
func readMIMEHeader(r *textproto.Reader, maxMemory, maxHeaders int64) (textproto.MIMEHeader, error) {
	header := textproto.MIMEHeader{}
	for {
		line, err := r.ReadContinuedLine()
		if err != nil {
			return header, err
		}
		if line == "" {
			return header, nil
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return header, errors.New("malformed MIME header line: " + line)
		}
		maxHeaders--
		if maxHeaders < 0 {
			return header, errors.New("message too large")
		}
		// Go counts the key, the value, and the 2 words of the slice header of
		// each value.
		maxMemory -= int64(len(line)) + 40
		if maxMemory < 0 {
			return header, errors.New("message too large")
		}
		key = textproto.CanonicalMIMEHeaderKey(strings.TrimRight(key, " \t"))
		if key == "" {
			return header, errors.New("malformed MIME header line: " + line)
		}
		header[key] = append(header[key], strings.TrimLeft(value, " \t"))
	}
}
