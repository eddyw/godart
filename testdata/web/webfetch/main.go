package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// target names the server of the test. The driver test writes the real
// address into this line before the build, because the web target reads no
// environment variable and no argument.
const target = "http://127.0.0.1:1"

// The web target reaches the network through the fetch API of the host. This
// program reads each part of that path: a status, a header, a body, a request
// header, a request body, and a body that needs more than 1 read.
func main() {
	get()
	post()
	big()
	missing()
}

func get() {
	res, err := http.Get(target + "/hello")
	if err != nil {
		fmt.Println("get:", err)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Println("status:", res.StatusCode)
	fmt.Println("type:", res.Header.Get("Content-Type"))
	fmt.Println("body:", string(body))
}

func post() {
	req, err := http.NewRequest(
		http.MethodPost, target+"/echo", strings.NewReader("ping"))
	if err != nil {
		fmt.Println("new:", err)
		return
	}
	req.Header.Set("X-Gart", "web")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("post:", err)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Println("echo:", string(body))
}

// big reads a body of 100000 bytes through a small buffer. The read of a
// stream gives 1 chunk at a time, thus the loop proves that the client holds
// no whole body.
func big() {
	res, err := http.Get(target + "/big")
	if err != nil {
		fmt.Println("big:", err)
		return
	}
	defer res.Body.Close()
	buf := make([]byte, 512)
	total, sum := 0, 0
	for {
		n, err := res.Body.Read(buf)
		for i := 0; i < n; i++ {
			sum += int(buf[i])
		}
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("read:", err)
			return
		}
	}
	fmt.Println("big:", total, sum)
}

func missing() {
	res, err := http.Get(target + "/none")
	if err != nil {
		fmt.Println("missing:", err)
		return
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	fmt.Println("missing:", res.StatusCode, res.Status)
}
