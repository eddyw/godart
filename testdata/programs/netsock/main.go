// Command netsock exercises DialUDP, ListenUDP, DialUnix, and the 2 option
// fields Deadline and Cancel of net.Dialer.
//
// A port and a temporary path each change between 2 runs, thus each line
// below names a property alone.
package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stop writes a failure and ends the program.
func stop(what string, err error) {
	if err != nil {
		fmt.Println(what+":", err)
		os.Exit(1)
	}
}

// timedOut reports whether an error came from a deadline.
func timedOut(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

// datagram runs a socket of a listen against a socket of a dial.
func datagram() {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	stop("listen udp", err)
	defer server.Close()

	local, ok := server.LocalAddr().(*net.UDPAddr)
	fmt.Println("local addr:", ok, local.Port > 0, local.Network())
	fmt.Println("server remote:", server.RemoteAddr() == nil)

	client, err := net.DialUDP("udp", nil, local)
	stop("dial udp", err)
	defer client.Close()

	remote, ok := client.RemoteAddr().(*net.UDPAddr)
	fmt.Println("client remote:", ok, remote.Port == local.Port)

	n, err := client.Write([]byte("ping"))
	fmt.Println("client write:", n, err == nil)

	buf := make([]byte, 32)
	n, from, err := server.ReadFromUDP(buf)
	stop("read from udp", err)
	fmt.Println("server read:", string(buf[:n]), from.IP.IsLoopback(), from.Port > 0)

	n, err = server.WriteToUDP([]byte("pong"), from)
	fmt.Println("server write:", n, err == nil)
	n, err = client.Read(buf)
	stop("client read", err)
	fmt.Println("client read:", string(buf[:n]))

	// A socket of a dial rejects a WriteTo.
	_, err = client.WriteTo([]byte("x"), from)
	fmt.Println("write to connected:", errors.Is(err, net.ErrWriteToConnected))

	// A deadline that passed ends a read.
	stop("deadline", server.SetReadDeadline(time.Now().Add(20*time.Millisecond)))
	_, _, err = server.ReadFromUDP(buf)
	fmt.Println("read deadline:", timedOut(err))
}

// addrPort runs the 4 conversions of net/netip and the 2 methods that read
// such an address.
func addrPort() {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	stop("listen udp", err)
	defer server.Close()
	local := server.LocalAddr().(*net.UDPAddr)

	ap := local.AddrPort()
	fmt.Println("addr port:", ap.Addr().Unmap().Is4(), ap.Port() == uint16(local.Port))
	back := net.UDPAddrFromAddrPort(ap)
	fmt.Println("udp round trip:", back.IP.Equal(local.IP), back.Port == local.Port)

	tcp := &net.TCPAddr{IP: net.ParseIP("10.1.2.3"), Port: 8080}
	tap := tcp.AddrPort()
	fmt.Println("tcp addr port:", tap.Addr().Unmap().String(), tap.Port())
	fmt.Println("tcp round trip:", net.TCPAddrFromAddrPort(tap).String())

	var noAddr *net.UDPAddr
	fmt.Println("nil addr port:", noAddr.AddrPort().IsValid())

	client, err := net.DialUDP("udp", nil, local)
	stop("dial udp", err)
	defer client.Close()

	n, err := client.WriteToUDPAddrPort([]byte("x"), ap)
	fmt.Println("write to connected addr port:", n, errors.Is(err, net.ErrWriteToConnected))

	n, err = client.Write([]byte("hello"))
	fmt.Println("write:", n, err == nil)
	buf := make([]byte, 32)
	n, source, err := server.ReadFromUDPAddrPort(buf)
	stop("read from addr port", err)
	fmt.Println("read addr port:", string(buf[:n]), source.Addr().IsLoopback(), source.Port() > 0)

	n, err = server.WriteToUDPAddrPort([]byte("back"), source)
	fmt.Println("write to addr port:", n, err == nil)
	n, err = client.Read(buf)
	stop("client read", err)
	fmt.Println("client read:", string(buf[:n]))
}

// unixSocket runs a listener and a dial of a Unix domain socket.
func unixSocket() {
	dir, err := os.MkdirTemp("", "gu")
	stop("temp dir", err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "s")

	ln, err := net.Listen("unix", path)
	stop("listen unix", err)
	defer ln.Close()
	fmt.Println("listener addr:", ln.Addr().Network(), ln.Addr().String() == path)

	report := make(chan string, 1)
	go func() { report <- serve(ln) }()

	c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	stop("dial unix", err)
	fmt.Println("dial network:", c.RemoteAddr().Network())
	n, err := c.Write([]byte("hi"))
	fmt.Println("client write:", n, err == nil)
	buf := make([]byte, 32)
	n, err = c.Read(buf)
	stop("client read", err)
	fmt.Println("client read:", string(buf[:n]))
	fmt.Println("close:", c.Close() == nil)
	fmt.Println("server:", <-report)
}

// serve accepts 1 connection of a Unix listener and writes an answer. It
// gives a report, because only main writes to standard output.
func serve(ln net.Listener) string {
	c, err := ln.Accept()
	if err != nil {
		return "accept: " + err.Error()
	}
	uc, ok := c.(*net.UnixConn)
	if !ok {
		c.Close()
		return "accept: the connection is no *net.UnixConn"
	}
	defer uc.Close()
	buf := make([]byte, 16)
	n, err := uc.Read(buf)
	if err != nil {
		return "read: " + err.Error()
	}
	if _, err := uc.Write([]byte("echo " + string(buf[:n]))); err != nil {
		return "write: " + err.Error()
	}
	return "ok"
}

// dialerOptions runs the 2 option fields of the dialer.
//
// A Cancel channel that closes before the dial gives a race in Go: the dial
// of a local listener can end before the watcher of the channel runs. Thus
// the test below holds a channel that never closes. It proves that the field
// ends no dial of its own.
func dialerOptions() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	stop("listen tcp", err)
	defer ln.Close()
	target := ln.Addr().String()

	open, err := (&net.Dialer{Deadline: time.Now().Add(5 * time.Second)}).Dial("tcp", target)
	fmt.Println("deadline ahead:", err == nil, open != nil)
	if open != nil {
		open.Close()
	}

	_, err = (&net.Dialer{Deadline: time.Now().Add(-time.Second)}).Dial("tcp", target)
	fmt.Println("deadline behind:", err != nil, timedOut(err))
	fmt.Println("deadline text:", err != nil && strings.HasSuffix(err.Error(), "i/o timeout"))

	never := make(chan struct{})
	quiet, err := (&net.Dialer{Cancel: never}).Dial("tcp", target)
	fmt.Println("cancel open:", err == nil, quiet != nil)
	if quiet != nil {
		quiet.Close()
	}
}

func main() {
	datagram()
	addrPort()
	unixSocket()
	dialerOptions()
}
