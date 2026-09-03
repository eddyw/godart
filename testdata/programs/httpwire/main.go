// httpwire shows the wire helpers of net/http: the read of a response, the
// read of a version, the write of a subset of a header, and the write of a
// request and of a response.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	version()
	readResponse()
	writeSubset()
	writeRequest()
	writeResponse()
	location()
}

// version reads ParseHTTPVersion.
func version() {
	for _, text := range []string{"HTTP/1.1", "HTTP/1.0", "HTTP/2.0", "HTTP/2", "x", "HTTP/1.x"} {
		major, minor, ok := http.ParseHTTPVersion(text)
		fmt.Printf("version %-8q %d %d %v\n", text, major, minor, ok)
	}
}

// readResponse reads ReadResponse.
func readResponse() {
	wire := "HTTP/1.1 201 Created\r\n" +
		"Content-Type: text/plain\r\n" +
		"X-One: a\r\n" +
		"X-One: b\r\n" +
		"Content-Length: 5\r\n" +
		"\r\n" +
		"hello"
	res, err := http.ReadResponse(bufio.NewReader(strings.NewReader(wire)), nil)
	if err != nil {
		fmt.Println("readResponse:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Printf("read %d %q proto=%s/%d.%d len=%d close=%v\n", res.StatusCode,
		res.Status, res.Proto, res.ProtoMajor, res.ProtoMinor, res.ContentLength, res.Close)
	fmt.Printf("read body=%q one=%v type=%q\n", string(body),
		res.Header.Values("X-One"), res.Header.Get("Content-Type"))

	chunked := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n" +
		"5\r\nhello\r\n3\r\n hi\r\n0\r\n\r\n"
	res2, err := http.ReadResponse(bufio.NewReader(strings.NewReader(chunked)), nil)
	if err != nil {
		fmt.Println("readResponse chunked:", err)
		return
	}
	body2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	fmt.Printf("chunked %q len=%d close=%v connection=%q\n", string(body2),
		res2.ContentLength, res2.Close, res2.Header.Get("Connection"))
}

// writeSubset reads Header.WriteSubset.
func writeSubset() {
	h := http.Header{}
	h.Set("A", "1")
	h.Set("B", "2")
	h.Add("C", "3")
	h.Add("C", "4")
	var all, part bytes.Buffer
	h.Write(&all)
	h.WriteSubset(&part, map[string]bool{"B": true})
	fmt.Printf("write %q\n", all.String())
	fmt.Printf("subset %q\n", part.String())
}

// writeRequest reads Request.Write and Request.WriteProxy.
func writeRequest() {
	req, err := http.NewRequest("POST", "http://example.com/p?q=1", strings.NewReader("body"))
	if err != nil {
		fmt.Println("writeRequest:", err)
		return
	}
	req.Header.Set("X-One", "1")
	var b bytes.Buffer
	if err := req.Write(&b); err != nil {
		fmt.Println("writeRequest write:", err)
		return
	}
	fmt.Printf("request %q\n", b.String())

	req2, _ := http.NewRequest("GET", "http://example.com/p", nil)
	var b2 bytes.Buffer
	req2.WriteProxy(&b2)
	fmt.Printf("proxy %q\n", b2.String())
}

// writeResponse reads Response.Write.
func writeResponse() {
	res := &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": {"text/plain"}},
		Body:          io.NopCloser(strings.NewReader("hi")),
		ContentLength: 2,
	}
	var b bytes.Buffer
	if err := res.Write(&b); err != nil {
		fmt.Println("writeResponse:", err)
		return
	}
	fmt.Printf("response %q\n", b.String())

	res2 := &http.Response{
		StatusCode:    404,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		ContentLength: 0,
		Close:         true,
	}
	var b2 bytes.Buffer
	res2.Write(&b2)
	fmt.Printf("response 2 %q\n", b2.String())
}

// location reads Response.Location and ErrNoLocation.
func location() {
	req, _ := http.NewRequest("GET", "http://example.com/a/b", nil)
	res := &http.Response{Header: http.Header{"Location": {"../c"}}, Request: req}
	u, err := res.Location()
	fmt.Println("location", u, err)
	empty := &http.Response{Header: http.Header{}}
	_, err = empty.Location()
	fmt.Println("location none:", err, err == http.ErrNoLocation)
}
