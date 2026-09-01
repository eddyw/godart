package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
)

func main() {
	jarRun()
	formRun()
	proxyRun()
	proxyEnv()
}

// jarRun proves that a jar keeps a cookie of an answer and sends it again.
func jarRun() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("seat"); err == nil {
			fmt.Fprintf(w, "seat %s", c.Value)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "seat", Value: "12", Path: "/"})
		fmt.Fprint(w, "no seat")
	}))
	defer ts.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	client := &http.Client{Jar: jar}
	for i := 0; i < 2; i++ {
		res, err := client.Get(ts.URL + "/")
		if err != nil {
			fmt.Println(err)
			return
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		fmt.Println("jar:", string(body))
	}
	u, _ := url.Parse(ts.URL)
	fmt.Println("jar holds", len(jar.Cookies(u)))
}

// formRun proves that a client writes a multipart body and that a server reads
// it back.
func formRun() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			fmt.Fprint(w, "error ", err)
			return
		}
		file, head, err := r.FormFile("note")
		if err != nil {
			fmt.Fprint(w, "error ", err)
			return
		}
		text, _ := io.ReadAll(file)
		fmt.Fprintf(w, "%s %s %s %q", r.FormValue("who"), r.PostFormValue("who"),
			head.Filename, string(text))
	}))
	defer ts.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("who", "ada")
	part, _ := w.CreateFormFile("note", "hello.txt")
	part.Write([]byte("a line of text"))
	w.Close()

	res, err := http.Post(ts.URL+"/upload", w.FormDataContentType(), &body)
	if err != nil {
		fmt.Println(err)
		return
	}
	text, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("form:", string(text))

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fmt.Fprintf(w, "%s %s %s", r.Form.Get("a"), r.Form.Get("b"), r.PostForm.Get("a"))
	}))
	defer plain.Close()
	res, err = http.PostForm(plain.URL+"/x?b=2", url.Values{"a": {"1"}})
	if err != nil {
		fmt.Println(err)
		return
	}
	text, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("form:", string(text))
}

// proxyRun proves that a request rides an HTTP proxy, and that an https request
// rides a tunnel of the same proxy.
func proxyRun() {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "plain %s", r.URL.Path)
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "secure %s", r.URL.Path)
	}))
	defer secure.Close()

	proxy, done := startProxy()
	defer done()

	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxy),
		TLSClientConfig: secure.Client().Transport.(*http.Transport).TLSClientConfig,
	}
	client := &http.Client{Transport: transport}

	res, err := client.Get(plain.URL + "/door")
	if err != nil {
		fmt.Println("proxy:", err)
		return
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("proxy:", string(body))

	res, err = client.Get(secure.URL + "/gate")
	if err != nil {
		fmt.Println("proxy:", err)
		return
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println("proxy:", string(body))
}

// startProxy runs an HTTP proxy that forwards a plain request and that opens a
// tunnel for a CONNECT request.
func startProxy() (*url.URL, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveProxy(conn)
		}
	}()
	target, _ := url.Parse("http://" + listener.Addr().String())
	return target, func() { listener.Close() }
}

func serveProxy(conn net.Conn) {
	head, err := readHead(conn)
	if err != nil {
		conn.Close()
		return
	}
	line := head[:strings.Index(head, "\r\n")]
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		conn.Close()
		return
	}
	if parts[0] == "CONNECT" {
		up, err := net.Dial("tcp", parts[1])
		if err != nil {
			conn.Close()
			return
		}
		io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		go func() {
			io.Copy(up, conn)
			up.Close()
		}()
		io.Copy(conn, up)
		conn.Close()
		return
	}
	target, err := url.Parse(parts[1])
	if err != nil {
		conn.Close()
		return
	}
	host := target.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	up, err := net.Dial("tcp", host)
	if err != nil {
		conn.Close()
		return
	}
	// The request line of the wire holds the whole URL, and the origin needs
	// the path alone.
	rest := head[strings.Index(head, "\r\n")+2:]
	io.WriteString(up, parts[0]+" "+target.RequestURI()+" "+parts[2]+"\r\n"+rest)
	io.Copy(conn, up)
	up.Close()
	conn.Close()
}

// readHead reads the request line and the header block of a request.
func readHead(conn net.Conn) (string, error) {
	var out []byte
	one := make([]byte, 1)
	for {
		n, err := conn.Read(one)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		out = append(out, one[0])
		if len(out) >= 4 && string(out[len(out)-4:]) == "\r\n\r\n" {
			return string(out), nil
		}
	}
}

// proxyEnv proves the reader of the environment.
func proxyEnv() {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/a", nil)
	target, err := http.ProxyFromEnvironment(req)
	fmt.Println("env:", target == nil, err == nil)
}
