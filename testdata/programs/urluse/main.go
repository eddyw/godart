package main

import (
	"fmt"
	"net/netip"
	"net/url"
	"sort"
	"strings"
)

func parse() {
	texts := []string{
		"https://user:pw@example.com:8443/a/b?x=1&y=two#frag",
		"http://example.com",
		"/just/a/path",
		"mailto:someone@example.com",
		"//host/path",
		"https://[::1]:80/p",
		"https://example.com/a%20b?q=a%2Bb",
		"postgres://u@h/db?sslmode=disable",
		"::not a url",
	}
	for _, text := range texts {
		u, err := url.Parse(text)
		if err != nil {
			fmt.Printf("%-50s error: %v\n", text, err)
			continue
		}
		user := ""
		if u.User != nil {
			pw, set := u.User.Password()
			user = fmt.Sprintf("%s/%s/%v", u.User.Username(), pw, set)
		}
		fmt.Printf("%-50s %s | %s | %s | %s | %s | %s | %s\n", text,
			u.Scheme, u.Host, u.Hostname(), u.Port(), u.Path, u.RawQuery, user)
		fmt.Println("   round trip:", u.String())
	}
}

func query() {
	v, err := url.ParseQuery("a=1&b=two&a=3&c")
	fmt.Println(err, len(v))
	keys := []string{}
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Println(" ", k, v[k])
	}
	fmt.Println(v.Get("a"), v.Get("missing"))
	v.Set("d", "4")
	v.Add("a", "5")
	fmt.Println(v.Encode())
	v.Del("a")
	fmt.Println(v.Encode(), v.Has("a"))

	_, err = url.ParseQuery("a=%zz")
	fmt.Println("bad query:", err)
}

func escapes() {
	fmt.Println(url.QueryEscape("a b&c=d/e"))
	fmt.Println(url.PathEscape("a b&c=d/e"))
	s, err := url.QueryUnescape("a+b%26c")
	fmt.Println(s, err)
	s, err = url.PathUnescape("a%20b%2Fc")
	fmt.Println(s, err)
	_, err = url.QueryUnescape("%zz")
	fmt.Println(err)
}

func resolve() {
	base, _ := url.Parse("https://example.com/a/b/c?q=1")
	for _, ref := range []string{"d", "/e", "../f", "?y=2", "#frag", "https://other.org/z"} {
		r, err := url.Parse(ref)
		if err != nil {
			fmt.Println(ref, err)
			continue
		}
		fmt.Printf("%-22s %s\n", ref, base.ResolveReference(r))
	}
	j, err := url.JoinPath("https://example.com/a", "b", "..", "c")
	fmt.Println(j, err)
}

func addresses() {
	for _, text := range []string{"127.0.0.1", "::1", "1.2.3.4", "2001:db8::1", "nope", "::ffff:1.2.3.4"} {
		a, err := netip.ParseAddr(text)
		if err != nil {
			fmt.Printf("%-16s error\n", text)
			continue
		}
		fmt.Printf("%-16s %s %v %v %v %v\n", text, a, a.Is4(), a.Is6(), a.Is4In6(), a.IsLoopback())
	}
	p, err := netip.ParseAddrPort("127.0.0.1:8080")
	fmt.Println(p, p.Port(), p.Addr(), err)
	pre, err := netip.ParsePrefix("10.0.0.0/8")
	fmt.Println(pre, pre.Bits(), pre.Contains(netip.MustParseAddr("10.1.2.3")), err)
	fmt.Println(strings.Repeat("-", 3))
}

func main() {
	parse()
	query()
	escapes()
	resolve()
	addresses()
}
