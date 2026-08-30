package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	text := strings.Repeat("compress me. ", 60)
	mux := http.NewServeMux()
	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("server sees", r.Header.Get("Accept-Encoding"))
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fmt.Fprint(w, text)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/plain")
		zw := gzip.NewWriter(w)
		io.WriteString(zw, text)
		zw.Close()
	})
	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	base := "http://" + listener.Addr().String()

	res, err := http.Get(base + "/text")
	if err != nil {
		fmt.Println("get:", err)
		os.Exit(1)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("plain", string(body) == text, err)
	fmt.Println("uncompressed", res.Uncompressed, res.ContentLength,
		res.Header.Get("Content-Encoding"), res.Header.Get("Content-Type"))

	// A client that names no compression reads the bytes of the wire.
	raw := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	res, err = raw.Get(base + "/text")
	if err != nil {
		fmt.Println("raw:", err)
		os.Exit(1)
	}
	body, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("raw", len(body) < len(text), res.Header.Get("Content-Encoding"), err)

	// A request that names its own encoding keeps it.
	req, _ := http.NewRequest(http.MethodGet, base+"/text", nil)
	req.Header.Set("Accept-Encoding", "identity")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("identity:", err)
		os.Exit(1)
	}
	body, err = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("identity", string(body) == text, res.Uncompressed, err)

	http.DefaultClient.CloseIdleConnections()
	server.Close()
}
