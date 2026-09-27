package main

import (
	"context"
	"fmt"
	"net"
)

func main() {
	ctx := context.Background()
	for _, r := range []*net.Resolver{nil, {PreferGo: true}} {
		for _, host := range []string{"", "127.0.0.1", "::1", "::ffff:127.0.0.1", "::", "fe80::1%test-zone"} {
			hosts, err := r.LookupHost(ctx, host)
			fmt.Printf("host %q %q %T %v\n", host, hosts, err, err)
			addrs, err := r.LookupIPAddr(ctx, host)
			fmt.Printf("addr %q %T %v", host, err, err)
			for _, a := range addrs {
				fmt.Printf(" %s/%d/%s", a.IP, len(a.IP), a.Zone)
			}
			fmt.Println()
			for _, network := range []string{"ip", "ip4", "ip6", "ip4:icmp"} {
				ips, err := r.LookupIP(ctx, network, host)
				fmt.Printf("ip %s %q %T %v", network, host, err, err)
				for _, ip := range ips {
					fmt.Printf(" %s/%d", ip, len(ip))
				}
				fmt.Println()
				values, err := r.LookupNetIP(ctx, network, host)
				fmt.Printf("netip %s %q %v %T %v\n", network, host, values, err, err)
			}
		}
	}
}
