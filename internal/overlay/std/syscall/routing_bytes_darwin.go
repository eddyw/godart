// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import (
	"bytes"
	"encoding/binary"
)

const minRoutingSockaddrLen = 4

func rsaAlignOf(salen int) int {
	if salen == 0 {
		return 4
	}
	return (salen + 3) & ^3
}

// parseSockaddrLink parses b as a datalink socket address.
func parseSockaddrLink(b []byte) (*SockaddrDatalink, error) {
	if len(b) < 8 {
		return nil, EINVAL
	}
	sa, _, err := parseLinkLayerAddr(b[4:])
	if err != nil {
		return nil, err
	}
	rsa := RawSockaddrDatalink{Len: b[0], Family: b[1], Index: binary.NativeEndian.Uint16(b[2:4])}
	sa.Len = rsa.Len
	sa.Family = rsa.Family
	sa.Index = rsa.Index
	return sa, nil
}

// parseLinkLayerAddr parses b as a datalink socket address in
// conventional BSD kernel form.
func parseLinkLayerAddr(b []byte) (*SockaddrDatalink, int, error) {
	// The encoding looks like the following:
	// +----------------------------+
	// | Type             (1 octet) |
	// +----------------------------+
	// | Name length      (1 octet) |
	// +----------------------------+
	// | Address length   (1 octet) |
	// +----------------------------+
	// | Selector length  (1 octet) |
	// +----------------------------+
	// | Data            (variable) |
	// +----------------------------+
	type linkLayerAddr struct {
		Type byte
		Nlen byte
		Alen byte
		Slen byte
	}
	lla := linkLayerAddr{Type: b[0], Nlen: b[1], Alen: b[2], Slen: b[3]}
	l := 4 + int(lla.Nlen) + int(lla.Alen) + int(lla.Slen)
	if len(b) < l {
		return nil, 0, EINVAL
	}
	b = b[4:]
	sa := &SockaddrDatalink{Type: lla.Type, Nlen: lla.Nlen, Alen: lla.Alen, Slen: lla.Slen}
	for i := 0; len(sa.Data) > i && i < l-4; i++ {
		sa.Data[i] = int8(b[i])
	}
	return sa, rsaAlignOf(l), nil
}

// parseSockaddrInet parses b as an internet socket address.
func parseSockaddrInet(b []byte, family byte) (Sockaddr, error) {
	switch family {
	case AF_INET:
		if len(b) < SizeofSockaddrInet4 {
			return nil, EINVAL
		}
		return sockaddrFromBytes(b)
	case AF_INET6:
		if len(b) < SizeofSockaddrInet6 {
			return nil, EINVAL
		}
		return sockaddrFromBytes(b)
	default:
		return nil, EINVAL
	}
}

const (
	offsetofInet4 = 4
	offsetofInet6 = 8
)

// parseNetworkLayerAddr parses b as an internet socket address in
// conventional BSD kernel form.
func parseNetworkLayerAddr(b []byte, family byte) (Sockaddr, error) {
	// The encoding looks similar to the NLRI encoding.
	// +----------------------------+
	// | Length           (1 octet) |
	// +----------------------------+
	// | Address prefix  (variable) |
	// +----------------------------+
	//
	// The differences between the kernel form and the NLRI
	// encoding are:
	//
	// - The length field of the kernel form indicates the prefix
	//   length in bytes, not in bits
	//
	// - In the kernel form, zero value of the length field
	//   doesn't mean 0.0.0.0/0 or ::/0
	//
	// - The kernel form appends leading bytes to the prefix field
	//   to make the <length, prefix> tuple to be conformed with
	//   the routing message boundary
	l := int(rsaAlignOf(int(b[0])))
	if len(b) < l {
		return nil, EINVAL
	}
	// Don't reorder case expressions.
	// The case expressions for IPv6 must come first.
	switch {
	case b[0] == SizeofSockaddrInet6:
		sa := &SockaddrInet6{}
		copy(sa.Addr[:], b[offsetofInet6:])
		return sa, nil
	case family == AF_INET6:
		sa := &SockaddrInet6{}
		if l-1 < offsetofInet6 {
			copy(sa.Addr[:], b[1:l])
		} else {
			copy(sa.Addr[:], b[l-offsetofInet6:l])
		}
		return sa, nil
	case b[0] == SizeofSockaddrInet4:
		sa := &SockaddrInet4{}
		copy(sa.Addr[:], b[offsetofInet4:])
		return sa, nil
	default: // an old fashion, AF_UNSPEC or unknown means AF_INET
		sa := &SockaddrInet4{}
		if l-1 < offsetofInet4 {
			copy(sa.Addr[:], b[1:l])
		} else {
			copy(sa.Addr[:], b[l-offsetofInet4:l])
		}
		return sa, nil
	}
}

