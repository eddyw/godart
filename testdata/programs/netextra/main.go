// Command netextra exercises the names that the net overlay and the net/http
// overlay took on: net.TCPConn, net.DialTCP, net.ListenConfig,
// http.Protocols, http.ResponseController, and http.Hijacker.
//
// Each line names a property, never a value that varies. A port, an address of
// a client, and a count of bytes of a header each change between 2 runs.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// payload is the text that the server writes with TCPConn.ReadFrom.
const payload = "the quick brown fox"

func main() {
	tcpConn()
	listenConfig()
	protocols()
	responseController()
}

// tcpConn proves that a dial and an accept each give a *net.TCPConn, and it
// runs the 2 copy methods of the type.
func tcpConn() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	done := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- "accept: " + err.Error()
			return
		}
		tc, ok := c.(*net.TCPConn)
		if !ok {
			c.Close()
			done <- "accept: the connection is no *net.TCPConn"
			return
		}
		n, err := tc.ReadFrom(strings.NewReader(payload))
		tc.Close()
		done <- fmt.Sprint("server ReadFrom ", n, " ", err == nil)
	}()

	raddr, err := net.ResolveTCPAddr("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("resolve:", err)
		os.Exit(1)
	}
	fmt.Println("addr network", raddr.Network(), "loopback", raddr.IP.IsLoopback())

	c, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		fmt.Println("dial tcp:", err)
		os.Exit(1)
	}
	var conn net.Conn = c
	fmt.Println("dial gives *net.TCPConn", conn != nil)
	fmt.Println("deadline", c.SetDeadline(time.Time{}) == nil)

	var buf bytes.Buffer
	m, err := c.WriteTo(&buf)
	c.Close()
	fmt.Println("client WriteTo", m, err == nil, buf.String())
	fmt.Println(<-done)

	// A plain Dial of the network tcp gives the same type.
	go func() {
		if c, err := ln.Accept(); err == nil {
			c.Close()
		}
	}()
	plain, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("dial:", err)
		os.Exit(1)
	}
	_, ok := plain.(*net.TCPConn)
	fmt.Println("Dial gives *net.TCPConn", ok)
	plain.Close()
	ln.Close()

	// A network that no TCP dial holds gives an error.
	_, err = net.DialTCP("quic", nil, raddr)
	fmt.Println("DialTCP of an unknown network fails", err != nil)
}

// listenConfig opens a listener and a datagram socket with a context.
func listenConfig() {
	var lc net.ListenConfig
	ctx := context.Background()

	ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("lc listen:", err)
		os.Exit(1)
	}
	addr := ln.Addr().(*net.TCPAddr)
	fmt.Println("lc listen", ln.Addr().Network(), addr.IP.String(), addr.Port > 0)
	go func() {
		if c, err := ln.Accept(); err == nil {
			io.WriteString(c, "listen config")
			c.Close()
		}
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("lc dial:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(c)
	c.Close()
	ln.Close()
	fmt.Println("lc read", string(body))

	pc, err := lc.ListenPacket(ctx, "udp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("lc packet:", err)
		os.Exit(1)
	}
	local := pc.LocalAddr().(*net.UDPAddr)
	fmt.Println("lc packet", pc.LocalAddr().Network(), local.IP.String(), local.Port > 0)
	n, err := pc.WriteTo([]byte("ping"), pc.LocalAddr())
	if err != nil {
		fmt.Println("lc write:", err)
		os.Exit(1)
	}
	datagram := make([]byte, 16)
	m, from, err := pc.ReadFrom(datagram)
	if err != nil {
		fmt.Println("lc read:", err)
		os.Exit(1)
	}
	pc.Close()
	fmt.Println("lc datagram", n, m, string(datagram[:m]), from.Network())

	_, err = lc.Listen(ctx, "quic", "127.0.0.1:0")
	fmt.Println("lc of an unknown network fails", err != nil)
}

// protocols runs each method of the set of HTTP protocols.
func protocols() {
	var p http.Protocols
	fmt.Println("protocols", p.String(), p.HTTP1(), p.HTTP2(), p.UnencryptedHTTP2())
	p.SetHTTP1(true)
	p.SetHTTP2(true)
	fmt.Println("protocols", p.String(), p.HTTP1(), p.HTTP2())
	p.SetUnencryptedHTTP2(true)
	fmt.Println("protocols", p.String(), p.UnencryptedHTTP2())
	// A copy of the value holds a set of its own.
	q := p
	p.SetHTTP2(false)
	fmt.Println("protocols", p.String(), q.String())
	p.SetHTTP1(false)
	p.SetUnencryptedHTTP2(false)
	fmt.Println("protocols", p.String(), p.HTTP1())
}

// bareWriter holds the 3 methods of a ResponseWriter and nothing else, thus a
// controller of it reports each other operation as not supported.
type bareWriter struct {
	http.ResponseWriter
}

// wrapWriter gives the inner writer through Unwrap, thus a controller of it
// reaches the writer of the server.
type wrapWriter struct {
	http.ResponseWriter
}

func (w wrapWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// hijackError gives the error of a hijack of a controller.
func hijackError(rc *http.ResponseController) error {
	_, _, err := rc.Hijack()
	return err
}

// responseController runs a server whose handlers drive the answer through a
// ResponseController.
func responseController() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		fmt.Fprintln(w, "read deadline", rc.SetReadDeadline(time.Time{}) == nil)
		fmt.Fprintln(w, "write deadline", rc.SetWriteDeadline(time.Time{}) == nil)
		fmt.Fprintln(w, "full duplex", rc.EnableFullDuplex() == nil)
		fmt.Fprintln(w, "flush", rc.Flush() == nil)

		bare := http.NewResponseController(bareWriter{ResponseWriter: w})
		fmt.Fprintln(w, "bare flush", errors.Is(bare.Flush(), http.ErrNotSupported))
		fmt.Fprintln(w, "bare hijack", errors.Is(hijackError(bare), http.ErrNotSupported))
		fmt.Fprintln(w, "bare deadline", errors.Is(bare.SetReadDeadline(time.Time{}), http.ErrNotSupported))
		fmt.Fprintln(w, "bare duplex", errors.Is(bare.EnableFullDuplex(), http.ErrNotSupported))

		wrapped := http.NewResponseController(wrapWriter{ResponseWriter: w})
		fmt.Fprintln(w, "wrapped flush", wrapped.Flush() == nil)
		fmt.Fprintln(w, "wrapped deadline", wrapped.SetWriteDeadline(time.Time{}) == nil)
	})
	mux.HandleFunc("/hijack", func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		conn, buf, err := rc.Hijack()
		if err != nil {
			http.Error(w, "hijack: "+err.Error(), http.StatusInternalServerError)
			return
		}
		buf.WriteString("HTTP/1.1 200 OK\r\n")
		buf.WriteString("Content-Type: text/plain\r\n")
		buf.WriteString("Content-Length: 12\r\n\r\n")
		buf.WriteString("hijacked ok!")
		buf.Flush()
		conn.Close()
	})
	server := &http.Server{Handler: mux}
	go server.Serve(ln)

	base := "http://" + ln.Addr().String()
	client := &http.Client{}
	fmt.Print(get(client, base+"/control"))
	fmt.Println("hijack body", strings.TrimSpace(get(client, base+"/hijack")))

	client.CloseIdleConnections()
	server.Close()
}

// get reads the body of 1 request.
func get(c *http.Client, target string) string {
	res, err := c.Get(target)
	if err != nil {
		return "error " + err.Error() + "\n"
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "read " + err.Error() + "\n"
	}
	return string(body)
}
