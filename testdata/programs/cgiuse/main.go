// cgiuse runs a CGI child behind an HTTP server, and it reads the answer.
package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const script = `#!/bin/sh
echo "Content-Type: text/plain"
echo "X-Method: $REQUEST_METHOD"
echo ""
echo "query=$QUERY_STRING"
echo "path=$PATH_INFO"
echo "extra=$MY_EXTRA"
`

func main() {
	dir, err := os.MkdirTemp("", "cgi")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)
	name := filepath.Join(dir, "hello.sh")
	if err := os.WriteFile(name, []byte(script), 0o755); err != nil {
		fmt.Println("write:", err)
		return
	}

	h := &cgi.Handler{
		Path: name,
		Root: "/cgi",
		Env:  []string{"MY_EXTRA=seven"},
	}
	mux := http.NewServeMux()
	mux.Handle("/cgi/", h)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	url := "http://" + ln.Addr().String() + "/cgi/hello.sh/deep?a=1&b=2"
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("status:", res.StatusCode)
	fmt.Println("method header:", res.Header.Get("X-Method"))
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	sort.Strings(lines)
	for _, line := range lines {
		fmt.Println(line)
	}
	srv.Close()
}
