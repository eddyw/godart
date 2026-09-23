package main

import (
	"fmt"
	"net"
)

func main() {
	for _, network := range []string{"", "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "http"} {
		for _, address := range []string{"", ":", "127.0.0.1:0", "[::]:0", "[::1]:65535", "[::ffff:127.0.0.1]:1", ":-1", ":65536", ":999999999999999999999999", "127.0.0.1:+1"} {
			tcp, err := net.ResolveTCPAddr(network, address)
			fmt.Printf("tcp %q %q: %v %T %v\n", network, address, tcp, err, err)
			udp, err := net.ResolveUDPAddr(network, address)
			fmt.Printf("udp %q %q: %v %T %v\n", network, address, udp, err, err)
		}
	}
	for _, network := range []string{"", "ip", "ip4", "ip6", "ip:0", "ip4:icmp", "ip6:IPv6-ICMP", "ip:igmp", "ip:tcp", "ip:udp", "ip:255", "ip:16777214", "ip:16777215", "ip:", "ip:+1", "ip:-1", "ip:unknown", "ip:sKip", "ip:0000001", "ip:1_0", "ip:1:2", "tcp", "tcp:icmp"} {
		for _, address := range []string{"", "127.0.0.1", "::", "::1", "::1%en0", "::%en0", "::ffff:127.0.0.1"} {
			ip, err := net.ResolveIPAddr(network, address)
			fmt.Printf("ip %q %q: %v %T %v\n", network, address, ip, err, err)
		}
	}
	tcp, err := net.ResolveTCPAddr("tcp", "127.0.0.1:http")
	fmt.Println(tcp, err)
	udp, err := net.ResolveUDPAddr("udp", "[::1]:domain")
	fmt.Println(udp, err)
}
