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
