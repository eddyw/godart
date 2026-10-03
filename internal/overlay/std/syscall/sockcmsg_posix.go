// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "encoding/binary"

func cmsgAlignOf(length int) int {
	return (length + cmsgAlignment - 1) & ^(cmsgAlignment - 1)
}

func CmsgLen(datalen int) int   { return cmsgAlignOf(SizeofCmsghdr) + datalen }
func CmsgSpace(datalen int) int { return cmsgAlignOf(SizeofCmsghdr) + cmsgAlignOf(datalen) }

type SocketControlMessage struct {
	Header Cmsghdr
	Data   []byte
}

func ParseSocketControlMessage(data []byte) ([]SocketControlMessage, error) {
	var messages []SocketControlMessage
	for offset := 0; offset+CmsgLen(0) <= len(data); {
		header, payload, err := socketControlMessageHeaderAndData(data[offset:])
		if err != nil {
			return nil, err
		}
		messages = append(messages, SocketControlMessage{Header: header, Data: payload})
		offset += cmsgAlignOf(int(header.Len))
	}
	return messages, nil
}

func socketControlMessageHeaderAndData(data []byte) (Cmsghdr, []byte, error) {
	var length uint64
	if SizeofCmsghdr == 16 {
		length = binary.NativeEndian.Uint64(data)
	} else {
		length = uint64(binary.NativeEndian.Uint32(data))
	}
	if length < SizeofCmsghdr || length > uint64(len(data)) {
		return Cmsghdr{}, nil, EINVAL
	}
	var header Cmsghdr
	header.SetLen(int(length))
	header.Level = int32(binary.NativeEndian.Uint32(data[SizeofCmsghdr-8:]))
	header.Type = int32(binary.NativeEndian.Uint32(data[SizeofCmsghdr-4:]))
	return header, data[cmsgAlignOf(SizeofCmsghdr):int(length)], nil
}

func UnixRights(fds ...int) []byte {
	data := make([]byte, CmsgSpace(4*len(fds)))
	var header Cmsghdr
	header.SetLen(CmsgLen(4 * len(fds)))
	if SizeofCmsghdr == 16 {
		binary.NativeEndian.PutUint64(data, uint64(header.Len))
	} else {
		binary.NativeEndian.PutUint32(data, uint32(header.Len))
	}
	binary.NativeEndian.PutUint32(data[SizeofCmsghdr-8:], uint32(SOL_SOCKET))
	binary.NativeEndian.PutUint32(data[SizeofCmsghdr-4:], uint32(SCM_RIGHTS))
	for i, fd := range fds {
		binary.NativeEndian.PutUint32(data[CmsgLen(0)+4*i:], uint32(fd))
	}
	return data
}

func ParseUnixRights(message *SocketControlMessage) ([]int, error) {
	if message.Header.Level != SOL_SOCKET || message.Header.Type != SCM_RIGHTS {
		return nil, EINVAL
	}
	fds := make([]int, len(message.Data)>>2)
	for i, j := 0, 0; i < len(message.Data); i += 4 {
		fds[j] = int(int32(binary.NativeEndian.Uint32(message.Data[i:])))
		j++
	}
	return fds, nil
}
