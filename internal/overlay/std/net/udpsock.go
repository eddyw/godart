// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"errors"
	"net/netip"
	"os"
	"syscall"
	"syscall/dart"
	"time"
)

// Go embeds the unexported type `conn` in `UDPConn`. The overlay names each
// method again. A promoted method of an unexported type reaches no other
// package in the Dart output.

// A UDPConn is a datagram socket of the network udp.
//
// A socket of a dial holds a peer. Such a socket reads the datagrams of the
// peer alone, and Write sends to the peer. A socket of a listen holds no
// peer. It reads each datagram that arrives, and each write names the
// target.
type UDPConn struct {
	p *packet
	// host and port name the peer of a socket that a dial connected. The
	// host is empty for a socket of a listen.
	host string
	port int
}

// newUDPConn wraps a socket that a listen opened.
func newUDPConn(p *packet) *UDPConn { return &UDPConn{p: p} }

// setOption writes 1 socket option of the datagram socket.
func (c *UDPConn) setOption(name string, value int) error {
	if c.p == nil {
		return syscall.EINVAL
	}
	if cause, code := dart.PacketOption(c.p.handle, name, value); cause != "" {
		return &OpError{Op: "set", Net: c.p.network, Addr: c.p.local, Err: socketSyscallCause("setsockopt", cause, code)}
	}
	return nil
}

// SetReadBuffer writes the size of the receive buffer of the socket.
func (c *UDPConn) SetReadBuffer(bytes int) error {
	return c.setOption("rcvbuf", bytes)
}

// SetWriteBuffer writes the size of the send buffer of the socket.
func (c *UDPConn) SetWriteBuffer(bytes int) error {
	return c.setOption("sndbuf", bytes)
}

// newDialedUDPConn wraps a socket that a dial connected to a peer.
func newDialedUDPConn(p *packet, host string, port int) *UDPConn {
	p.remote = udpAddrOf(host, port)
	return &UDPConn{p: p, host: host, port: port}
}

// connected reports whether a dial named the peer of the socket.
func (c *UDPConn) connected() bool { return c.host != "" }

// Read reads 1 datagram into the slice and gives the count. A datagram that
// holds more bytes than the slice loses the rest, exactly as in Go.
// An empty slice leaves the datagram unread.
func (c *UDPConn) Read(b []byte) (int, error) {
	if len(b) == 0 && c.p != nil && c.p.handle >= 0 {
		return 0, nil
	}
	n, _, err := c.p.readFrom(b, "read")
	return n, err
}

// Write sends the bytes of the slice to the peer of the socket. A socket that
// no dial connected gives an error, exactly as in Go.
func (c *UDPConn) Write(b []byte) (int, error) {
	if c.p == nil {
		return 0, syscall.EINVAL
	}
	if !c.connected() {
		return 0, errors.New("write udp: destination address required")
	}
	n, cause, code := dart.PacketWrite(c.p.handle, b, c.host, c.port)
	if cause != "" {
		return n, &OpError{Op: "write", Net: c.p.network, Source: c.p.local, Addr: c.p.remote, Err: socketSyscallCause("write", cause, code)}
	}
	return n, nil
}

// ReadFrom reads 1 datagram and gives the address of its source.
func (c *UDPConn) ReadFrom(b []byte) (int, Addr, error) { return c.p.ReadFrom(b) }

// ReadFromUDP reads 1 datagram and gives the address of its source.
func (c *UDPConn) ReadFromUDP(b []byte) (int, *UDPAddr, error) {
	n, host, port, err := c.readFrom(b)
	if err != nil {
		return n, nil, err
	}
	return n, udpAddrOf(host, port), nil
}

// ReadFromUDPAddrPort reads 1 datagram and gives the address of its source in
// the form of net/netip.
func (c *UDPConn) ReadFromUDPAddrPort(b []byte) (int, netip.AddrPort, error) {
	n, host, port, err := c.readFrom(b)
	if err != nil {
		return n, netip.AddrPort{}, err
	}
	addr, ok := netip.AddrFromSlice(ParseIP(host))
	if !ok {
		return n, netip.AddrPort{}, &ParseError{Type: "IP address", Text: host}
	}
	return n, netip.AddrPortFrom(addr.Unmap(), uint16(port)), nil
}

