//go:build dart

package net

import "net/netip"

// The count of bytes of the 2 forms of an IP address.
const (
	IPv4len = 4
	IPv6len = 16
)

// An IP is an IP address. It holds 4 bytes or 16 bytes, and each address of
// this package holds the 16-byte form.
//
// Go also holds IPNet, ParseCIDR, and a mask. docs/parity.md holds that gap.
type IP []byte

// v4Prefix stands in front of the 4 bytes of an IPv4 address inside the
// 16-byte form.
var v4Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

// IPv4 gives the 16-byte form of an IPv4 address.
func IPv4(a, b, c, d byte) IP {
	out := make(IP, IPv6len)
	copy(out, v4Prefix)
	out[12], out[13], out[14], out[15] = a, b, c, d
	return out
}

// ParseIP reads the text of an IP address. It gives nil for a text that holds
// no address.
func ParseIP(s string) IP {
	addr, err := netip.ParseAddr(s)
	if err != nil || addr.Zone() != "" {
		return nil
	}
	if addr.Is4() {
		a := addr.As4()
		return IPv4(a[0], a[1], a[2], a[3])
	}
	a := addr.As16()
	out := make(IP, IPv6len)
	copy(out, a[:])
	return out
}

// To4 gives the 4-byte form of an IPv4 address, and it gives nil for each other
// address.
func (ip IP) To4() IP {
	if len(ip) == IPv4len {
		return ip
	}
	if len(ip) == IPv6len && isZeros(ip[0:10]) && ip[10] == 0xff && ip[11] == 0xff {
		return ip[12:16]
	}
	return nil
}

// To16 gives the 16-byte form of an address, and it gives nil for a length that
// no address holds.
func (ip IP) To16() IP {
	if len(ip) == IPv4len {
		return IPv4(ip[0], ip[1], ip[2], ip[3])
	}
	if len(ip) == IPv6len {
		return ip
	}
	return nil
}

// Equal reports whether the 2 addresses name 1 host. The 4-byte form and the
// 16-byte form of 1 IPv4 address are equal.
func (ip IP) Equal(x IP) bool {
	a, b := ip.To16(), x.To16()
	if a == nil || b == nil {
		return len(ip) == len(x) && sameBytes(ip, x)
	}
	return sameBytes(a, b)
}

// IsLoopback reports whether the address names the host itself.
func (ip IP) IsLoopback() bool {
	if v4 := ip.To4(); v4 != nil {
		return v4[0] == 127
	}
	return len(ip) == IPv6len && isZeros(ip[0:15]) && ip[15] == 1
}

// IsUnspecified reports whether the address is 0.0.0.0 or ::.
func (ip IP) IsUnspecified() bool {
	return len(ip) > 0 && isZeros(ip)
}

// String gives the text of the address. An empty address gives `<nil>`.
func (ip IP) String() string {
	if len(ip) == 0 {
		return "<nil>"
	}
	if v4 := ip.To4(); v4 != nil {
		return netip.AddrFrom4([4]byte{v4[0], v4[1], v4[2], v4[3]}).String()
	}
	if len(ip) == IPv6len {
		var a [16]byte
		copy(a[:], ip)
		return netip.AddrFrom16(a).String()
	}
	return "?" + hexText(ip)
}

// AppendText writes the text of the address after the bytes of b, and it
// gives the result. An empty address writes nothing. It reports an error for
// a length that no address holds.
//
// AppendText gives the interface encoding.TextAppender.
func (ip IP) AppendText(b []byte) ([]byte, error) {
	if len(ip) == 0 {
		return b, nil
	}
	if len(ip) != IPv4len && len(ip) != IPv6len {
		return b, &AddrError{Err: "invalid IP address", Addr: hexText(ip)}
	}
	return append(b, ip.String()...), nil
}

// MarshalText gives the text of the address. An empty address gives an empty
// slice.
//
// MarshalText gives the interface encoding.TextMarshaler.
func (ip IP) MarshalText() ([]byte, error) {
	// 24 bytes hold each IPv4 address and each short IPv6 address.
	b, err := ip.AppendText(make([]byte, 0, 24))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UnmarshalText reads the text of an address. An empty text gives an empty
// address.
//
// UnmarshalText gives the interface encoding.TextUnmarshaler.
func (ip *IP) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*ip = nil
		return nil
	}
	s := string(text)
	x := ParseIP(s)
	if x == nil {
		return &ParseError{Type: "IP address", Text: s}
	}
	*ip = x
	return nil
}

func isZeros(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hexText(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&0xf])
	}
	return string(out)
}
