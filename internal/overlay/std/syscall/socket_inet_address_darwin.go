// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "encoding/binary"

func (sa *SockaddrInet4) sockaddrBytes() ([]byte, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, EINVAL
	}
	sa.raw.Len = SizeofSockaddrInet4
	sa.raw.Family = AF_INET
	data := make([]byte, SizeofSockaddrInet4)
	data[2], data[3] = byte(sa.Port>>8), byte(sa.Port)
	sa.raw.Port = binary.NativeEndian.Uint16(data[2:4])
	sa.raw.Addr = sa.Addr
	data[0], data[1] = sa.raw.Len, sa.raw.Family
	copy(data[4:8], sa.raw.Addr[:])
	for i, value := range sa.raw.Zero {
		data[8+i] = byte(value)
	}
	return data, nil
}

func (sa *SockaddrInet6) sockaddrBytes() ([]byte, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, EINVAL
	}
	sa.raw.Len = SizeofSockaddrInet6
	sa.raw.Family = AF_INET6
	data := make([]byte, SizeofSockaddrInet6)
	data[2], data[3] = byte(sa.Port>>8), byte(sa.Port)
	sa.raw.Port = binary.NativeEndian.Uint16(data[2:4])
	sa.raw.Addr = sa.Addr
	sa.raw.Scope_id = sa.ZoneId
	data[0], data[1] = sa.raw.Len, sa.raw.Family
	binary.NativeEndian.PutUint32(data[4:8], sa.raw.Flowinfo)
	copy(data[8:24], sa.raw.Addr[:])
	binary.NativeEndian.PutUint32(data[24:28], sa.raw.Scope_id)
	return data, nil
}
