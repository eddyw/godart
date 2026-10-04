// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "encoding/binary"

type SockaddrNetlink struct {
	Family uint16
	Pad    uint16
	Pid    uint32
	Groups uint32
	raw    RawSockaddrNetlink
}

func (sa *SockaddrNetlink) sockaddrBytes() ([]byte, error) {
	sa.raw.Family = AF_NETLINK
	sa.raw.Pad = sa.Pad
	sa.raw.Pid = sa.Pid
	sa.raw.Groups = sa.Groups
	data := make([]byte, SizeofSockaddrNetlink)
	binary.NativeEndian.PutUint16(data, sa.raw.Family)
	binary.NativeEndian.PutUint16(data[2:], sa.raw.Pad)
	binary.NativeEndian.PutUint32(data[4:], sa.raw.Pid)
	binary.NativeEndian.PutUint32(data[8:], sa.raw.Groups)
	return data, nil
}
