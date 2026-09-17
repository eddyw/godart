//go:build dart

package http

import (
	"log"
	"net/http/internal"
	"strings"
	"time"
)

// The helpers that the ported fs.go reads. The real net/http holds each one in
// another file of the package.

// TimeFormat is the time format of an HTTP header. It writes the time in UTC.
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// The 2 time formats that a client of an old server may write.
var timeFormats = []string{
	TimeFormat,
	time.RFC850,
	time.ANSIC,
}

// ParseTime reads the value of a header that holds a time.
func ParseTime(text string) (time.Time, error) {
	var err error
	for _, layout := range timeFormats {
		var t time.Time
		t, err = time.Parse(layout, text)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

// DetectContentType reads the first 512 bytes and gives the MIME type of them.
// A form that no rule names gives application/octet-stream.
func DetectContentType(data []byte) string {
	return internal.DetectContentType(data)
}

// get gives the first value of a key that already holds its canonical form.
func (h Header) get(key string) string {
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// has reports whether the header holds the key.
func (h Header) has(key string) bool {
	_, ok := h[key]
	return ok
}

// htmlReplacer escapes the 5 characters of HTML that a directory listing must
// not write raw.
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&#34;",
	"'", "&#39;",
)

// logf writes a message of the server. The context of the request names the
// Server, thus the line reaches the field ErrorLog of that server.
func logf(r *Request, format string, v ...any) {
	s, _ := r.Context().Value(ServerContextKey).(*Server)
	if s != nil {
		s.logf(format, v...)
		return
	}
	log.Printf(format, v...)
}
