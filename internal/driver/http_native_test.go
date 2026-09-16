package driver_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestHTTPNativeServer(t *testing.T) {
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write([]byte("native server body")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Accept-Encoding", r.Header.Get("Accept-Encoding"))
		w.Header().Set("X-Protocol", strconv.Itoa(r.ProtoMajor))
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/body", http.StatusFound)
			return
		}
		body := []byte("native server body")
		if r.Header.Get("Accept-Encoding") == "gzip" {
			body = compressed.Bytes()
			w.Header().Set("Content-Encoding", "gzip")
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		if r.URL.Path == "/short" {
			body = body[:len(body)-4]
		}
		w.Write(body)
	})
	h1 := httptest.NewServer(handler)
	defer h1.Close()
	h2 := httptest.NewUnstartedServer(handler)
	h2.EnableHTTP2 = true
	h2.StartTLS()
	defer h2.Close()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/httpnative\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), fmt.Sprintf(`package main
import ("compress/gzip"; "crypto/tls"; "fmt"; "io"; "net/http")
func main() {
 for _, base := range []string{%q, %q} {
  for _, name := range []string{"automatic", "explicit", "disabled", "head", "range", "redirect", "short"} {
   tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, ForceAttemptHTTP2: true, DisableCompression: name == "disabled"}
   client := &http.Client{Transport: tr}
   path, method := "/body", "GET"
   if name == "redirect" || name == "short" { path = "/" + name }
   if name == "head" { method = "HEAD" }
   req, err := http.NewRequest(method, base + path, nil)
   if err != nil { panic(err) }
   if name == "explicit" { req.Header.Set("Accept-Encoding", "gzip") }
   if name == "range" { req.Header.Set("Range", "bytes=0-3") }
   resp, err := client.Do(req)
   if err != nil { panic(err) }
   var reader io.Reader = resp.Body
   if name == "explicit" {
    z, err := gzip.NewReader(resp.Body)
    if err != nil { panic(err) }
    defer z.Close()
    reader = z
   }
   body, readErr := io.ReadAll(reader)
   fmt.Printf("%%s protocol=%%s accept=%%q encoding=%%q uncompressed=%%t length=%%d body=%%q error=%%v\n", name, resp.Header.Get("X-Protocol"), resp.Header.Get("X-Accept-Encoding"), resp.Header.Get("Content-Encoding"), resp.Uncompressed, resp.ContentLength, body, readErr)
   if err := resp.Body.Close(); err != nil { panic(err) }
   tr.CloseIdleConnections()
  }
 }
}
`, h1.URL, h2.URL))
	want := runProgram(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	got := runProgram(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
	if got != want {
		t.Fatalf("native server response differs:\nGo:\n%s\nGart:\n%s", want, got)
	}
}
