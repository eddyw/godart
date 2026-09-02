package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func main() {
	closeRun()
	tlsRun()
	testServerRun()
}

// closeRun proves that CloseClientConnections ends the connection that the
// client holds, thus the next request reaches the server on a fresh port.
func closeRun() {
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.RemoteAddr)
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()
	client := ts.Client()
	get := func() {
		res, err := client.Get(ts.URL + "/")
		if err != nil {
			fmt.Println("get:", err)
			return
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
	}
	get()
	get()
	ts.CloseClientConnections()
	get()
	if len(seen) != 3 {
		fmt.Println("the server read", len(seen), "requests")
		return
	}
	fmt.Println("reuse:", seen[0] == seen[1])
	fmt.Println("reuse after close:", seen[1] == seen[2])
}

// tlsRun proves that a start writes the config of the TLS form, and that a
// config of the caller keeps each field that it holds.
func tlsRun() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "secure ", r.URL.Path)
	})

	ts := httptest.NewUnstartedServer(handler)
	fmt.Println("tls before start:", ts.TLS == nil)
	ts.StartTLS()
	defer ts.Close()
	fmt.Println("tls after start:", len(ts.TLS.Certificates), ts.TLS.NextProtos)

	res, err := ts.Client().Get(ts.URL + "/door")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(res.StatusCode, string(body))

	kept := httptest.NewUnstartedServer(handler)
	kept.TLS = &tls.Config{NextProtos: []string{"http/1.1"}, ServerName: "example"}
	kept.StartTLS()
	defer kept.Close()
	fmt.Println("tls kept:", kept.TLS.ServerName, kept.TLS.NextProtos,
		len(kept.TLS.Certificates))
}

// testServerRun proves that a server of a test answers each host, and that a
// nil handler answers the status 500.
func testServerRun() {
	t := &testing.T{}
	ts := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "host %s path %s", r.Host, r.URL.Path)
	}))
	defer ts.Close()
	client := ts.Client()
	fmt.Println("url:", ts.URL)
	fmt.Println("no listener:", ts.Listener == nil)
	for _, target := range []string{
		"http://www.example.com/a",
		"https://go.dev/b",
		"http://10.0.0.1/c",
	} {
		res, err := client.Get(target)
		if err != nil {
			fmt.Println("get:", err)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		fmt.Println(res.StatusCode, res.Proto, string(body), res.Request.URL)
	}

	empty := httptest.NewTestServer(t, nil)
	defer empty.Close()
	res, err := empty.Client().Get("http://any.example.com/x")
	if err != nil {
		fmt.Println("nil handler:", err)
		return
	}
	res.Body.Close()
	fmt.Println("nil handler:", res.StatusCode)
}
