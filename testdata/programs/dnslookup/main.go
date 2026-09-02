package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
)

// The program runs a DNS server on the loopback interface, and it reads each
// function of the Lookup family through it. No lookup reaches the network.

// big names the record that the UDP server truncates, thus the client asks
// again over TCP.
const big = "big.example.com."

func main() {
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen udp:", err)
		return
	}
	defer udp.Close()
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen tcp:", err)
		return
	}
	defer tcp.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); serveUDP(udp) }()
	go func() { defer wg.Done(); serveTCP(tcp) }()

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			if network == "tcp" {
				return net.Dial("tcp", tcp.Addr().String())
			}
			return net.Dial("udp", udp.LocalAddr().String())
		},
	}
	ctx := context.Background()

	hosts, err := r.LookupHost(ctx, "example.com.")
	sort.Strings(hosts)
	fmt.Println("host:", hosts, err)

	v4, err := r.LookupIP(ctx, "ip4", "example.com.")
	fmt.Println("ip4:", texts(v4), err)

	v6, err := r.LookupIP(ctx, "ip6", "example.com.")
	fmt.Println("ip6:", texts(v6), err)

	addrs, err := r.LookupIPAddr(ctx, "example.com.")
	names := make([]string, 0, len(addrs))
	for _, a := range addrs {
		names = append(names, a.IP.String())
	}
	sort.Strings(names)
	fmt.Println("ipaddr:", names, err)

	netips, err := r.LookupNetIP(ctx, "ip4", "example.com.")
	fmt.Println("netip:", netips, err)

	cname, err := r.LookupCNAME(ctx, "www.example.com.")
	fmt.Println("cname:", cname, err)

	mx, err := r.LookupMX(ctx, "example.com.")
	for _, m := range mx {
		fmt.Printf("mx: %d %s\n", m.Pref, m.Host)
	}
	fmt.Println("mx err:", err)

	ns, err := r.LookupNS(ctx, "example.com.")
	hostsOf := make([]string, 0, len(ns))
	for _, n := range ns {
		hostsOf = append(hostsOf, n.Host)
	}
	sort.Strings(hostsOf)
	fmt.Println("ns:", hostsOf, err)

	txt, err := r.LookupTXT(ctx, "example.com.")
	sort.Strings(txt)
	fmt.Println("txt:", txt, err)

	target, srv, err := r.LookupSRV(ctx, "sip", "tcp", "example.com.")
	fmt.Println("srv name:", target, err)
	for _, s := range srv {
		fmt.Printf("srv: %d %d %d %s\n", s.Priority, s.Weight, s.Port, s.Target)
	}

	ptr, err := r.LookupAddr(ctx, "93.184.216.34")
	fmt.Println("addr:", ptr, err)

	// The UDP answer of this name holds the truncation flag, thus the client
	// asks again over TCP.
	long, err := r.LookupTXT(ctx, big)
	fmt.Println("tcp fallback:", len(long), lengthOf(long), err)

	// The report names the server of /etc/resolv.conf, thus the program reads
	// the fields and not the whole text.
	_, err = r.LookupHost(ctx, "missing.example.com.")
	var dnsErr *net.DNSError
	if ok := asDNSError(err, &dnsErr); ok {
		fmt.Println("missing:", dnsErr.Err, dnsErr.Name, dnsErr.IsNotFound, dnsErr.IsTimeout)
	} else {
		fmt.Println("missing gave", err)
	}

	port, err := r.LookupPort(ctx, "tcp", "80")
	fmt.Println("port 80:", port, err)
	port, err = r.LookupPort(ctx, "tcp", "http")
	fmt.Println("port http:", port, err)
	_, err = r.LookupPort(ctx, "tcp", "no-such-service")
	fmt.Println("port unknown:", err)
	port, err = r.LookupPort(ctx, "", "")
	fmt.Println("port empty:", port, err)
	_, err = r.LookupPort(ctx, "tcp", "70000")
	fmt.Println("port large:", err)
	_, err = r.LookupPort(ctx, "tcp", "-1")
	fmt.Println("port negative:", err)
	_, err = r.LookupPort(ctx, "sctp", "http")
	fmt.Println("port network:", err)

	// A literal address needs no server.
	literal, err := net.LookupHost("127.0.0.1")
	fmt.Println("literal:", literal, err)

	udp.Close()
	tcp.Close()
	wg.Wait()
}

// texts gives the text of each address.
func texts(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	sort.Strings(out)
	return out
}

// lengthOf gives the length of each text.
func lengthOf(list []string) []int {
	out := make([]int, 0, len(list))
	for _, s := range list {
		out = append(out, len(s))
	}
	return out
}

// asDNSError reads a DNSError out of an error.
func asDNSError(err error, out **net.DNSError) bool {
	e, ok := err.(*net.DNSError)
	if ok {
		*out = e
	}
	return ok
}

// serveUDP answers each query of the datagram socket.
func serveUDP(c net.PacketConn) {
	buf := make([]byte, 1500)
	for {
		n, from, err := c.ReadFrom(buf)
		if err != nil {
			return
		}
		query := append([]byte{}, buf[:n]...)
		host, _, _, ok := question(query)
		if !ok {
			continue
		}
		out := answer(query, host == big)
		if out != nil {
			c.WriteTo(out, from)
		}
	}
}

// serveTCP answers each query of the stream socket. A DNS message of a stream
// carries a length of 2 bytes before it.
func serveTCP(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			head := make([]byte, 2)
			if !readFull(c, head) {
				return
			}
			length := int(head[0])<<8 | int(head[1])
			query := make([]byte, length)
			if !readFull(c, query) {
				return
			}
			out := answer(query, false)
			if out == nil {
				return
			}
			c.Write(append([]byte{byte(len(out) >> 8), byte(len(out))}, out...))
		}(c)
	}
}

func readFull(c net.Conn, b []byte) bool {
	for off := 0; off < len(b); {
		n, err := c.Read(b[off:])
		off += n
		if err != nil {
			return off == len(b)
		}
	}
	return true
}

// unused keeps the import of strings for the record table of dns.go.
var _ = strings.TrimSuffix
