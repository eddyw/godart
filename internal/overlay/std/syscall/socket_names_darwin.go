// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "encoding/binary"

func sockaddrFromBytes(data []byte) (Sockaddr, error) {
	switch int(data[1]) {
	case AF_UNIX:
		length := int(data[0])
		if length < 2 || length > SizeofSockaddrUnix {
			return nil, EINVAL
		}
		path := data[2:length]
		n := 0
		for n < len(path) && path[n] != 0 {
			n++
		}
		return &SockaddrUnix{Name: string(path[:n])}, nil
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
