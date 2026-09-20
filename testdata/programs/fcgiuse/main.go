package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/textproto"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func record(w io.Writer, kind byte, body []byte) {
	_, err := w.Write(append([]byte{1, kind, 0, 1, byte(len(body) >> 8), byte(len(body)), 0, 0}, body...))
	must(err)
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)
	done := make(chan error, 1)
	go func() {
		done <- fcgi.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			must(err)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "%s %s %s %s", r.Method, r.URL.Path, r.URL.RawQuery, body)
		}))
	}()
	conn, err := net.Dial("tcp", listener.Addr().String())
	must(err)
	must(conn.SetDeadline(time.Now().Add(5 * time.Second)))
	record(conn, 1, []byte{0, 1, 0, 0, 0, 0, 0, 0})
	var params []byte
	for _, pair := range [][2]string{{"REQUEST_METHOD", "POST"}, {"SERVER_PROTOCOL", "HTTP/1.1"}, {"REQUEST_URI", "/test?q=1"}, {"QUERY_STRING", "q=1"}, {"SCRIPT_NAME", "/test"}, {"CONTENT_LENGTH", "3"}} {
		params = append(params, byte(len(pair[0])), byte(len(pair[1])))
		params = append(params, pair[0]...)
		params = append(params, pair[1]...)
	}
	record(conn, 4, params)
	record(conn, 4, nil)
	record(conn, 5, []byte("abc"))
	record(conn, 5, nil)
	var stdout bytes.Buffer
	for {
		var header [8]byte
		_, err = io.ReadFull(conn, header[:])
		must(err)
		body := make([]byte, int(header[4])<<8|int(header[5]))
		_, err = io.ReadFull(conn, body)
		must(err)
		_, err = io.CopyN(io.Discard, conn, int64(header[6]))
		must(err)
		switch header[1] {
		case 6:
			stdout.Write(body)
		case 7:
			if len(body) > 0 {
				panic(string(body))
			}
		case 3:
			must(conn.Close())
			must(listener.Close())
			if <-done == nil {
				panic("Serve accepted a closed listener")
			}
			response := bufio.NewReader(&stdout)
			headers, err := textproto.NewReader(response).ReadMIMEHeader()
			must(err)
			result, err := io.ReadAll(response)
			must(err)
			fmt.Println(headers.Get("Status"), headers.Get("Content-Type"), string(result))
			if fcgi.Serve(nil, nil) == nil {
				panic("Serve accepted a non-listener stdin")
			}
			fmt.Println("nil listener rejected")
			return
		}
	}
}
