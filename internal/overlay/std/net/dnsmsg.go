//go:build dart

package net

import (
	"errors"
	"strings"
)

// This file holds the message format of the DNS. The real source of Go reads
// the vendored package `golang.org/x/net/dns/dnsmessage`, and that package
// stands outside the surface.
//
// The subset holds the record types that the Lookup family of `net` needs.

// The record types of the Lookup family.
const (
	dnsTypeA     = 1
	dnsTypeNS    = 2
	dnsTypeCNAME = 5
	dnsTypePTR   = 12
	dnsTypeMX    = 15
	dnsTypeTXT   = 16
	dnsTypeAAAA  = 28
	dnsTypeSRV   = 33
)

// dnsClassINET is the only class that the subset asks for.
const dnsClassINET = 1

// The response codes of the header that the resolver reads.
const (
	dnsRcodeSuccess  = 0
	dnsRcodeFormat   = 1
	dnsRcodeServer   = 2
	dnsRcodeName     = 3
	dnsRcodeNotImpl  = 4
	dnsRcodeRefused  = 5
	dnsMaxMessage    = 65535
	dnsMaxNameLength = 255
	dnsMaxPointers   = 32
)

// dnsRecord holds 1 resource record of an answer. Each field of a payload
// stands here, and the type names the fields that hold a value.
type dnsRecord struct {
	name  string
	rtype uint16
	// ip holds the payload of an A record and of an AAAA record.
	ip IP
	// target holds the name of a CNAME, an NS, a PTR, an MX, and an SRV.
	target string
	// pref holds the preference of an MX.
	pref uint16
	// priority, weight, and port hold the rest of an SRV.
	priority uint16
	weight   uint16
	port     uint16
	// txt holds the text of a TXT, with each string joined and no separator.
	txt string
}

// dnsMessage holds the part of an answer that the resolver reads.
type dnsMessage struct {
	id        uint16
	rcode     int
	truncated bool
	question  string
	qtype     uint16
	answers   []dnsRecord
}

var errDNSMalformed = errors.New("the answer of the server holds no valid message")

// dnsQuery writes the query of 1 name and 1 type.
func dnsQuery(id uint16, name string, qtype uint16) ([]byte, error) {
	out := make([]byte, 0, 32+len(name))
	out = append(out,
		byte(id>>8), byte(id),
		0x01, 0x00, // the recursion desired bit
		0x00, 0x01, // 1 question
		0x00, 0x00, // no answer
		0x00, 0x00, // no authority
		0x00, 0x00, // no additional record
	)
	body, err := dnsName(name)
	if err != nil {
		return nil, err
	}
	out = append(out, body...)
	out = append(out, byte(qtype>>8), byte(qtype), 0x00, dnsClassINET)
	return out, nil
}

// dnsName writes a name in the label form. The result ends with the root
// label, which holds no byte.
func dnsName(name string) ([]byte, error) {
	name = strings.TrimSuffix(name, ".")
	if len(name) > dnsMaxNameLength {
		return nil, errors.New("the name holds more than 255 characters")
	}
	out := make([]byte, 0, len(name)+2)
	if name != "" {
		for _, label := range strings.Split(name, ".") {
			if label == "" {
				return nil, errors.New("the name holds an empty label")
			}
			if len(label) > 63 {
				return nil, errors.New("the label " + label + " holds more than 63 characters")
			}
			out = append(out, byte(len(label)))
			out = append(out, label...)
		}
	}
	return append(out, 0), nil
}

// dnsParse reads an answer.
func dnsParse(b []byte) (*dnsMessage, error) {
	if len(b) < 12 {
		return nil, errDNSMalformed
	}
	m := &dnsMessage{
		id:        uint16(b[0])<<8 | uint16(b[1]),
		rcode:     int(b[3] & 0x0f),
		truncated: b[2]&0x02 != 0,
	}
	qdcount := int(b[4])<<8 | int(b[5])
	ancount := int(b[6])<<8 | int(b[7])
	off := 12
	for i := 0; i < qdcount; i++ {
		name, next, err := dnsReadName(b, off)
		if err != nil {
			return nil, err
		}
		if next+4 > len(b) {
			return nil, errDNSMalformed
		}
		if i == 0 {
			m.question = name
			m.qtype = uint16(b[next])<<8 | uint16(b[next+1])
		}
		off = next + 4
	}
	for i := 0; i < ancount; i++ {
		rec, next, err := dnsReadRecord(b, off)
		if err != nil {
			return nil, err
		}
		off = next
		if rec != nil {
			m.answers = append(m.answers, *rec)
		}
	}
	return m, nil
}

