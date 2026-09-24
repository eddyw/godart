// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"context"
	"net/netip"
	"strconv"
	"strings"
)

// The address types and the mask family of Go.
//
// Each address of the package holds an IP and a port, thus Conn.RemoteAddr and
// Conn.LocalAddr give a value that a program can read.

// An IPMask is the mask of an IP network. It holds 4 bytes or 16 bytes.
type IPMask []byte

// IPv4Mask gives the 4-byte mask of the 4 bytes.
func IPv4Mask(a, b, c, d byte) IPMask { return IPMask{a, b, c, d} }

// CIDRMask gives a mask that holds ones leading bits, of a mask of bits bits.
// It gives nil for a pair that no mask holds.
func CIDRMask(ones, bits int) IPMask {
	if bits != 8*IPv4len && bits != 8*IPv6len {
		return nil
	}
	if ones < 0 || ones > bits {
		return nil
	}
	out := make(IPMask, bits/8)
	for i := range out {
		if ones >= 8 {
			out[i] = 0xff
			ones -= 8
			continue
		}
		out[i] = ^byte(0xff >> ones)
		ones = 0
	}
	return out
}

// Size gives the count of leading ones and the count of bits. It gives 0 and 0
// for a mask that holds no leading block of ones.
func (m IPMask) Size() (ones, bits int) {
	ones, bits = simpleMaskLength(m), len(m)*8
	if ones == -1 {
		return 0, 0
	}
	return ones, bits
}

// String gives the hexadecimal form of the mask.
func (m IPMask) String() string {
	if len(m) == 0 {
		return "<nil>"
	}
	return hexText(m)
}

// simpleMaskLength gives the count of leading ones of a mask whose bits stand
// in 1 block. It gives -1 for each other mask.
func simpleMaskLength(m IPMask) int {
	var n int
	for i, b := range m {
		if b == 0xff {
			n += 8
			continue
		}
		for b&0x80 != 0 {
			n++
			b <<= 1
		}
		if b != 0 {
			return -1
		}
		for i++; i < len(m); i++ {
			if m[i] != 0 {
				return -1
			}
		}
		break
	}
	return n
}

// Mask gives the address with the mask applied. It gives nil when the 2
// lengths do not fit.
func (ip IP) Mask(mask IPMask) IP {
	if len(mask) == IPv6len && len(ip) == IPv4len && allFF(mask[:12]) {
		mask = mask[12:]
	}
	if len(mask) == IPv4len && len(ip) == IPv6len && sameBytes(ip[:12], v4Prefix) {
		ip = ip[12:]
	}
	n := len(ip)
	if n != len(mask) {
		return nil
	}
	out := make(IP, n)
	for i := range n {
		out[i] = ip[i] & mask[i]
	}
	return out
}

func allFF(b []byte) bool {
	for _, c := range b {
		if c != 0xff {
			return false
		}
	}
	return true
}

// DefaultMask gives the default mask of an IPv4 address. It gives nil for each
// other address.
func (ip IP) DefaultMask() IPMask {
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	switch {
	case ip[0] < 0x80:
		return IPv4Mask(0xff, 0, 0, 0)
	case ip[0] < 0xC0:
		return IPv4Mask(0xff, 0xff, 0, 0)
	}
	return IPv4Mask(0xff, 0xff, 0xff, 0)
}

// An IPNet is an IP network. IP holds the address of the network, and Mask
// holds its mask.
type IPNet struct {
	IP   IP
	Mask IPMask
}

// Contains reports whether the network holds the address.
func (n *IPNet) Contains(ip IP) bool {
	nn, m := networkNumberAndMask(n)
	if x := ip.To4(); x != nil {
		ip = x
	}
	if len(ip) != len(nn) {
		return false
	}
	for i := range ip {
		if nn[i]&m[i] != ip[i]&m[i] {
			return false
		}
	}
	return true
}

// Network gives the name "ip+net".
func (n *IPNet) Network() string { return "ip+net" }

// String gives the CIDR form of the network, for example 10.0.0.0/8.
func (n *IPNet) String() string {
	if n == nil {
		return "<nil>"
	}
	nn, m := networkNumberAndMask(n)
	if nn == nil || m == nil {
		return "<nil>"
	}
	ones := simpleMaskLength(m)
	if ones == -1 {
		return nn.String() + "/" + hexText(m)
	}
	return nn.String() + "/" + strconv.Itoa(ones)
}

