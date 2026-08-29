package main

import (
	"crypto/tls"
	"fmt"
	"net"
)

func main() {
	closed := freePort()
	c, err := tls.Dial("tcp", closed, &tls.Config{InsecureSkipVerify: true})
	fmt.Println("dial", c == nil, err != nil)

	_, err = tls.Dial("udp", closed, nil)
	fmt.Println("network", err != nil)

	cfg := &tls.Config{
		ServerName:         "example.com",
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true,
	}
	clone := cfg.Clone()
	fmt.Println(clone.ServerName, clone.NextProtos, clone.InsecureSkipVerify)
	clone.NextProtos[0] = "changed"
	fmt.Println(cfg.NextProtos[0])

	var none *tls.Config
	fmt.Println(none.Clone() == nil)

	fmt.Println(tls.VersionTLS12 == 0x0303, tls.VersionTLS13 == 0x0304)
}

// freePort gives an address that holds no listener.
func freePort() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "127.0.0.1:1"
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}
