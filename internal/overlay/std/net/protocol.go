// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"strings"
	"sync"
	"syscall/dart"
)

func resolveIPNetwork(network string) (string, error) {
	family := network
	colon := strings.LastIndexByte(network, ':')
	if colon >= 0 {
		family = network[:colon]
	}
	switch family {
	case "ip", "ip4", "ip6":
	default:
		return "", UnknownNetworkError(network)
	}
	if colon >= 0 {
		protocol := network[colon+1:]
		if n, ok := ipProtocolNumber(protocol); !ok || n != len(protocol) {
			readIPProtocols()
			var lower [len("RSVP-E2E-IGNORE") + 10]byte
			n := copy(lower[:], protocol)
			for i, c := range lower[:n] {
				if 'A' <= c && c <= 'Z' {
					lower[i] += 'a' - 'A'
				}
			}
			if n != len(protocol) || !ipProtocols[string(lower[:n])] {
				return "", &AddrError{Err: "unknown IP protocol specified", Addr: protocol}
			}
		}
	}
	return family, nil
}

func ipProtocolNumber(s string) (int, bool) {
	n, i := 0, 0
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		n = n*10 + int(s[i]-'0')
		if n >= 0xffffff {
			return i, false
		}
		i++
	}
	return i, i > 0
}

var (
	ipProtocolsOnce sync.Once
	ipProtocols     map[string]bool
)

func readIPProtocols() {
	ipProtocolsOnce.Do(func() {
		ipProtocols = map[string]bool{
			"icmp": true, "igmp": true, "tcp": true,
			"udp": true, "ipv6-icmp": true,
		}
		text, cause := dart.ReadFile("/etc/protocols")
		if cause != "" {
			return
		}
		for _, line := range strings.Split(string(text), "\n") {
			line, _, _ = strings.Cut(line, "#")
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			if _, ok := ipProtocolNumber(fields[1]); !ok {
				continue
			}
			ipProtocols[fields[0]] = true
			for _, alias := range fields[2:] {
				ipProtocols[alias] = true
			}
		}
	})
}