func networkNumberAndMask(n *IPNet) (IP, IPMask) {
	ip := n.IP.To4()
	if ip == nil {
		ip = n.IP
		if len(ip) != IPv6len {
			return nil, nil
		}
	}
	m := n.Mask
	switch len(m) {
	case IPv4len:
		if len(ip) != IPv4len {
			return nil, nil
		}
	case IPv6len:
		if len(ip) == IPv4len {
			m = m[12:]
		}
	default:
		return nil, nil
	}
	return ip, m
}

// ParseCIDR reads the CIDR form of an address, for example 10.0.0.1/8. It
// gives the address and the network that holds it.
func ParseCIDR(s string) (IP, *IPNet, error) {
	i := strings.IndexByte(s, '/')
	if i < 0 {
		return nil, nil, &ParseError{Type: "CIDR address", Text: s}
	}
	addr, mask := s[:i], s[i+1:]
	ipAddr, err := netip.ParseAddr(addr)
	if err != nil || ipAddr.Zone() != "" {
		return nil, nil, &ParseError{Type: "CIDR address", Text: s}
	}
	bits := ipAddr.BitLen()
	ones, err := strconv.Atoi(mask)
	if err != nil || mask == "" || ones < 0 || ones > bits || mask[0] < '0' || mask[0] > '9' {
		return nil, nil, &ParseError{Type: "CIDR address", Text: s}
	}
	m := CIDRMask(ones, bits)
	addr16 := ipAddr.As16()
	ip := IP(addr16[:])
	return ip, &IPNet{IP: ip.Mask(m), Mask: m}, nil
}

// A ParseError names a text that holds no address.
type ParseError struct {
	Type string
	Text string
}

func (e *ParseError) Error() string {
	return "invalid " + e.Type + ": " + e.Text
}

// Timeout reports whether the failure came from a deadline. A parse error
// holds none.
func (e *ParseError) Timeout() bool { return false }

// Temporary reports whether a second try may work. A parse error gives false.
func (e *ParseError) Temporary() bool { return false }

// A TCPAddr is the address of a TCP connection or of a TCP listener.
type TCPAddr struct {
	IP   IP
	Port int
	Zone string
}

func (a *TCPAddr) Network() string { return "tcp" }

func (a *TCPAddr) String() string { return addrText(a.IP, a.Port, a.Zone) }

// A UDPAddr is the address of a datagram socket.
type UDPAddr struct {
	IP   IP
	Port int
	Zone string
}

func (a *UDPAddr) Network() string { return "udp" }

func (a *UDPAddr) String() string { return addrText(a.IP, a.Port, a.Zone) }

// An IPAddr is an address that holds no port.
type IPAddr struct {
	IP   IP
	Zone string
}

func (a *IPAddr) Network() string { return "ip" }

func (a *IPAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	out := ""
	if len(a.IP) != 0 {
		out = a.IP.String()
	}
	if a.Zone != "" {
		return out + "%" + a.Zone
	}
	return out
}

// A UnixAddr is the address of a Unix socket.
type UnixAddr struct {
	Name string
	Net  string
}

func (a *UnixAddr) Network() string {
	if a == nil || a.Net == "" {
		return "unix"
	}
	return a.Net
}

func (a *UnixAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	return a.Name
}

func addrText(ip IP, port int, zone string) string {
	host := ""
	if len(ip) != 0 {
		host = ip.String()
	}
	if zone != "" {
		host += "%" + zone
	}
	return JoinHostPort(host, strconv.Itoa(port))
}

// ResolveTCPAddr resolves a TCP address through DefaultResolver.
func ResolveTCPAddr(network, address string) (*TCPAddr, error) {
	switch network {
	case "":
		network = "tcp"
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, UnknownNetworkError(network)
	}
	ip, port, zone, err := parseAddr(network, address)
	if err != nil {
		return nil, err
	}
	return &TCPAddr{IP: ip, Port: port, Zone: zone}, nil
}

// ResolveUDPAddr reads the text of a UDP address.
func ResolveUDPAddr(network, address string) (*UDPAddr, error) {
	switch network {
	case "":
		network = "udp"
	case "udp", "udp4", "udp6":
	default:
		return nil, UnknownNetworkError(network)
	}
	ip, port, zone, err := parseAddr(network, address)
	if err != nil {
		return nil, err
	}
	return &UDPAddr{IP: ip, Port: port, Zone: zone}, nil
}

