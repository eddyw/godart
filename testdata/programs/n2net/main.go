// Command n2net proves the address names, the 2 listener types, and the 3
// dial methods of a Dialer that the sixteenth session added to the package
// net.
package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
)

func main() {
	addresses()
	errorValues()
	resolve()
	tcpListener()
	unixListener()
}

// addresses prints the 6 tests of an address and each well-known address.
func addresses() {
	list := []string{
		"10.1.2.3", "172.16.0.1", "172.32.0.1", "192.168.1.1", "8.8.8.8",
		"127.0.0.1", "224.0.0.1", "169.254.1.1", "255.255.255.255", "0.0.0.0",
		"::1", "fe80::1", "ff02::1", "ff01::1", "fc00::1", "2001:db8::1", "::",
	}
	for _, text := range list {
		ip := net.ParseIP(text)
		fmt.Printf("%-16s priv=%t mult=%t iflocal=%t linkm=%t linku=%t glob=%t unspec=%t loop=%t\n",
			text, ip.IsPrivate(), ip.IsMulticast(), ip.IsInterfaceLocalMulticast(),
			ip.IsLinkLocalMulticast(), ip.IsLinkLocalUnicast(), ip.IsGlobalUnicast(),
			ip.IsUnspecified(), ip.IsLoopback())
	}
	fmt.Println("v4:", net.IPv4bcast, net.IPv4allsys, net.IPv4allrouter, net.IPv4zero)
	fmt.Println("v6:", net.IPv6zero, net.IPv6unspecified, net.IPv6loopback)
	fmt.Println("v6 groups:", net.IPv6interfacelocalallnodes,
		net.IPv6linklocalallnodes, net.IPv6linklocalallrouters)
	fmt.Println("bcast global:", net.IPv4bcast.IsGlobalUnicast())
	// A length that no address holds gives false at each test.
	odd := net.IP{0, 0, 0, 0, 0}
	fmt.Println("odd:", odd.IsUnspecified(), odd.IsGlobalUnicast(), odd.IsMulticast())
}

// errorValues prints the 2 error types that hold Timeout and Temporary.
func errorValues() {
	var bad net.Error = net.InvalidAddrError("invalid address 1.2.3.4:99999")
	fmt.Println("invalid:", bad, bad.Timeout(), bad.Temporary())

	_, _, err := net.ParseCIDR("nope")
	pe, ok := err.(*net.ParseError)
	fmt.Println("parse:", ok, pe, pe.Timeout(), pe.Temporary())
	_, isNetError := err.(net.Error)
	fmt.Println("parse is net.Error:", isNetError)
}

// resolve prints the address that ResolveIPAddr reads.
func resolve() {
	a, err := net.ResolveIPAddr("ip", "192.0.2.5")
	fmt.Println("ip4:", a, a.Network(), err)
	a, err = net.ResolveIPAddr("ip6", "2001:db8::1")
	fmt.Println("ip6:", a, err)
	a, err = net.ResolveIPAddr("", "10.0.0.1")
	fmt.Println("empty network:", a, err)
	_, err = net.ResolveIPAddr("bogus", "1.2.3.4")
	fmt.Println("bogus:", err)
}

// tcpListener proves TCPListener, ListenTCP, AcceptTCP, and Dialer.DialTCP.
func tcpListener() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	tl, ok := l.(*net.TCPListener)
	fmt.Println("listen gives *TCPListener:", ok, tl.Addr().Network())
	l.Close()

	tl, err = net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		fmt.Println("listen tcp:", err)
		return
	}
	defer tl.Close()
	done := make(chan string, 1)
	go func() {
		c, err := tl.AcceptTCP()
		if err != nil {
			done <- "accept: " + err.Error()
			return
		}
		buf := make([]byte, 5)
		n, _ := c.Read(buf)
		c.Write([]byte("pong"))
		c.Close()
		done <- string(buf[:n])
	}()

	target, err := netip.ParseAddrPort(tl.Addr().String())
	if err != nil {
		fmt.Println("addr port:", err)
		return
	}
	var d net.Dialer
	c, err := d.DialTCP(context.Background(), "tcp", netip.AddrPort{}, target)
	if err != nil {
		fmt.Println("dial tcp:", err)
		return
	}
	c.Write([]byte("ping"))
	answer := make([]byte, 4)
	n, _ := c.Read(answer)
	c.Close()
	fmt.Println("tcp:", <-done, string(answer[:n]))

	// Dart binds no local address of a dial, thus a laddr that holds a value
	// gives an error.
	_, err = d.DialTCP(context.Background(), "tcp", target, target)
	fmt.Println("local address rejected:", err != nil)
	_, err = d.DialTCP(context.Background(), "udp", netip.AddrPort{}, target)
	fmt.Println("bad network rejected:", err != nil)
}

// unixListener proves UnixListener, ListenUnix, AcceptUnix, and the 4
// datagram methods of a stream connection of the domain.
func unixListener() {
	dir, err := os.MkdirTemp("", "n2net")
	if err != nil {
		fmt.Println("temp:", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "s.sock")

	ul, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		fmt.Println("listen unix:", err)
		return
	}
	defer ul.Close()
	fmt.Println("unix listener:", ul.Addr().Network())

	done := make(chan int, 1)
	go func() {
		c, err := ul.AcceptUnix()
		if err != nil {
			done <- -1
			return
		}
		buf := make([]byte, 8)
		n, addr, _ := c.ReadFrom(buf)
		fmt.Println("readfrom:", string(buf[:n]), addr == nil)
		c.Write([]byte("pong"))
		c.Close()
		done <- n
	}()

	var d net.Dialer
	uc, err := d.DialUnix(context.Background(), "unix", nil,
		&net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		fmt.Println("dial unix:", err)
		return
	}
	uc.Write([]byte("hello"))
	reply := make([]byte, 8)
	m, unixAddr, _ := uc.ReadFromUnix(reply)
	fmt.Println("readfromunix:", string(reply[:m]), unixAddr == nil)
	fmt.Println("wrote:", <-done)
	uc.Close()

	_, err = d.DialUnix(context.Background(), "unixgram", nil,
		&net.UnixAddr{Name: path, Net: "unixgram"})
	fmt.Println("unixgram rejected:", err != nil)
	_, err = net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	fmt.Println("unixpacket rejected:", err != nil)
}
