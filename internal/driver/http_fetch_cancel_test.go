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
	"time"
)

func main() {
	base := os.Args[1]
	cause := errors.New("caller canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/headers", nil)
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
	cancel(cause)
	fmt.Println("headers", errors.Is(<-result, cause))
	ctx, cancel = context.WithCancelCause(context.Background())
	req, _ = http.NewRequestWithContext(ctx, "GET", base+"/body", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	cancel(cause)
	_, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("body", errors.Is(err, cause))
	ctx, cancel = context.WithCancelCause(context.Background())
	req, _ = http.NewRequestWithContext(ctx, "GET", base+"/read", nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if _, err = io.ReadFull(res.Body, make([]byte, 5)); err != nil {
		panic(err)
	}
	go func() { time.Sleep(20 * time.Millisecond); cancel(cause) }()
	_, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("read", errors.Is(err, cause))
	ctx, cancel = context.WithCancelCause(context.Background())
	req, _ = http.NewRequestWithContext(ctx, "GET", base+"/ok", nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("normal", string(data))
	cancel(cause)
	_, err = res.Body.Read(make([]byte, 1))
	fmt.Println("eof", err == io.EOF)
	res.Body.Close()
}
`)
	want := runProgram(t, dir, "go", "run", ".", server())
	if want != "headers true\nbody true\nread true\nnormal ok\neof true" {
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
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"), server())
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{server()})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("native: %s\nGart: %s\nexit: %d", want, got.Text, got.Exit)
			}
		})
	}
}