// readFrom reads 1 datagram and gives the host and the port of its source.
func (c *UDPConn) readFrom(b []byte) (int, string, int, error) {
	if c.p == nil {
		return 0, "", 0, syscall.EINVAL
	}
	n, host, port, cause, code := dart.PacketRead(c.p.handle, b)
	if cause != "" {
		return n, "", 0, &OpError{Op: "read", Net: c.p.network, Source: c.p.local, Addr: c.p.remote, Err: socketSyscallCause("recvfrom", cause, code)}
	}
	return n, host, port, nil
}

func (c *UDPConn) writeTo(b []byte, addr *UDPAddr) (int, error) {
	if c.connected() {
		return 0, ErrWriteToConnected
	}
	if addr == nil {
		return 0, errMissingAddress
	}
	ip := addr.IP
	ipv4 := c.p.local.IP.To4() != nil
	if len(ip) != 0 && (ip.To16() == nil || ipv4 && ip.To4() == nil) {
		cause := "non-IPv6 address"
		if ipv4 {
			cause = "non-IPv4 address"
		}
		return 0, &AddrError{Err: cause, Addr: ip.String()}
	}
	if len(ip) == 0 {
		ip = IPv4zero
	}
	if !ipv4 && ip.Equal(IPv4zero) {
		ip = IPv6zero
	}
	n, cause, code := dart.PacketWrite(c.p.handle, b, ip.String(), addr.Port)
	if cause != "" {
		return n, socketSyscallCause("sendto", cause, code)
	}
	return n, nil
}

func (a *UDPAddr) opAddr() Addr {
	if a == nil {
		return nil
	}
	return a
}

type addrPortUDPAddr struct {
	netip.AddrPort
}

func (addrPortUDPAddr) Network() string { return "udp" }

func (c *UDPConn) WriteTo(b []byte, addr Addr) (int, error) {
	if c.p == nil {
		return 0, syscall.EINVAL
	}
	a, ok := addr.(*UDPAddr)
	if !ok {
		return 0, &OpError{Op: "write", Net: c.p.network, Source: c.p.local, Addr: addr, Err: syscall.EINVAL}
	}
	n, err := c.writeTo(b, a)
	if err != nil {
		err = &OpError{Op: "write", Net: c.p.network, Source: c.p.local, Addr: a.opAddr(), Err: err}
	}
	return n, err
}

func (c *UDPConn) WriteToUDP(b []byte, addr *UDPAddr) (int, error) {
	if c.p == nil {
		return 0, syscall.EINVAL
	}
	n, err := c.writeTo(b, addr)
	if err != nil {
		err = &OpError{Op: "write", Net: c.p.network, Source: c.p.local, Addr: addr.opAddr(), Err: err}
	}
	return n, err
}

func (c *UDPConn) WriteToUDPAddrPort(b []byte, addr netip.AddrPort) (int, error) {
	if c.p == nil {
		return 0, syscall.EINVAL
	}
	var target *UDPAddr
	if addr.IsValid() {
		target = UDPAddrFromAddrPort(addr)
	}
	n, err := c.writeTo(b, target)
	if err != nil {
		err = &OpError{Op: "write", Net: c.p.network, Source: c.p.local, Addr: addrPortUDPAddr{addr}, Err: err}
	}
	return n, err
}

// Close ends the socket.
func (c *UDPConn) Close() error {
	if c.p == nil {
		return syscall.EINVAL
	}
	return c.p.Close()
}

// LocalAddr gives the address of the socket.
func (c *UDPConn) LocalAddr() Addr { return c.p.LocalAddr() }

// RemoteAddr gives the peer of a socket that a dial connected. A socket of a
// listen gives nil.
func (c *UDPConn) RemoteAddr() Addr {
	if !c.connected() {
		return nil
	}
	return udpAddrOf(c.host, c.port)
}

