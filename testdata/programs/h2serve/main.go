package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

func main() {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Printf("server: %s %s %s %q\n", r.Proto, r.Method, r.URL.Path, string(body))
		w.Header().Set("X-Route", r.URL.Path)
		fmt.Fprintf(w, "answer of %s over %s", r.URL.Path, r.Proto)
	}))
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()

	client := ts.Client()

	res, err := client.Get(ts.URL + "/door")
	if err != nil {
		fmt.Println(err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("client:", res.Proto, res.StatusCode, res.Header.Get("X-Route"), string(body))

	res, err = client.Post(ts.URL+"/drop", "text/plain", strings.NewReader("a payload of the client"))
	if err != nil {
		fmt.Println(err)
		return
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("client:", res.Proto, res.StatusCode, res.Header.Get("X-Route"), string(body))

	res, err = client.Get(ts.URL + "/again")
	if err != nil {
		fmt.Println(err)
		return
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("client:", res.Proto, res.StatusCode, res.Header.Get("X-Route"), string(body))
}
