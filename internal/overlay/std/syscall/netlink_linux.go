// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "encoding/binary"

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
