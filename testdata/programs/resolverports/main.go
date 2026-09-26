package main

import (
	"context"
	"errors"
	"fmt"
	"net"
)

func main() {
	for _, preferGo := range []bool{false, true} {
		r := net.Resolver{PreferGo: preferGo}
		for _, network := range []string{"tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "ip", ""} {
			for _, service := range []string{"--badport--", "bad\x00port"} {
				port, err := r.LookupPort(context.Background(), network, service)
				var dns *net.DNSError
				if !errors.As(err, &dns) {
					panic(fmt.Sprintf("LookupPort = %d, %T %v", port, err, err))
				}
				fmt.Printf("%t %q %q %d %q %q %q %t %t %t %v\n", preferGo, network, service,
					port, dns.Name, dns.Err, dns.Server, dns.IsNotFound, dns.Timeout(), dns.Temporary(), dns.Unwrap())
			}
		}
	}
}
