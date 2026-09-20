package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
)

func main() {
	payload := bytes.Repeat([]byte{0, 1, 2, 3}, 4096)
	for _, size := range []int{2048, 2049, len(payload)} {
		for _, declared := range []bool{false, true} {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if declared {
					w.Header().Set("Content-Length", strconv.Itoa(size))
				}
				for i := 0; i < size; i += 512 {
					end := i + 512
					if end > size {
						end = size
					}
					if _, err := w.Write(payload[i:end]); err != nil {
						panic(err)
					}
				}
			}))
			res, err := http.Get(server.URL)
			if err != nil {
				panic(err)
			}
			body, err := io.ReadAll(res.Body)
			if err != nil {
				panic(err)
			}
			res.Body.Close()
			fmt.Println(size, declared, res.ContentLength, res.TransferEncoding, res.Header.Get("Content-Type"), bytes.Equal(body, payload[:size]))
			server.Close()
		}
	}
}
