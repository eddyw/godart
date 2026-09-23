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
	tcp, err := net.ResolveTCPAddr("tcp", "127.0.0.1:http")
	fmt.Println(tcp, err)
	udp, err := net.ResolveUDPAddr("udp", "[::1]:domain")
	fmt.Println(udp, err)
}
