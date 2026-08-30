// Package api holds a small HTTP handler.
package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Handler answers each route of the api.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", hello)
	mux.HandleFunc("/echo", echo)
	mux.HandleFunc("/teapot", teapot)
	mux.HandleFunc("/empty", empty)
	return mux
}

func hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	w.Header().Set("X-Route", "hello")
	fmt.Fprintf(w, "hello %s", name)
}

func echo(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read: "+err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "%s:%s", r.Method, strings.ToUpper(string(body)))
}

func teapot(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "short and stout", http.StatusTeapot)
}

func empty(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