// SetDeadline writes the deadline of a read and of a write.
func (c *UDPConn) SetDeadline(t time.Time) error { return c.p.SetDeadline(t) }

// SetReadDeadline writes the deadline of a read.
func (c *UDPConn) SetReadDeadline(t time.Time) error { return c.p.SetReadDeadline(t) }

// SetWriteDeadline writes the deadline of a write.
func (c *UDPConn) SetWriteDeadline(t time.Time) error { return c.p.SetWriteDeadline(t) }

// DialUDP opens a datagram socket that holds a peer. The network is udp,
// udp4, or udp6.
func DialUDP(network string, laddr, raddr *UDPAddr) (*UDPConn, error) {
	switch network {
	case "udp", "udp4", "udp6":
	default:
		return nil, &OpError{Op: "dial", Net: network, Source: laddr.opAddr(), Addr: raddr.opAddr(), Err: UnknownNetworkError(network)}
	}
	if raddr == nil {
		return nil, &OpError{Op: "dial", Net: network, Source: laddr.opAddr(), Err: errMissingAddress}
	}
	ipv6 := network == "udp6" || network == "udp" &&
		(len(raddr.IP) > IPv4len && raddr.IP.To4() == nil ||
			laddr != nil && len(laddr.IP) > IPv4len && laddr.IP.To4() == nil)
	if laddr != nil {
		ip := laddr.IP
		if len(ip) != 0 && (ip.To16() == nil || !ipv6 && ip.To4() == nil) {
			cause := "non-IPv4 address"
			if ipv6 {
				cause = "non-IPv6 address"
			}
			return nil, &OpError{Op: "dial", Net: network, Source: laddr, Addr: raddr, Err: &AddrError{Err: cause, Addr: ip.String()}}
		}
	}
	host := raddr.IP.String()
	if len(raddr.IP) == 0 {
		host = "127.0.0.1"
	}
	if ipv6 && (len(raddr.IP) == 0 || raddr.IP.Equal(IPv4zero)) {
		host = "::1"
	}
	if raddr.Zone != "" {
		host += "%" + raddr.Zone
	}
	localHost, localPort := "", 0
	if laddr != nil {
		if len(laddr.IP) != 0 {
			localHost = laddr.IP.String()
			if ipv6 && laddr.IP.Equal(IPv4zero) {
				localHost = IPv6zero.String()
			}
			if ipv6 && laddr.Zone != "" {
				localHost += "%" + laddr.Zone
			}
		}
		localPort = laddr.Port
	}
	handle, localIP, local, cause, code, call, remoteIP := dart.DialUDP(host, raddr.Port, localHost, localPort)
	if cause != "" {
		return nil, &OpError{Op: "dial", Net: network, Source: laddr.opAddr(), Addr: raddr, Err: socketSyscallCause(call, cause, code)}
	}
	return newDialedUDPConn(
		&packet{handle: handle, network: network, local: udpAddrOf(localIP, local)},
		remoteIP,
		raddr.Port,
	), nil
}