// ResolveIPAddr resolves an address without a port through DefaultResolver.
// The network can include an IP protocol name or number.
func ResolveIPAddr(network, address string) (*IPAddr, error) {
	if network == "" {
		network = "ip"
	}
	family, err := resolveIPNetwork(network)
	if err != nil {
		return nil, err
	}
	return resolveInternetIP(family, address, network == "ip" && strings.Contains(address, ":"))
}

// ResolveUnixAddr reads the text of a Unix socket address.
func ResolveUnixAddr(network, address string) (*UnixAddr, error) {
	switch network {
	case "unix", "unixgram", "unixpacket":
		return &UnixAddr{Name: address, Net: network}, nil
	}
	return nil, UnknownNetworkError(network)
}

// An AddrError reports a failure that names an address.
type AddrError struct {
	Err  string
	Addr string
}

func (e *AddrError) Error() string {
	if e == nil {
		return "<nil>"
	}
	s := e.Err
	if e.Addr != "" {
		s = "address " + e.Addr + ": " + s
	}
	return s
}

// Timeout reports whether the failure came from a deadline. An address error
// holds none.
func (e *AddrError) Timeout() bool { return false }

// Temporary reports whether a second try may work. An address error gives
// false.
func (e *AddrError) Temporary() bool { return false }

// An InvalidAddrError names an address that an operation cannot take. The
// text of the value is the whole message.
type InvalidAddrError string

func (e InvalidAddrError) Error() string { return string(e) }

// Timeout reports whether the failure came from a deadline. An invalid
// address holds none.
func (e InvalidAddrError) Timeout() bool { return false }

// Temporary reports whether a second try may work. An invalid address gives
// false.
func (e InvalidAddrError) Temporary() bool { return false }

// An UnknownNetworkError names a network that no operation holds.
type UnknownNetworkError string

func (e UnknownNetworkError) Error() string { return "unknown network " + string(e) }

func (e UnknownNetworkError) Timeout() bool { return false }

func (e UnknownNetworkError) Temporary() bool { return false }

// parseAddr reads a host and a port, and it gives the address of the host.
func parseAddr(network, address string) (IP, int, string, error) {
	if address == "" {
		return nil, 0, "", nil
	}
	host, portText, err := SplitHostPort(address)
	if err != nil {
		return nil, 0, "", err
	}
	port, err := LookupPort(network, portText)
	if err != nil {
		return nil, 0, "", err
	}
	addr, err := resolveInternetIP(network, host, (network == "tcp" || network == "udp") && strings.Contains(address, "["))
	if err != nil {
		return nil, 0, "", err
	}
	return addr.IP, port, addr.Zone, nil
}

func resolveInternetIP(network, host string, prefer6 bool) (*IPAddr, error) {
	if host == "" {
		return &IPAddr{}, nil
	}
	family := "ip"
	if last := network[len(network)-1]; last == '4' || last == '6' {
		family += string(last)
	}
	var addresses []IPAddr
	if ip, err := netip.ParseAddr(host); err == nil {
		addresses = []IPAddr{{IP: IP(ip.AsSlice()).To16(), Zone: ip.Zone()}}
	} else {
		ips, err := DefaultResolver.LookupIP(context.Background(), family, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			addresses = append(addresses, IPAddr{IP: ip})
		}
	}
	if len(addresses) == 1 && addresses[0].IP.Equal(IPv6unspecified) {
		addresses = append(addresses, IPAddr{IP: IPv4zero})
	}
	var first *IPAddr
	for i := range addresses {
		addr := &addresses[i]
		if !keepIP(family, addr.IP) {
			continue
		}
		if first == nil {
			first = addr
		}
		if (addr.IP.To4() == nil) == prefer6 {
			return addr, nil
		}
	}
	if first != nil {
		return first, nil
	}
	return nil, &AddrError{Err: "no suitable address found", Addr: host}
}

// tcpAddrOf builds a TCPAddr from a host text and a port. A host that is not
// an IP address leaves the IP empty, and String then writes the host again.
func tcpAddrOf(host string, port int) *TCPAddr {
	zone := ""
	if i := strings.LastIndexByte(host, '%'); i >= 0 {
		host, zone = host[:i], host[i+1:]
	}
	return &TCPAddr{IP: ParseIP(host), Port: port, Zone: zone}
}

// udpAddrOf builds a UDPAddr from a host text and a port.
func udpAddrOf(host string, port int) *UDPAddr {
	ip := ParseIP(host)
	if !strings.Contains(host, ":") {
		ip = ip.To4()
	}
	return &UDPAddr{IP: ip, Port: port}
}