// RoutingMessage represents a routing message.
//
// Deprecated: Use golang.org/x/net/route instead.
type RoutingMessage interface {
	sockaddr() ([]Sockaddr, error)
}

const anyMessageLen = 4

type anyMessage struct {
	Msglen  uint16
	Version uint8
	Type    uint8
}

// RouteMessage represents a routing message containing routing
// entries.
//
// Deprecated: Use golang.org/x/net/route instead.
type RouteMessage struct {
	Header RtMsghdr
	Data   []byte
}

func (m *RouteMessage) sockaddr() ([]Sockaddr, error) {
	var sas [RTAX_MAX]Sockaddr
	b := m.Data[:]
	family := uint8(AF_UNSPEC)
	for i := uint(0); i < RTAX_MAX && len(b) >= minRoutingSockaddrLen; i++ {
		if m.Header.Addrs&(1<<i) == 0 {
			continue
		}
		rsa := RawSockaddr{Len: b[0], Family: b[1]}
		switch rsa.Family {
		case AF_LINK:
			sa, err := parseSockaddrLink(b)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(rsa.Len)):]
		case AF_INET, AF_INET6:
			sa, err := parseSockaddrInet(b, rsa.Family)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(rsa.Len)):]
			family = rsa.Family
		default:
			sa, err := parseNetworkLayerAddr(b, family)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(b[0])):]
		}
	}
	return sas[:], nil
}

// InterfaceMessage represents a routing message containing
// network interface entries.
//
// Deprecated: Use golang.org/x/net/route instead.
type InterfaceMessage struct {
	Header IfMsghdr
	Data   []byte
}

func (m *InterfaceMessage) sockaddr() ([]Sockaddr, error) {
	var sas [RTAX_MAX]Sockaddr
	if m.Header.Addrs&RTA_IFP == 0 {
		return nil, nil
	}
	sa, err := parseSockaddrLink(m.Data[:])
	if err != nil {
		return nil, err
	}
	sas[RTAX_IFP] = sa
	return sas[:], nil
}

// InterfaceAddrMessage represents a routing message containing
// network interface address entries.
//
// Deprecated: Use golang.org/x/net/route instead.
type InterfaceAddrMessage struct {
	Header IfaMsghdr
	Data   []byte
}

func (m *InterfaceAddrMessage) sockaddr() ([]Sockaddr, error) {
	var sas [RTAX_MAX]Sockaddr
	b := m.Data[:]
	family := uint8(AF_UNSPEC)
	for i := uint(0); i < RTAX_MAX && len(b) >= minRoutingSockaddrLen; i++ {
		if m.Header.Addrs&(1<<i) == 0 {
			continue
		}
		rsa := RawSockaddr{Len: b[0], Family: b[1]}
		switch rsa.Family {
		case AF_LINK:
			sa, err := parseSockaddrLink(b)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(rsa.Len)):]
		case AF_INET, AF_INET6:
			sa, err := parseSockaddrInet(b, rsa.Family)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(rsa.Len)):]
			family = rsa.Family
		default:
			sa, err := parseNetworkLayerAddr(b, family)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(b[0])):]
		}
	}
	return sas[:], nil
}

