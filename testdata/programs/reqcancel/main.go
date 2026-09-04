package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func main() {
	hold := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			<-hold
			io.WriteString(w, "late")
			return
		}
		io.WriteString(w, "quick")
	}))
	defer srv.Close()

	// A request with an open Cancel channel runs to its end.
	open := make(chan struct{})
	req, _ := http.NewRequest("GET", srv.URL+"/fast", nil)
	req.Cancel = open
	res, err := http.DefaultClient.Do(req)
	fmt.Println("open channel:", err == nil)
	if err == nil {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		fmt.Println("body:", string(b))
	}

	// The close of the channel ends a request that stands open.
	stop := make(chan struct{})
	slow, _ := http.NewRequest("GET", srv.URL+"/slow", nil)
	slow.Cancel = stop
	go func() {
		time.Sleep(150 * time.Millisecond)
		close(stop)
	}()
	_, err = http.DefaultClient.Do(slow)
	fmt.Println("cancelled:", err != nil)
	close(hold)

	// A channel that is already closed ends the request at its start.
	done := make(chan struct{})
	close(done)
	third, _ := http.NewRequest("GET", srv.URL+"/fast", nil)
	third.Cancel = done
	_, err = http.DefaultClient.Do(third)
	fmt.Println("closed first:", err != nil)

	// A request of a server holds no such channel.
	sr, _ := http.NewRequest("POST", "/x", strings.NewReader("b"))
	fmt.Println("server side nil:", sr.Cancel == nil)
	var _ net.Conn
}