// dnsReadRecord reads 1 resource record. It gives a nil record for a type that
// the subset does not hold, and the offset then stands after the record.
func dnsReadRecord(b []byte, off int) (*dnsRecord, int, error) {
	name, next, err := dnsReadName(b, off)
	if err != nil {
		return nil, 0, err
	}
	if next+10 > len(b) {
		return nil, 0, errDNSMalformed
	}
	rtype := uint16(b[next])<<8 | uint16(b[next+1])
	length := int(b[next+8])<<8 | int(b[next+9])
	body := next + 10
	end := body + length
	if end > len(b) {
		return nil, 0, errDNSMalformed
	}
	rec := &dnsRecord{name: name, rtype: rtype}
	switch rtype {
	case dnsTypeA:
		if length != 4 {
			return nil, 0, errDNSMalformed
		}
		rec.ip = IP(append([]byte{}, b[body:end]...)).To16()
	case dnsTypeAAAA:
		if length != 16 {
			return nil, 0, errDNSMalformed
		}
		rec.ip = IP(append([]byte{}, b[body:end]...))
	case dnsTypeNS, dnsTypeCNAME, dnsTypePTR:
		target, _, err := dnsReadName(b, body)
		if err != nil {
			return nil, 0, err
		}
		rec.target = target
	case dnsTypeMX:
		if length < 3 {
			return nil, 0, errDNSMalformed
		}
		rec.pref = uint16(b[body])<<8 | uint16(b[body+1])
		target, _, err := dnsReadName(b, body+2)
		if err != nil {
			return nil, 0, err
		}
		rec.target = target
	case dnsTypeSRV:
		if length < 7 {
			return nil, 0, errDNSMalformed
		}
		rec.priority = uint16(b[body])<<8 | uint16(b[body+1])
		rec.weight = uint16(b[body+2])<<8 | uint16(b[body+3])
		rec.port = uint16(b[body+4])<<8 | uint16(b[body+5])
		target, _, err := dnsReadName(b, body+6)
		if err != nil {
			return nil, 0, err
		}
		rec.target = target
	case dnsTypeTXT:
		// A TXT record holds 1 string or more. Go joins them with no
		// separator, thus a long text that the server split reads whole.
		var text strings.Builder
		for i := body; i < end; {
			n := int(b[i])
			if i+1+n > end {
				return nil, 0, errDNSMalformed
			}
			text.Write(b[i+1 : i+1+n])
			i += 1 + n
		}
		rec.txt = text.String()
	default:
		return nil, end, nil
	}
	return rec, end, nil
}

// dnsReadName reads a name that starts at off. A label that holds the 2 high
// bits is a pointer to an earlier name, thus the read follows it.
func dnsReadName(b []byte, off int) (string, int, error) {
	var out strings.Builder
	next := 0
	jumps := 0
	for {
		if off >= len(b) {
			return "", 0, errDNSMalformed
		}
		n := int(b[off])
		switch {
		case n == 0:
			off++
			if next == 0 {
				next = off
			}
			if out.Len() == 0 {
				return ".", next, nil
			}
			return out.String(), next, nil
		case n&0xc0 == 0xc0:
			if off+1 >= len(b) {
				return "", 0, errDNSMalformed
			}
			target := (n&0x3f)<<8 | int(b[off+1])
			if next == 0 {
				next = off + 2
			}
			jumps++
			if jumps > dnsMaxPointers || target >= off {
				return "", 0, errDNSMalformed
			}
			off = target
		case n&0xc0 != 0:
			return "", 0, errDNSMalformed
		default:
			if off+1+n > len(b) {
				return "", 0, errDNSMalformed
			}
			if out.Len()+n+1 > dnsMaxNameLength {
				return "", 0, errDNSMalformed
			}
			out.Write(b[off+1 : off+1+n])
			out.WriteByte('.')
			off += 1 + n
		}
	}
}