// ParseRoutingMessage parses b as routing messages and returns the
// slice containing the [RoutingMessage] interfaces.
//
// Deprecated: Use golang.org/x/net/route instead.
func ParseRoutingMessage(b []byte) (msgs []RoutingMessage, err error) {
	nmsgs, nskips := 0, 0
	for len(b) >= anyMessageLen {
		nmsgs++
		any := anyMessage{Msglen: binary.NativeEndian.Uint16(b), Version: b[2], Type: b[3]}
		if any.Version != RTM_VERSION {
			if any.Msglen == 0 {
				return nil, ENOSYS
			}
			b = b[any.Msglen:]
			continue
		}
		if m := any.toRoutingMessage(b); m == nil {
			nskips++
		} else {
			msgs = append(msgs, m)
		}
		// A zero-length unknown frame cannot advance the original decoder.
		if any.Msglen == 0 {
			return nil, ENOSYS
		}
		b = b[any.Msglen:]
	}
	// We failed to parse any of the messages - version mismatch?
	if nmsgs != len(msgs)+nskips {
		return nil, EINVAL
	}
	return msgs, nil
}

// ParseRoutingSockaddr parses msg's payload as raw sockaddrs and
// returns the slice containing the [Sockaddr] interfaces.
//
// Deprecated: Use golang.org/x/net/route instead.
func ParseRoutingSockaddr(msg RoutingMessage) ([]Sockaddr, error) {
	sas, err := msg.sockaddr()
	if err != nil {
		return nil, err
	}
	return sas, nil
}

func (any *anyMessage) toRoutingMessage(b []byte) RoutingMessage {
	switch any.Type {
	case RTM_ADD, RTM_DELETE, RTM_CHANGE, RTM_GET, RTM_LOSING, RTM_REDIRECT, RTM_MISS, RTM_LOCK, RTM_RESOLVE:
		var header RtMsghdr
		binary.Read(bytes.NewReader(b[:SizeofRtMsghdr]), binary.NativeEndian, &header)
		return &RouteMessage{Header: header, Data: b[SizeofRtMsghdr:any.Msglen]}
	case RTM_IFINFO:
		var header IfMsghdr
		binary.Read(bytes.NewReader(b[:SizeofIfMsghdr]), binary.NativeEndian, &header)
		return &InterfaceMessage{Header: header, Data: b[SizeofIfMsghdr:any.Msglen]}
	case RTM_NEWADDR, RTM_DELADDR:
		var header IfaMsghdr
		binary.Read(bytes.NewReader(b[:SizeofIfaMsghdr]), binary.NativeEndian, &header)
		return &InterfaceAddrMessage{Header: header, Data: b[SizeofIfaMsghdr:any.Msglen]}
	case RTM_NEWMADDR2, RTM_DELMADDR:
		var header IfmaMsghdr2
		binary.Read(bytes.NewReader(b[:SizeofIfmaMsghdr2]), binary.NativeEndian, &header)
		return &InterfaceMulticastAddrMessage{Header: header, Data: b[SizeofIfmaMsghdr2:any.Msglen]}
	}
	return nil
}

// InterfaceMulticastAddrMessage represents a routing message
// containing network interface address entries.
//
// Deprecated: Use golang.org/x/net/route instead.
type InterfaceMulticastAddrMessage struct {
	Header IfmaMsghdr2
	Data   []byte
}

func (m *InterfaceMulticastAddrMessage) sockaddr() ([]Sockaddr, error) {
	var sas [RTAX_MAX]Sockaddr
	b := m.Data[:]
	for i := uint(0); i < RTAX_MAX && len(b) >= minRoutingSockaddrLen; i++ {
		if m.Header.Addrs&(1<<i) == 0 {
			continue
		}
		rsa := RawSockaddr{Len: b[0], Family: b[1]}
		switch rsa.Family {
		case AF_LINK:
			sa, err := parseSockaddrLink(b)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(rsa.Len)):]
		case AF_INET, AF_INET6:
			sa, err := parseSockaddrInet(b, rsa.Family)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[rsaAlignOf(int(rsa.Len)):]
		default:
			sa, l, err := parseLinkLayerAddr(b)
			if err != nil {
				return nil, err
			}
			sas[i] = sa
			b = b[l:]
		}
	}
	return sas[:], nil
}
