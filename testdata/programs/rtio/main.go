// Command rtio moves a large block of bytes through each channel of the input
// and output runtime: a file, a TCP connection, a UDP packet, a child process,
// an HTTP/2 stream, and a base64 field of encoding/json.
//
// The runtime moved 1 byte at a time on each of those paths. A block copy
// gives the same bytes, thus the sum below holds for both forms. The pattern
// holds each value of a byte, thus a copy that loses the high bit fails.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
)

// pattern gives n bytes that hold each value of a byte in turn.
func pattern(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i % 251)
	}
	return out
}

// sum gives the FNV-1a hash of the bytes, as text.
func sum(b []byte) string {
	h := uint32(2166136261)
	for _, c := range b {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}

func report(name string, want, got []byte) {
	fmt.Printf("%s %d %s %v\n", name, len(got), sum(got), bytes.Equal(want, got))
}

func fileCase(dir string, data []byte) {
	path := filepath.Join(dir, "block.bin")
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("file", err)
		return
	}
	if _, err := f.Write(data); err != nil {
		fmt.Println("file write", err)
		return
	}
	f.Close()

	f, err = os.Open(path)
	if err != nil {
		fmt.Println("file open", err)
		return
	}
	got := make([]byte, len(data))
	if _, err := io.ReadFull(f, got); err != nil {
		fmt.Println("file read", err)
		return
	}
	f.Close()
	report("file", data, got)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Println("writefile", err)
		return
	}
	whole, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("readfile", err)
		return
	}
	report("wholefile", data, whole)
}

func tcpCase(data []byte) {
	list, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("tcp listen", err)
		return
	}
	defer list.Close()
	go func() {
		c, err := list.Accept()
		if err != nil {
			return
		}
		buf := make([]byte, len(data))
		if _, err := io.ReadFull(c, buf); err == nil {
			c.Write(buf)
		}
		c.Close()
	}()
	conn, err := net.Dial("tcp", list.Addr().String())
	if err != nil {
		fmt.Println("tcp dial", err)
		return
	}
	go func() { conn.Write(data) }()
	got := make([]byte, len(data))
	if _, err := io.ReadFull(conn, got); err != nil {
		fmt.Println("tcp read", err)
		return
	}
	conn.Close()
	report("tcp", data, got)
}

func udpCase(data []byte) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("udp listen", err)
		return
	}
	defer server.Close()
	client, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("udp client", err)
		return
	}
	defer client.Close()
	if _, err := client.WriteTo(data, server.LocalAddr()); err != nil {
		fmt.Println("udp write", err)
		return
	}
	got := make([]byte, len(data)+16)
	n, _, err := server.ReadFrom(got)
	if err != nil {
		fmt.Println("udp read", err)
		return
	}
	report("udp", data, got[:n])
}

func processCase(data []byte) {
	cat := exec.Command("cat")
	cat.Stdin = bytes.NewReader(data)
	var out bytes.Buffer
	cat.Stdout = &out
	if err := cat.Run(); err != nil {
		fmt.Println("process", err)
		return
	}
	report("process", data, out.Bytes())
}

func http2Case(data []byte) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	}))
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()
	res, err := ts.Client().Post(ts.URL+"/echo", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		fmt.Println("http2", err)
		return
	}
	got, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("proto", res.Proto)
	report("http2", data, got)
}

func jsonCase(data []byte) {
	type box struct {
		Data []byte `json:"data"`
	}
	text, err := json.Marshal(box{Data: data})
	if err != nil {
		fmt.Println("json", err)
		return
	}
	var back box
	if err := json.Unmarshal(text, &back); err != nil {
		fmt.Println("json back", err)
		return
	}
	fmt.Println("jsonlen", len(text))
	report("json", data, back.Data)
}

func main() {
	dir, err := os.MkdirTemp("", "rtio")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)

	big := pattern(1 << 20)
	half := pattern(1 << 19)
	small := pattern(8000)
	mid := pattern(1 << 17)

	fileCase(dir, big)
	tcpCase(half)
	udpCase(small)
	processCase(half)
	http2Case(half)
	jsonCase(mid)
}
