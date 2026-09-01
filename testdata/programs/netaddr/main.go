package main

import (
	"fmt"
	"net"
)

func main() {
	// The mask family.
	fmt.Println(net.CIDRMask(24, 32), net.CIDRMask(64, 128))
	fmt.Println(net.CIDRMask(33, 32) == nil, net.CIDRMask(8, 20) == nil)
	m := net.CIDRMask(24, 32)
	ones, bits := m.Size()
	fmt.Println(ones, bits, m.String())
	fmt.Println(net.IPv4Mask(255, 255, 0, 0).String())

	// ParseCIDR and IPNet.
	for _, in := range []string{"10.1.2.3/8", "192.168.1.1/24", "2001:db8::1/32", "bad/8", "10.0.0.1/40"} {
		ip, n, err := net.ParseCIDR(in)
		fmt.Println(in, ip, n, err)
	}
	_, n, _ := net.ParseCIDR("10.0.0.0/8")
	fmt.Println(n.Contains(net.ParseIP("10.5.6.7")), n.Contains(net.ParseIP("11.5.6.7")))
	fmt.Println(n.Network(), n.String())

	// The mask of an address.
	ip := net.ParseIP("192.168.1.130")
	fmt.Println(ip.Mask(net.CIDRMask(24, 32)))
	fmt.Println(ip.DefaultMask())

	// The address types.
	a := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	fmt.Println(a.Network(), a.String())
	u := &net.UDPAddr{IP: net.ParseIP("::1"), Port: 53}
	fmt.Println(u.Network(), u.String())
	fmt.Println((&net.UnixAddr{Name: "/tmp/s", Net: "unix"}).String())
	fmt.Println((&net.IPAddr{IP: net.ParseIP("10.0.0.1")}).String())

	r, err := net.ResolveTCPAddr("tcp", "127.0.0.1:80")
	fmt.Println(r, err)
	_, err = net.ResolveTCPAddr("sctp", "127.0.0.1:80")
	fmt.Println(err)
}
