// httpcookie shows the cookie family and the request family of net/http:
// ParseCookie, ParseSetCookie, Cookie.Valid, the basic authentication, the
// deep copy of a request, and the fresh body of a redirect.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func main() {
	parse()
	valid()
	write()
	auth()
	clone()
	body()
	proto()
}

// parse reads ParseCookie and ParseSetCookie.
func parse() {
	list, err := http.ParseCookie("a=1; b=\"two words\"; c=3")
	fmt.Println("ParseCookie err:", err, "count:", len(list))
	for _, c := range list {
		fmt.Printf("  %s=%q quoted=%v\n", c.Name, c.Value, c.Quoted)
	}
	_, err = http.ParseCookie("no equals")
	fmt.Println("ParseCookie bad:", err)

	line := "sid=abc; Path=/x; Domain=example.com; Max-Age=60; HttpOnly; Secure; " +
		"Partitioned; SameSite=Strict; Expires=Wed, 21 Oct 2015 07:28:00 GMT; Extra=1"
	c, err := http.ParseSetCookie(line)
	fmt.Println("ParseSetCookie err:", err)
	fmt.Printf("  name=%s value=%q path=%q domain=%q max=%d\n",
		c.Name, c.Value, c.Path, c.Domain, c.MaxAge)
	fmt.Printf("  http=%v secure=%v part=%v same=%d raw=%q unparsed=%v\n",
		c.HttpOnly, c.Secure, c.Partitioned, c.SameSite, c.RawExpires, c.Unparsed)
	fmt.Println("  expires:", c.Expires.UTC().Format(http.TimeFormat))
	_, err = http.ParseSetCookie("")
	fmt.Println("ParseSetCookie empty:", err)
}

// valid reads Cookie.Valid.
func valid() {
	good := &http.Cookie{Name: "a", Value: "1", Path: "/", Domain: "example.com"}
	fmt.Println("valid good:", good.Valid())
	fmt.Println("valid no name:", (&http.Cookie{Value: "1"}).Valid())
	fmt.Println("valid bad path:", (&http.Cookie{Name: "a", Path: "x"}).Valid())
	fmt.Println("valid bad domain:", (&http.Cookie{Name: "a", Domain: "-x-"}).Valid())
	part := &http.Cookie{Name: "a", Partitioned: true}
	fmt.Println("valid partitioned:", part.Valid())
	// A year below 1601 is also invalid. The Time of the overlay counts
	// nanoseconds of an int64 from the year 1970, thus it names no such year,
	// and this program cannot show that rule.
	dated := &http.Cookie{Name: "a", Expires: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	fmt.Println("valid dated:", dated.Valid())
}

// write reads Cookie.String and the field Partitioned.
func write() {
	c := &http.Cookie{
		Name: "sid", Value: "a b", Path: "/", MaxAge: 60,
		HttpOnly: true, Secure: true, Partitioned: true,
		SameSite: http.SameSiteNoneMode,
	}
	fmt.Printf("String %q\n", c.String())
	fmt.Printf("plain %q\n", (&http.Cookie{Name: "p", Value: "v"}).String())
}

// auth reads BasicAuth, SetBasicAuth, and CookiesNamed.
func auth() {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		fmt.Println("auth:", err)
		return
	}
	name, secret, ok := req.BasicAuth()
	fmt.Printf("auth before %q %q %v\n", name, secret, ok)
	req.SetBasicAuth("Aladdin", "open sesame")
	fmt.Printf("auth header %q\n", req.Header.Get("Authorization"))
	name, secret, ok = req.BasicAuth()
	fmt.Printf("auth after %q %q %v\n", name, secret, ok)

	req.AddCookie(&http.Cookie{Name: "x", Value: "1"})
	req.AddCookie(&http.Cookie{Name: "y", Value: "2"})
	req.AddCookie(&http.Cookie{Name: "x", Value: "3"})
	fmt.Println("cookies:", len(req.Cookies()))
	for _, c := range req.CookiesNamed("x") {
		fmt.Printf("  x=%q\n", c.Value)
	}
	fmt.Println("named none:", len(req.CookiesNamed("z")),
		len(req.CookiesNamed("")))
	one, err := req.Cookie("y")
	fmt.Printf("cookie y=%q err=%v\n", one.Value, err)
	_, err = req.Cookie("z")
	fmt.Println("cookie z:", err, err == http.ErrNoCookie)
}

// clone reads Request.Clone.
func clone() {
	req, err := http.NewRequest("POST", "http://example.com/p?q=1", strings.NewReader("body"))
	if err != nil {
		fmt.Println("clone:", err)
		return
	}
	req.Header.Set("X-One", "1")
	req.Form = url.Values{"k": {"v"}}
	out := req.Clone(context.Background())
	out.Header.Set("X-One", "2")
	out.URL.Path = "/other"
	out.Form.Set("k", "w")
	fmt.Printf("clone header %q %q\n", req.Header.Get("X-One"), out.Header.Get("X-One"))
	fmt.Printf("clone path %q %q\n", req.URL.Path, out.URL.Path)
	fmt.Printf("clone form %q %q\n", req.Form.Get("k"), out.Form.Get("k"))
	fmt.Println("clone method", out.Method, "query", out.URL.RawQuery)
}

// body reads Request.GetBody.
func body() {
	req, err := http.NewRequest("POST", "http://example.com/", strings.NewReader("hello"))
	if err != nil {
		fmt.Println("body:", err)
		return
	}
	fmt.Println("length", req.ContentLength, "getBody", req.GetBody != nil)
	first, _ := io.ReadAll(req.Body)
	again, err := req.GetBody()
	if err != nil {
		fmt.Println("body get:", err)
		return
	}
	second, _ := io.ReadAll(again)
	fmt.Printf("body %q %q\n", string(first), string(second))

	empty, _ := http.NewRequest("POST", "http://example.com/", strings.NewReader(""))
	fmt.Println("empty length", empty.ContentLength, "noBody", empty.Body == http.NoBody)

	plain, _ := http.NewRequest("POST", "http://example.com/", onlyReader{})
	fmt.Println("plain length", plain.ContentLength, "getBody", plain.GetBody != nil)
}

// An onlyReader holds no length, thus NewRequest writes no GetBody.
type onlyReader struct{}

func (onlyReader) Read(p []byte) (int, error) { return 0, io.EOF }

// proto reads Request.ProtoAtLeast and Response.ProtoAtLeast.
func proto() {
	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	fmt.Println("proto", req.Proto, req.ProtoAtLeast(1, 0), req.ProtoAtLeast(1, 1),
		req.ProtoAtLeast(2, 0))
	res := &http.Response{ProtoMajor: 1, ProtoMinor: 0}
	fmt.Println("res", res.ProtoAtLeast(1, 0), res.ProtoAtLeast(1, 1))
}
