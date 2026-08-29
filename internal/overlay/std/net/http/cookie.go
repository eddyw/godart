//go:build dart

package http

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// SameSite names the rule of a cookie for a request of another site.
type SameSite int

// The values of SameSite.
const (
	SameSiteDefaultMode SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

// A Cookie holds 1 cookie of a request or of a response.
type Cookie struct {
	Name  string
	Value string

	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite SameSite
	// Raw holds the text of the field that a read gave.
	Raw string
}

// ErrNoCookie reports a name that the request does not hold.
var ErrNoCookie = errors.New("http: named cookie not present")

// String writes the cookie in the form of the field Set-Cookie. A cookie with
// an empty name gives an empty string.
func (c *Cookie) String() string {
	if c == nil || !validCookieName(c.Name) {
		return ""
	}
	var b strings.Builder
	b.WriteString(c.Name)
	b.WriteByte('=')
	b.WriteString(sanitizeCookieValue(c.Value))
	if len(c.Path) > 0 {
		b.WriteString("; Path=" + c.Path)
	}
	if len(c.Domain) > 0 {
		b.WriteString("; Domain=" + strings.TrimPrefix(c.Domain, "."))
	}
	if !c.Expires.IsZero() {
		b.WriteString("; Expires=" + c.Expires.UTC().Format(timeFormat))
	}
	if c.MaxAge > 0 {
		b.WriteString("; Max-Age=" + strconv.Itoa(c.MaxAge))
	} else if c.MaxAge < 0 {
		b.WriteString("; Max-Age=0")
	}
	if c.HttpOnly {
		b.WriteString("; HttpOnly")
	}
	if c.Secure {
		b.WriteString("; Secure")
	}
	switch c.SameSite {
	case SameSiteDefaultMode:
		b.WriteString("; SameSite")
	case SameSiteNoneMode:
		b.WriteString("; SameSite=None")
	case SameSiteLaxMode:
		b.WriteString("; SameSite=Lax")
	case SameSiteStrictMode:
		b.WriteString("; SameSite=Strict")
	}
	return b.String()
}

// timeFormat holds the layout of a date of HTTP.
const timeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// SetCookie adds a field Set-Cookie to the answer.
func SetCookie(w ResponseWriter, cookie *Cookie) {
	if text := cookie.String(); text != "" {
		w.Header().Add("Set-Cookie", text)
	}
}

// AddCookie adds a cookie to the field Cookie of the request.
func (r *Request) AddCookie(c *Cookie) {
	pair := c.Name + "=" + sanitizeCookieValue(c.Value)
	if old := r.Header.Get("Cookie"); old != "" {
		r.Header.Set("Cookie", old+"; "+pair)
		return
	}
	r.Header.Set("Cookie", pair)
}

// Cookies gives each cookie of the request.
func (r *Request) Cookies() []*Cookie {
	return readCookies(r.Header.Values("Cookie"), "")
}

// Cookie gives the cookie of the name, and it gives ErrNoCookie for a name that
// the request does not hold.
func (r *Request) Cookie(name string) (*Cookie, error) {
	if name == "" {
		return nil, ErrNoCookie
	}
	for _, c := range readCookies(r.Header.Values("Cookie"), name) {
		return c, nil
	}
	return nil, ErrNoCookie
}

// Cookies gives each cookie of the field Set-Cookie of the response.
func (r *Response) Cookies() []*Cookie {
	return readSetCookies(r.Header.Values("Set-Cookie"))
}

// readCookies reads the field Cookie of a request. An empty filter gives each
// cookie.
func readCookies(lines []string, filter string) []*Cookie {
	var out []*Cookie
	for _, line := range lines {
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			name, value, _ := strings.Cut(part, "=")
			name = strings.TrimSpace(name)
			if !validCookieName(name) {
				continue
			}
			if filter != "" && filter != name {
				continue
			}
			out = append(out, &Cookie{Name: name, Value: unquoteCookie(value)})
		}
	}
	return out
}

// readSetCookies reads the field Set-Cookie of a response.
func readSetCookies(lines []string) []*Cookie {
	var out []*Cookie
	for _, line := range lines {
		parts := strings.Split(line, ";")
		if len(parts) == 0 {
			continue
		}
		name, value, ok := strings.Cut(strings.TrimSpace(parts[0]), "=")
		if !ok || !validCookieName(strings.TrimSpace(name)) {
			continue
		}
		c := &Cookie{
			Name:  strings.TrimSpace(name),
			Value: unquoteCookie(value),
			Raw:   line,
		}
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key, value, _ := strings.Cut(part, "=")
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "path":
				c.Path = value
			case "domain":
				c.Domain = strings.TrimPrefix(value, ".")
			case "secure":
				c.Secure = true
			case "httponly":
				c.HttpOnly = true
			case "max-age":
				n, err := strconv.Atoi(value)
				if err == nil {
					c.MaxAge = n
					if n <= 0 {
						c.MaxAge = -1
					}
				}
			case "expires":
				if t, err := time.Parse(timeFormat, value); err == nil {
					c.Expires = t.UTC()
				}
			case "samesite":
				switch strings.ToLower(value) {
				case "lax":
					c.SameSite = SameSiteLaxMode
				case "strict":
					c.SameSite = SameSiteStrictMode
				case "none":
					c.SameSite = SameSiteNoneMode
				default:
					c.SameSite = SameSiteDefaultMode
				}
			}
		}
		out = append(out, c)
	}
	return out
}

func validCookieName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !validTokenByte(name[i]) {
			return false
		}
	}
	return true
}

// sanitizeCookieValue drops each byte that a cookie value cannot hold, and it
// puts the value in quotes when it holds a space or a comma.
func sanitizeCookieValue(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c < 0x20 || c >= 0x7f || c == '"' || c == ';' || c == '\\' {
			continue
		}
		b.WriteByte(c)
	}
	out := b.String()
	if strings.ContainsAny(out, " ,") {
		return `"` + out + `"`
	}
	return out
}

func unquoteCookie(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}
