// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"sync"
)

// NetlinkRouteRequest represents a request message to receive routing
// and link states from the kernel.
type NetlinkRouteRequest struct {
	Header NlMsghdr
	Data   RtGenmsg
}

func (rr *NetlinkRouteRequest) toWireFormat() []byte {
	b := make([]byte, rr.Header.Len)
	binary.NativeEndian.PutUint32(b, rr.Header.Len)
	binary.NativeEndian.PutUint16(b[4:], rr.Header.Type)
	binary.NativeEndian.PutUint16(b[6:], rr.Header.Flags)
	binary.NativeEndian.PutUint32(b[8:], rr.Header.Seq)
	binary.NativeEndian.PutUint32(b[12:], rr.Header.Pid)
	b[16] = rr.Data.Family
	return b
}

func newNetlinkRouteRequest(proto, seq, family int) []byte {
	rr := &NetlinkRouteRequest{}
	rr.Header.Len = uint32(NLMSG_HDRLEN + SizeofRtGenmsg)
	rr.Header.Type = uint16(proto)
	rr.Header.Flags = NLM_F_DUMP | NLM_F_REQUEST
	rr.Header.Seq = uint32(seq)
	rr.Data.Family = uint8(family)
	return rr.toWireFormat()
}

var pageBufPool = &sync.Pool{New: func() any {
	b := make([]byte, Getpagesize())
	return &b
}}

// NetlinkRIB returns routing information base, as known as RIB, which
// consists of network facility information, states and parameters.
func NetlinkRIB(proto, family int) ([]byte, error) {
	s, err := Socket(AF_NETLINK, SOCK_RAW|SOCK_CLOEXEC, NETLINK_ROUTE)
	if err != nil {
		return nil, err
	}
	defer Close(s)
	sa := &SockaddrNetlink{Family: AF_NETLINK}
	if err := Bind(s, sa); err != nil {
		return nil, err
	}
	wb := newNetlinkRouteRequest(proto, 1, family)
	if err := Sendto(s, wb, 0, sa); err != nil {
		return nil, err
	}
	lsa, err := Getsockname(s)
	if err != nil {
		return nil, err
	}
	lsanl, ok := lsa.(*SockaddrNetlink)
	if !ok {
		return nil, EINVAL
	}
	var tab []byte

	rbNew := pageBufPool.Get().(*[]byte)
	defer pageBufPool.Put(rbNew)
done:
	for {
		rb := *rbNew
		nr, _, err := Recvfrom(s, rb, 0)
		if err != nil {
			return nil, err
		}
		if nr < NLMSG_HDRLEN {
			return nil, EINVAL
		}
		rb = rb[:nr]
		tab = append(tab, rb...)
		msgs, err := ParseNetlinkMessage(rb)
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if m.Header.Seq != 1 || m.Header.Pid != lsanl.Pid {
				return nil, EINVAL
			}
			if m.Header.Type == NLMSG_DONE {
				break done
			}
			if m.Header.Type == NLMSG_ERROR {
				return nil, EINVAL
			}
		}
	}
	return tab, nil
}

type NetlinkMessage struct {
	Header NlMsghdr
	Data   []byte
}

func ParseNetlinkMessage(data []byte) ([]NetlinkMessage, error) {
	var messages []NetlinkMessage
	for len(data) >= NLMSG_HDRLEN {
		header := NlMsghdr{
			Len:   binary.NativeEndian.Uint32(data),
			Type:  binary.NativeEndian.Uint16(data[4:]),
			Flags: binary.NativeEndian.Uint16(data[6:]),
			Seq:   binary.NativeEndian.Uint32(data[8:]),
			Pid:   binary.NativeEndian.Uint32(data[12:]),
		}
		length := (int(header.Len) + NLMSG_ALIGNTO - 1) & ^(NLMSG_ALIGNTO - 1)
		if int(header.Len) < NLMSG_HDRLEN || length > len(data) {
			return nil, EINVAL
		}
		messages = append(messages, NetlinkMessage{
			Header: header,
			Data:   data[NLMSG_HDRLEN:int(header.Len)],
		})
		data = data[length:]
	}
	return messages, nil
}

type NetlinkRouteAttr struct {
	Attr  RtAttr
	Value []byte
}

func ParseNetlinkRouteAttr(message *NetlinkMessage) ([]NetlinkRouteAttr, error) {
	var data []byte
	switch message.Header.Type {
	case RTM_NEWLINK, RTM_DELLINK:
		data = message.Data[SizeofIfInfomsg:]
	case RTM_NEWADDR, RTM_DELADDR:
		data = message.Data[SizeofIfAddrmsg:]
	case RTM_NEWROUTE, RTM_DELROUTE:
		data = message.Data[SizeofRtMsg:]
	default:
		return nil, EINVAL
	}
	var attributes []NetlinkRouteAttr
	for len(data) >= SizeofRtAttr {
		header := RtAttr{
			Len:  binary.NativeEndian.Uint16(data),
			Type: binary.NativeEndian.Uint16(data[2:]),
		}
		if int(header.Len) < SizeofRtAttr || int(header.Len) > len(data) {
			return nil, EINVAL
		}
		attributes = append(attributes, NetlinkRouteAttr{
			Attr:  header,
			Value: data[SizeofRtAttr:int(header.Len)],
		})
		length := (int(header.Len) + RTA_ALIGNTO - 1) & ^(RTA_ALIGNTO - 1)
		// The SDK preserves a slice panic when the aligned length exceeds len(data).
		data = data[length:]
	}
	return attributes, nil
}
