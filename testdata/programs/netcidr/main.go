package main

import (
	"fmt"
	"net"
)

func main() {
	for _, input := range []string{
		"192.0.2.1/24", "::ffff:192.0.2.1/128", "::ffff:192.0.2.1/120",
		"::ffff:192.0.2.1/96", "::ffff:192.0.2.1/32", "2001:db8::1/64",
		"::ffff:192.0.2.1/129", "192.0.2.1/128", "fe80::1%en0/64",
		"192.0.2.1/+24", "192.0.2.1/00024",
	} {
		ip, network, err := net.ParseCIDR(input)
		fmt.Println(input, ip, len(ip), err)
		if err == nil {
			ones, bits := network.Mask.Size()
			fmt.Println(network, len(network.IP), ones, bits, network.Contains(ip))
		}
	}
}
