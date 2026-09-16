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
	"sync"
)

var events []string
var eventsMu sync.Mutex

func note(format string, args ...any) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	events = append(events, fmt.Sprintf(format, args...))
}

func flush(title string) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	fmt.Println("--", title)
	// Transport hooks can run concurrently, without a fixed total order.
	sort.Strings(events)
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

// maskErr replaces the port of the address inside the text of an error. A dial
// that fails names the address with its port, and the mask of the address
// alone leaves that text.
func maskErr(err error, addr string) any {
	if err == nil {
		return nil
	}
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return err
	}
	return strings.ReplaceAll(err.Error(), addr[i:], ":PORT")
}

func newTrace(written, idle chan struct{}) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			note("GetConn %s", mask(hostPort))
		},
		GotConn: func(i httptrace.GotConnInfo) {
			note("GotConn conn=%t reused=%t wasIdle=%t", i.Conn != nil, i.Reused, i.WasIdle)
		},
		PutIdleConn: func(err error) {
			note("PutIdleConn err=%v", err)
			close(idle)
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
			// The text of the error holds the port too, and the resolver of
			// the system gives 2 addresses for the name localhost. Go dials
			// the second 1 only when the first 1 fails, thus the mask covers
			// the whole text.
			note("ConnectDone %s %s err=%v", network, mask(addr), maskErr(err, addr))
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
			close(written)
		},
	}
}

// read runs 1 request with the trace and prints the body of the answer.
func read(client *http.Client, req *http.Request) {
	written := make(chan struct{})
	idle := make(chan struct{})
	ctx := httptrace.WithClientTrace(req.Context(), newTrace(written, idle))
	res, err := client.Do(req.WithContext(ctx))
	if err != nil {
		note("error %v", err)
		return
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	<-written
	<-idle
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

	// Resolve IPv4 explicitly because the test server listens only on IPv4.
	target, err := url.Parse(plain.URL)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	named := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("no IPv4 address for %s", host)
			}
			return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
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
