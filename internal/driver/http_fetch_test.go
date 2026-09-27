package driver_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestHTTPFetchRequests(t *testing.T) {
	server := func() string {
		started := make(chan struct{})
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ok":
				io.WriteString(w, "ok")
				return
			case "/failure":
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Errorf("hijack: %v", err)
					return
				}
				conn.Close()
				return
			case "/headers":
				close(started)
			case "/ready":
				select {
				case <-started:
				case <-r.Context().Done():
				}
				return
			case "/body", "/read":
				io.WriteString(w, "start")
				w.(http.Flusher).Flush()
			}
			select {
			case <-r.Context().Done():
			case <-time.After(10 * time.Second):
				t.Errorf("request %s did not cancel", r.URL.Path)
				io.WriteString(w, "end")
			}
		}))
		t.Cleanup(s.Close)
		return s.URL
	}
	headerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/method" {
			io.WriteString(w, r.Method)
			return
		}
		value := r.Header.Get("X-Bytes")
		if r.URL.Path == "/response" {
			value = "ascii \xc3\xa9 \xe4\xb8\xad \x80\xff"
		}
		w.Header().Set("X-Bytes", value)
		fmt.Fprintf(w, "%x", []byte(value))
	}))
	defer headerServer.Close()
	protocolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.Copy(w, r.Body) }))
	defer protocolServer.Close()
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fetchcancel\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func runRequests(base, kind string) {
	cause := errors.New("caller canceled")
	newRequest := func(path string) (*http.Request, func()) {
		req, _ := http.NewRequest("GET", base+path, nil)
		if kind == "context" {
			ctx, cancel := context.WithCancelCause(context.Background())
			return req.WithContext(ctx), func() { cancel(cause) }
		}
		if kind == "transport" {
			return req, func() { http.DefaultTransport.(*http.Transport).CancelRequest(req) }
		}
		cancelc := make(chan struct{})
		req.Cancel = cancelc
		return req, func() { close(cancelc) }
	}
	isCanceled := func(err error) bool {
		if kind == "context" {
			return errors.Is(err, cause)
		}
		return err != nil && strings.Contains(err.Error(), "net/http: request canceled")
	}
	if kind != "transport" {
		req, cancel := newRequest("/headers")
		result := make(chan error, 1)
		go func() {
			res, err := http.DefaultClient.Do(req)
			if res != nil {
				res.Body.Close()
			}
			result <- err
		}()
		ready, err := http.Get(base + "/ready")
		if err != nil {
			panic(err)
		}
		ready.Body.Close()
		cancel()
		fmt.Println(kind, "headers", isCanceled(<-result))
	}

	req, cancel := newRequest("/body")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	cancel()
	_, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(kind, "body", isCanceled(err))
	req, cancel = newRequest("/read")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if _, err = io.ReadFull(res.Body, make([]byte, 5)); err != nil {
		panic(err)
	}
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	_, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(kind, "read", isCanceled(err))
	req, cancel = newRequest("/ok")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(kind, "normal", string(data))
	cancel()
	_, err = res.Body.Read(make([]byte, 1))
	fmt.Println(kind, "eof", err == io.EOF)
	res.Body.Close()
	req, cancel = newRequest("/failure")
	res, err = http.DefaultClient.Do(req)
	cancel()
	if res != nil {
		res.Body.Close()
	}
	fmt.Println(kind, "failure", err != nil && !errors.Is(err, cause) && !strings.Contains(err.Error(), "request completed"))
}
func runHeaderRequests(base string) {
 for i, value := range []string{"ascii", "café", "中", "\x80\xff"} {
  req,_ := http.NewRequest("GET",base+"/echo",nil)
  req.Header.Set("X-Bytes",value)
  res,err := http.DefaultClient.Do(req)
  if err != nil {fmt.Println("headers request",i,"error",err);continue}
  body,err := io.ReadAll(res.Body)
  res.Body.Close()
  fmt.Println("headers request",i,string(body)==fmt.Sprintf("%x",[]byte(value)),res.Header.Get("X-Bytes")==value,err==nil)
 }
 res,err:=http.Get(base+"/response")
 if err!=nil {panic(err)}
 body,err:=io.ReadAll(res.Body)
 res.Body.Close()
 want:="ascii \xc3\xa9 \xe4\xb8\xad \x80\xff"
 fmt.Println("headers response",string(body)==fmt.Sprintf("%x",[]byte(want)),res.Header.Get("X-Bytes")==want,err==nil)
 req,_:=http.NewRequest("GET",base+"/method",nil)
 req.Method=""
 res,err=http.DefaultTransport.RoundTrip(req)
 if err!=nil {panic(err)}
 body,err=io.ReadAll(res.Body)
 res.Body.Close()
 fmt.Println("empty method",string(body)=="GET",req.Method=="",res.Request==req,err==nil)
}
type protocolFunc func(*http.Request) (*http.Response, error)
func (f protocolFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func runProtocolFallback(base string) {
 for _, mode := range []string{"unread", "read", "closed", "unrewindable"} {
  tr := &http.Transport{}
  seen := ""
  tr.RegisterProtocol("http", protocolFunc(func(req *http.Request) (*http.Response, error) {
   if mode != "unread" {
    data, err := io.ReadAll(req.Body)
    if err != nil { return nil, err }
    seen = string(data)
   }
   if mode == "closed" { req.Body.Close() }
   return nil, http.ErrSkipAltProtocol
  }))
  req, err := http.NewRequest("POST", base, strings.NewReader("body"))
  if err != nil { panic(err) }
  if mode == "unrewindable" { req.GetBody = nil }
  res, err := tr.RoundTrip(req)
  if mode == "unrewindable" {
   fmt.Println("protocol", mode, seen=="body", err!=nil && strings.Contains(err.Error(),"cannot rewind body"), res==nil)
  } else {
   if err != nil { panic(err) }
   data, err := io.ReadAll(res.Body)
   res.Body.Close()
   if err != nil { panic(err) }
   wantSeen := "body"
   if mode == "unread" { wantSeen = "" }
   fmt.Println("protocol", mode, seen==wantSeen, string(data)=="body", res.Request==req)
  }
  tr.CloseIdleConnections()
 }
}
func runDialErrors(base string) {
 cause := errors.New("custom dial error")
 for _, kind := range []string{"dial", "context", "tls", "tls-context"} {
  tr := &http.Transport{}
  called := false
  dial := func(string, string) (net.Conn, error) { called = true; return nil, cause }
  dialContext := func(context.Context, string, string) (net.Conn, error) { called = true; return nil, cause }
  switch kind {
  case "dial": tr.Dial = dial
  case "context": tr.DialContext = dialContext
  case "tls": tr.DialTLS = dial
  case "tls-context": tr.DialTLSContext = dialContext
  }
  req, _ := http.NewRequest("GET", base, nil)
  if strings.HasPrefix(kind, "tls") { req.URL.Scheme = "https" }
  res, err := tr.RoundTrip(req)
  if res != nil { res.Body.Close() }
  fmt.Println("dial", kind, called, errors.Is(err, cause), res == nil)
  tr.CloseIdleConnections()
 }
}
func main() { runRequests(os.Args[1], "context"); runRequests(os.Args[2], "channel"); runRequests(os.Args[3], "transport"); runHeaderRequests(os.Args[4]); runProtocolFallback(os.Args[5]); runDialErrors(os.Args[5]) }
`)
	want := runProgram(t, dir, "go", "run", ".", server(), server(), server(), headerServer.URL, protocolServer.URL)
	const expected = `context headers true
context body true
context read true
context normal ok
context eof true
context failure true
channel headers true
channel body true
channel read true
channel normal ok
channel eof true
channel failure true
transport body true
transport read true
transport normal ok
transport eof true
transport failure true
headers request 0 true true true
headers request 1 true true true
headers request 2 true true true
headers request 3 true true true
headers response true true true
empty method true true true true
protocol unread true true true
protocol read true true true
protocol closed true true true
protocol unrewindable true true true
dial dial true true true
dial context true true true
dial tls true true true
dial tls-context true true true`
	if want != expected {
		t.Fatalf("native fetch result: %q", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), server(), server(), server(), headerServer.URL, protocolServer.URL)
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{server(), server(), server(), headerServer.URL, protocolServer.URL})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("native: %s\nGart: %s\nexit: %d", want, got.Text, got.Exit)
			}
		})
	}
}