// ListenUDP opens a datagram socket on an address. The network is udp, udp4,
// or udp6. A laddr of nil, or a port of 0, asks the system for a free port.
func ListenUDP(network string, laddr *UDPAddr) (*UDPConn, error) {
	switch network {
	case "udp", "udp4", "udp6":
	default:
		var addr Addr
		if laddr != nil {
			addr = laddr
		}
		return nil, &OpError{Op: "listen", Net: network, Addr: addr, Err: UnknownNetworkError(network)}
	}
	if laddr == nil {
		laddr = &UDPAddr{}
	}
	fail := func(err error) (*UDPConn, error) {
		return nil, &OpError{Op: "listen", Net: network, Addr: laddr, Err: err}
	}
	ip := laddr.IP
	if len(ip) != 0 && (ip.To16() == nil || network == "udp4" && ip.To4() == nil) {
		cause := "non-IPv4 address"
		if network == "udp6" || network == "udp" && len(ip) > IPv4len {
			cause = "non-IPv6 address"
		}
		return fail(&AddrError{Err: cause, Addr: ip.String()})
	}
	if laddr.Port < 0 || laddr.Port > 65535 {
		return fail(&os.SyscallError{Syscall: "bind", Err: syscall.EINVAL})
	}
	host := "0.0.0.0"
	if network == "udp6" {
		host = "::"
	}
	if len(ip) != 0 {
		host = ip.String()
	}
	local := udpAddrOf(host, laddr.Port)
	if laddr.Zone != "" && local.IP.To4() == nil {
		local.Zone = laddr.Zone
		host += "%" + laddr.Zone
	}
	handle, taken, cause, code := dart.ListenUDP(host, laddr.Port)
	if cause != "" {
		return fail(socketSyscallCause("bind", cause, code))
	}
	local.Port = taken
	return newUDPConn(&packet{handle: handle, network: network, local: local}), nil
}

// ListenMulticastUDP opens a datagram socket that reads the datagrams of a
// multicast group. The network is udp, udp4, or udp6.
//
// An ifi of nil joins the group on each interface of the system. Go marks the
// function as older than the direct use of ListenPacket with a join, and it
// keeps the loopback of the group open.
func ListenMulticastUDP(network string, ifi *Interface, gaddr *UDPAddr) (*UDPConn, error) {
	switch network {
	case "udp", "udp4", "udp6":
	default:
		return nil, &OpError{Op: "listen", Net: network, Addr: gaddr.opAddr(), Err: UnknownNetworkError(network)}
	}
	if gaddr == nil || len(gaddr.IP) == 0 {
		return nil, &OpError{Op: "listen", Net: network, Addr: gaddr.opAddr(), Err: errMissingAddress}
	}
	// The socket takes the port of the group and shares it, because each
	// member of a group reads the same port.
	handle, taken, cause := dart.ListenMulticastUDP("0.0.0.0", gaddr.Port)
	if cause != "" {
		return nil, errors.New("listen " + network + ": " + cause)
	}
	name := ""
	if ifi != nil {
		name = ifi.Name
	}
	if cause := dart.JoinMulticast(handle, gaddr.IP.String(), name); cause != "" {
		dart.PacketClose(handle)
		return nil, errors.New("listen " + network + ": " + cause)
	}
	return newUDPConn(&packet{
		network: network,
		handle:  handle,
		local:   udpAddrOf("0.0.0.0", taken),
	}), nil
}

// AddrPort gives the address in the form of net/netip. An address with no IP
// gives the zero value of netip.Addr with the port.
func (a *TCPAddr) AddrPort() netip.AddrPort {
	if a == nil {
		return netip.AddrPort{}
	}
	na, _ := netip.AddrFromSlice(a.IP)
	na = na.WithZone(a.Zone)
	return netip.AddrPortFrom(na, uint16(a.Port))
}

// TCPAddrFromAddrPort gives the TCP address of an address of net/netip.
func TCPAddrFromAddrPort(addr netip.AddrPort) *TCPAddr {
	return &TCPAddr{
		IP:   addr.Addr().AsSlice(),
		Zone: addr.Addr().Zone(),
		Port: int(addr.Port()),
	}
}

// AddrPort gives the address in the form of net/netip.
func (a *UDPAddr) AddrPort() netip.AddrPort {
	if a == nil {
		return netip.AddrPort{}
	}
	na, _ := netip.AddrFromSlice(a.IP)
	na = na.WithZone(a.Zone)
	return netip.AddrPortFrom(na, uint16(a.Port))
}

// UDPAddrFromAddrPort gives the UDP address of an address of net/netip.
func UDPAddrFromAddrPort(addr netip.AddrPort) *UDPAddr {
	return &UDPAddr{
		IP:   addr.Addr().AsSlice(),
		Zone: addr.Addr().Zone(),
		Port: int(addr.Port()),
	}
}
