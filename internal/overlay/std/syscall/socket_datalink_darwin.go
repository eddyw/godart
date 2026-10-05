// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "encoding/binary"

type SockaddrDatalink struct {
	Len    uint8
	Family uint8
	Index  uint16
	Type   uint8
	Nlen   uint8
	Alen   uint8
	Slen   uint8
	Data   [12]int8
	raw    RawSockaddrDatalink
}

func (sa *SockaddrDatalink) sockaddrBytes() ([]byte, error) {
	if sa.Index == 0 {
		return nil, EINVAL
	}
	sa.raw.Len = sa.Len
	sa.raw.Family = AF_LINK
	sa.raw.Index = sa.Index
	sa.raw.Type = sa.Type
	sa.raw.Nlen = sa.Nlen
	sa.raw.Alen = sa.Alen
	sa.raw.Slen = sa.Slen
	sa.raw.Data = sa.Data
	data := make([]byte, SizeofSockaddrDatalink)
	data[0], data[1] = sa.raw.Len, sa.raw.Family
	binary.NativeEndian.PutUint16(data[2:4], sa.raw.Index)
	data[4], data[5], data[6], data[7] = sa.raw.Type, sa.raw.Nlen, sa.raw.Alen, sa.raw.Slen
	for i, value := range sa.raw.Data {
		data[8+i] = byte(value)
	}
	return data, nil
}
