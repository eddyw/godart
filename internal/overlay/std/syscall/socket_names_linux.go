// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "encoding/binary"

func sockaddrFromBytes(data []byte) (Sockaddr, error) {
	switch int(binary.NativeEndian.Uint16(data)) {
	case AF_UNIX:
		path := data[2:SizeofSockaddrUnix]
		if path[0] == 0 {
			path[0] = '@'
		}
		n := 0
		for n < len(path) && path[n] != 0 {
			n++
		}
		return &SockaddrUnix{Name: string(path[:n])}, nil
	case AF_NETLINK:
		return &SockaddrNetlink{Family: binary.NativeEndian.Uint16(data), Pad: binary.NativeEndian.Uint16(data[2:]), Pid: binary.NativeEndian.Uint32(data[4:]), Groups: binary.NativeEndian.Uint32(data[8:])}, nil
	case AF_INET:
		sa := &SockaddrInet4{Port: int(data[2])<<8 + int(data[3])}
		copy(sa.Addr[:], data[4:8])
		return sa, nil
	case AF_INET6:
		sa := &SockaddrInet6{Port: int(data[2])<<8 + int(data[3]), ZoneId: binary.NativeEndian.Uint32(data[24:28])}
		copy(sa.Addr[:], data[8:24])
		return sa, nil
	}
	return nil, EAFNOSUPPORT
}
