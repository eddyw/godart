// httpmux shows the routing patterns of Go 1.22 in a ServeMux: a method, a
// wildcard, the anchor of the end, and the redirect of the trailing slash.
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "one id=%s pattern=%s", r.PathValue("id"), r.Pattern)
	})
	mux.HandleFunc("POST /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "write id=%s", r.PathValue("id"))
	})
	mux.HandleFunc("/items/{id}/parts/{part}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "part id=%s part=%s", r.PathValue("id"), r.PathValue("part"))
	})
	mux.HandleFunc("/files/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "files rest=%q", r.PathValue("rest"))
	})
	mux.HandleFunc("/exact/{$}", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "exact root")
	})
	mux.HandleFunc("/tree/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "tree "+r.URL.Path)
	})
	mux.HandleFunc("GET /items/special", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "the literal wins")
	})
	mux.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("extra", "written")
		fmt.Fprintf(w, "extra=%s empty=%q", r.PathValue("extra"), r.PathValue("gone"))
	})

	calls := []struct{ method, target string }{
		{"GET", "/items/7"},
		{"POST", "/items/7"},
		{"DELETE", "/items/7"},
		{"GET", "/items/special"},
		{"GET", "/items/7/parts/left"},
		{"GET", "/files/a/b/c.txt"},
		{"GET", "/exact/"},
		{"GET", "/exact/deeper"},
		{"GET", "/tree"},
		{"GET", "/tree/leaf"},
		{"GET", "/missing"},
		{"GET", "/set"},
		{"GET", "/items/a%2fb"},
		{"GET", "/tree/./leaf"},
	}
	for _, c := range calls {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(c.method, c.target, nil))
		res := w.Result()
		body, _ := io.ReadAll(res.Body)
		fmt.Printf("%s %s -> %d %s%s\n", c.method, c.target, res.StatusCode,
			field(res.Header, "Location"), field(res.Header, "Allow"))
		if len(body) > 0 {
			fmt.Printf("   body %q\n", strings.TrimRight(string(body), "\n"))
		}
	}

	// Handler gives the handler and the text of the pattern, and it fills no
	// wildcard.
	req := httptest.NewRequest("GET", "/items/7", nil)
	_, pattern := mux.Handler(req)
	fmt.Println("pattern", pattern, "value", req.PathValue("id") == "")

	// A pattern that conflicts with a registered pattern stops the program.
	fmt.Println("conflict:", trap(func() {
		mux.HandleFunc("/items/{other}/parts/{p}", func(http.ResponseWriter, *http.Request) {})
	}))
	fmt.Println("syntax:", trap(func() {
		mux.HandleFunc("/bad/{a...}/tail", func(http.ResponseWriter, *http.Request) {})
	}))
	fmt.Println("twice:", trap(func() {
		mux.HandleFunc("/set", func(http.ResponseWriter, *http.Request) {})
	}))

	// A host in a pattern wins over a pattern with no host.
	hosts := http.NewServeMux()
	hosts.HandleFunc("/here", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "any host")
	})
	hosts.HandleFunc("a.example/here", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "a.example")
	})
	for _, host := range []string{"a.example", "b.example", "a.example:8080"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "http://"+host+"/here", nil)
		r.Host = host
		hosts.ServeHTTP(w, r)
		body, _ := io.ReadAll(w.Result().Body)
		fmt.Printf("host %s -> %q\n", host, string(body))
	}
}

// field gives 1 field of a header in a short form, and an empty text for a
// field that the header does not hold.
func field(h http.Header, key string) string {
	if v := h.Get(key); v != "" {
		return " " + key + "=" + v
	}
	return ""
}

// trap runs the function and gives the text of the panic that it raises.
func trap(f func()) (text string) {
	defer func() {
		if p := recover(); p != nil {
			line, _, _ := strings.Cut(fmt.Sprint(p), "\n")
			// The message names the file and the line of the registration, and
			// that position differs between the 2 runs.
			text = cut(line)
		}
	}()
	f()
	return "no panic"
}

// cut drops each position of a file from a message of a conflict.
func cut(line string) string {
	var out []string
	for _, part := range strings.Split(line, " ") {
		if strings.Contains(part, ".go:") || strings.HasPrefix(part, "/") {
			part = "<position>"
		}
		out = append(out, part)
	}
	return strings.Join(out, " ")
}
