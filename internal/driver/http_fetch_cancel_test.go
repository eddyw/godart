package driver_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestHTTPFetchContextCancellation(t *testing.T) {
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
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fetchcancel\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	req, cancel = newRequest("/body")
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
func main() { runRequests(os.Args[1], "context"); runRequests(os.Args[2], "channel") }
`)
	want := runProgram(t, dir, "go", "run", ".", server(), server())
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
channel failure true`
	if want != expected {
		t.Fatalf("native cancellation result: %q", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), server(), server())
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{server(), server()})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("native: %s\nGart: %s\nexit: %d", want, got.Text, got.Exit)
			}
		})
	}
}
