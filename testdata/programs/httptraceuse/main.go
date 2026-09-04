// Command httptraceuse proves that each hook of a client trace fires.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"sort"
	"strings"
)

// events holds each hook call in the order that the transport made it.
var events []string

func note(format string, args ...any) {
	events = append(events, fmt.Sprintf(format, args...))
}

func flush(title string) {
	fmt.Println("--", title)
	for _, line := range events {
		fmt.Println("   " + line)
	}
	events = nil
}

// mask replaces the port of an address, because the system gives a free port
// that differs at each run.
func mask(addr string) string {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return addr
	}
	return addr[:i] + ":PORT"
}

func newTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			note("GetConn %s", mask(hostPort))
		},
		GotConn: func(i httptrace.GotConnInfo) {
			note("GotConn conn=%t reused=%t wasIdle=%t", i.Conn != nil, i.Reused, i.WasIdle)
		},
		PutIdleConn: func(err error) {
			note("PutIdleConn err=%v", err)
		},
		GotFirstResponseByte: func() {
			note("GotFirstResponseByte")
		},
		Got100Continue: func() {
			note("Got100Continue")
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			keys := make([]string, 0, len(header))
			for k := range header {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			note("Got1xxResponse %d fields=%v", code, keys)
			return nil
		},
		DNSStart: func(i httptrace.DNSStartInfo) {
			note("DNSStart %s", i.Host)
		},
		DNSDone: func(i httptrace.DNSDoneInfo) {
			note("DNSDone found=%t coalesced=%t err=%v", len(i.Addrs) > 0, i.Coalesced, i.Err)
		},
		ConnectStart: func(network, addr string) {
			note("ConnectStart %s %s", network, mask(addr))
		},
		ConnectDone: func(network, addr string, err error) {
			note("ConnectDone %s %s err=%v", network, mask(addr), err)
		},
		TLSHandshakeStart: func() {
			note("TLSHandshakeStart")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			note("TLSHandshakeDone complete=%t err=%v", state.HandshakeComplete, err)
		},
		WroteHeaderField: func(key string, value []string) {
			if key == "Host" && len(value) == 1 {
				note("WroteHeaderField %s=[%s]", key, mask(value[0]))
				return
			}
			note("WroteHeaderField %s=%v", key, value)
		},
		WroteHeaders: func() {
			note("WroteHeaders")
		},
		Wait100Continue: func() {
			note("Wait100Continue")
		},
		WroteRequest: func(i httptrace.WroteRequestInfo) {
			note("WroteRequest err=%v", i.Err)
		},
	}
}

// read runs 1 request with the trace and prints the body of the answer.
func read(client *http.Client, req *http.Request) {
	ctx := httptrace.WithClientTrace(req.Context(), newTrace())
	res, err := client.Do(req.WithContext(ctx))
	if err != nil {
		note("error %v", err)
		return
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		note("error %v", err)
		return
	}
	note("status=%d body=%q", res.StatusCode, string(body))
}

func handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s %s %d", r.Method, r.URL.Path, len(body))
}

func main() {
	plain := httptest.NewServer(http.HandlerFunc(handler))
	defer plain.Close()

	client := &http.Client{Transport: &http.Transport{
		ExpectContinueTimeout: 2 * 1000 * 1000 * 1000,
	}}

	req, _ := http.NewRequest("GET", plain.URL+"/one", nil)
	read(client, req)
	flush("a fresh connection")

	req, _ = http.NewRequest("GET", plain.URL+"/two", nil)
	read(client, req)
	flush("the connection of the pool")

	req, _ = http.NewRequest("POST", plain.URL+"/three", strings.NewReader("hello"))
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "text/plain")
	read(client, req)
	flush("the answer 100 of the server")

	// A dial that names a Resolver of Go reads the host name here, thus the 2
	// hooks of the lookup fire. A dial with no Resolver gives the name to the
	// socket, which reports no event of the lookup.
	target, err := url.Parse(plain.URL)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	named := &http.Client{Transport: &http.Transport{
		DialContext: (&net.Dialer{Resolver: net.DefaultResolver}).DialContext,
	}}
	req, _ = http.NewRequest("GET", "http://localhost:"+target.Port()+"/five", nil)
	read(named, req)
	flush("the lookup of a host name")

	secure := httptest.NewTLSServer(http.HandlerFunc(handler))
	defer secure.Close()
	tlsClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	req, _ = http.NewRequest("GET", secure.URL+"/four", nil)
	read(tlsClient, req)
	flush("the handshake of TLS")

	fmt.Println("-- the trace of a context")
	base := context.Background()
	fmt.Println("   none:", httptrace.ContextClientTrace(base) == nil)
	first := &httptrace.ClientTrace{WroteHeaders: func() { note("first") }}
	second := &httptrace.ClientTrace{WroteHeaders: func() { note("second") }}
	ctx := httptrace.WithClientTrace(base, first)
	ctx = httptrace.WithClientTrace(ctx, second)
	found := httptrace.ContextClientTrace(ctx)
	fmt.Println("   found:", found == second)
	found.WroteHeaders()
	fmt.Println("   order:", events)
	events = nil
}
