package main

import (
	"crypto"
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

	// The end of a TLS stream must give the io.EOF of the package io itself,
	// and not a value of an equal text.
	fmt.Fprint(conn, "GET /raw HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
	if _, err := io.ReadAll(conn); err != nil {
		fmt.Println("read:", err)
		return
	}
	_, err = conn.Read(make([]byte, 1))
	fmt.Println("eof:", err == io.EOF)
	conn.Close()

	pair := s.TLS.Certificates[0]
	_, parsed := pair.PrivateKey.(crypto.Signer)
	fmt.Println("parsed key:", parsed)
	manual := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "certificate fields")
	}))
	manual.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: pair.Certificate, PrivateKey: pair.PrivateKey}}}
	manual.StartTLS()
	defer manual.Close()
	res, err = manual.Client().Get(manual.URL)
	if err != nil {
		panic(err)
	}
	body, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("manual:", string(body), err)
}
