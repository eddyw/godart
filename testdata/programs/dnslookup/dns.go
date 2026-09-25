package main

import (
	"strconv"
	"strings"
)

// This file holds a small DNS server. The program runs it on the loopback
// interface, thus the lookups read no network.

const (
	typeA     = 1
	typeNS    = 2
	typeCNAME = 5
	typePTR   = 12
	typeMX    = 15
	typeTXT   = 16
	typeAAAA  = 28
	typeSRV   = 33
)

// record holds 1 answer of the table.
type record struct {
	rtype uint16
	// data holds the payload of the record, in the wire form.
	data []byte
}

// table gives the records of each name and type of the server.
var table = map[string][]record{
	"ordered.example.com.|1":         {{typeA, ipv4(127, 0, 0, 2)}, {typeA, ipv4(127, 0, 0, 1)}},
	"ordered.example.com.|28":        {{typeAAAA, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}},
	"v4.example.com.|1":              {{typeA, ipv4(192, 0, 2, 1)}},
	"v6.example.com.|28":             {{typeAAAA, ipv6()}},
	"zero.example.com.|28":           {{typeAAAA, make([]byte, 16)}},
	"example.com.|1":                 {{typeA, ipv4(93, 184, 216, 34)}},
	"example.com.|28":                {{typeAAAA, ipv6()}},
	"www.example.com.|5":             {{typeCNAME, name("example.com.")}},
	"example.com.|2":                 {{typeNS, name("ns1.example.com.")}, {typeNS, name("ns2.example.com.")}},
	"example.com.|16":                {{typeTXT, text("v=spf1 -all")}, {typeTXT, text("part1", "part2")}},
	"34.216.184.93.in-addr.arpa.|12": {{typePTR, name("example.com.")}},
	"_sip._tcp.example.com.|33": {
		{typeSRV, srv(20, 10, 5061, "backup.example.com.")},
		{typeSRV, srv(10, 60, 5060, "sipserver.example.com.")},
	},
	"example.com.|15": {
		{typeMX, mx(20, "mail2.example.com.")},
		{typeMX, mx(5, "mail1.example.com.")},
	},
	"big.example.com.|16": {{typeTXT, text(strings.Repeat("x", 200))}},
}

func ipv4(a, b, c, d byte) []byte { return []byte{a, b, c, d} }

// ipv6 gives the address 2606:2800:220:1:248:1893:25c8:1946.
func ipv6() []byte {
	return []byte{
		0x26, 0x06, 0x28, 0x00, 0x02, 0x20, 0x00, 0x01,
		0x02, 0x48, 0x18, 0x93, 0x25, 0xc8, 0x19, 0x46,
	}
}

// name writes a name in the label form.
func name(s string) []byte {
	out := []byte{}
	for _, label := range strings.Split(strings.TrimSuffix(s, "."), ".") {
		out = append(out, byte(len(label)))
		out = append(out, label...)
	}
	return append(out, 0)
}

// text writes 1 TXT record that holds each string.
func text(parts ...string) []byte {
	out := []byte{}
	for _, p := range parts {
		out = append(out, byte(len(p)))
		out = append(out, p...)
	}
	return out
}

func mx(pref uint16, host string) []byte {
	return append([]byte{byte(pref >> 8), byte(pref)}, name(host)...)
}

func srv(priority, weight, port uint16, target string) []byte {
	head := []byte{
		byte(priority >> 8), byte(priority),
		byte(weight >> 8), byte(weight),
		byte(port >> 8), byte(port),
	}
	return append(head, name(target)...)
}

// question reads the name and the type of the first question of a query, and
// it gives the offset after it.
func question(b []byte) (string, uint16, int, bool) {
	if len(b) < 12 {
		return "", 0, 0, false
	}
	var out strings.Builder
	off := 12
	for {
		if off >= len(b) {
			return "", 0, 0, false
		}
		n := int(b[off])
		if n == 0 {
			off++
			break
		}
		if off+1+n > len(b) {
			return "", 0, 0, false
		}
		out.Write(b[off+1 : off+1+n])
		out.WriteByte('.')
		off += 1 + n
	}
	if off+4 > len(b) {
		return "", 0, 0, false
	}
	qtype := uint16(b[off])<<8 | uint16(b[off+1])
	return out.String(), qtype, off + 4, true
}

// answer writes the answer of 1 query. A name that the table does not hold
// gives the code 3, which names a name that does not exist.
//
// truncate sets the flag that sends the client to TCP, and it drops each
// record, exactly as a server does for an answer that a datagram cannot hold.
func answer(query []byte, truncate bool) []byte {
	host, qtype, end, ok := question(query)
	if !ok {
		return nil
	}
	records := table[host+"|"+strconv.Itoa(int(qtype))]
	rcode := byte(0)
	if len(records) == 0 {
		rcode = 3
	}
	flags2 := byte(0x80)
	if truncate {
		flags2 |= 0x02
		records = nil
	}
	count := len(records)
	out := []byte{
		query[0], query[1],
		flags2, 0x80 | rcode,
		0x00, 0x01,
		byte(count >> 8), byte(count),
		0x00, 0x00,
		0x00, 0x00,
	}
	out = append(out, query[12:end]...)
	for _, r := range records {
		// A pointer to the offset 12 names the question again.
		out = append(out, 0xc0, 0x0c)
		out = append(out, byte(r.rtype>>8), byte(r.rtype))
		out = append(out, 0x00, 0x01)
		out = append(out, 0x00, 0x00, 0x01, 0x2c)
		out = append(out, byte(len(r.data)>>8), byte(len(r.data)))
		out = append(out, r.data...)
	}
	return out
}
