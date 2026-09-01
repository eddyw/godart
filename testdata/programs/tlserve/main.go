package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
)

func main() {
	s := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "secure %s %s", r.Method, r.URL.Path)
		}))
	defer s.Close()
	// The raw dial below can end while the server still shakes hands, and
	// the server of Go then logs a handshake error at a time of its own.
	s.Config.ErrorLog = log.New(io.Discard, "", 0)
	fmt.Println(strings.HasPrefix(s.URL, "https://"))

	res, err := s.Client().Get(s.URL + "/door")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(res.StatusCode, string(body), err)

	res, err = s.Client().Get(s.URL + "/again")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(res.StatusCode, string(body))

	// A raw dial shows the protocol that ALPN took.
	conn, err := tls.Dial("tcp", strings.TrimPrefix(s.URL, "https://"),
		&tls.Config{InsecureSkipVerify: true, NextProtos: []string{"http/1.1"}})
	if err != nil {
		fmt.Println("dial:", err)
		return
	}
	fmt.Println("alpn:", conn.ConnectionState().NegotiatedProtocol)
	conn.Close()
}
