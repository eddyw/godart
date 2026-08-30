package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelloRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/hello?name=go", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Body.String(); got != "hello go" {
		t.Errorf("body = %q, want %q", got, "hello go")
	}
	if got := w.Header().Get("X-Route"); got != "hello" {
		t.Errorf("X-Route = %q, want %q", got, "hello")
	}
}

func TestResultOfRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("abc")))
	res := w.Result()
	if res.Status != "200 OK" {
		t.Errorf("status = %q, want %q", res.Status, "200 OK")
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "POST:ABC" {
		t.Errorf("body = %q, want %q", body, "POST:ABC")
	}
	if got := res.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
}

func TestNewRequestFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/echo", strings.NewReader("hello"))
	if req.Host != "example.com" {
		t.Errorf("Host = %q, want example.com", req.Host)
	}
	if req.RemoteAddr != "192.0.2.1:1234" {
		t.Errorf("RemoteAddr = %q", req.RemoteAddr)
	}
	if req.ContentLength != 5 {
		t.Errorf("ContentLength = %d, want 5", req.ContentLength)
	}
	if req.Proto != "HTTP/1.1" {
		t.Errorf("Proto = %q", req.Proto)
	}
}

func TestStatusOfRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/teapot", nil))
	if w.Code != http.StatusTeapot {
		t.Errorf("code = %d, want %d", w.Code, http.StatusTeapot)
	}
	if got := strings.TrimSpace(w.Body.String()); got != "short and stout" {
		t.Errorf("body = %q", got)
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/empty", nil))
	res := w.Result()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("code = %d", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want empty", got)
	}
}

func TestServer(t *testing.T) {
	server := httptest.NewServer(Handler())
	defer server.Close()

	res, err := server.Client().Get(server.URL + "/hello")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q", body)
	}

	res, err = server.Client().Post(server.URL+"/echo", "text/plain", strings.NewReader("xyz"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	body, err = io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "POST:XYZ" {
		t.Errorf("body = %q", body)
	}
}

func TestUnstartedServer(t *testing.T) {
	server := httptest.NewUnstartedServer(Handler())
	if server.URL != "" {
		t.Errorf("URL = %q, want empty", server.URL)
	}
	server.Start()
	defer server.Close()
	if !strings.HasPrefix(server.URL, "http://127.0.0.1:") {
		t.Errorf("URL = %q", server.URL)
	}
	res, err := server.Client().Get(server.URL + "/teapot")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusTeapot {
		t.Errorf("code = %d", res.StatusCode)
	}
}
