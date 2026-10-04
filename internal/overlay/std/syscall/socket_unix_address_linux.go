// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "encoding/binary"

func (sa *SockaddrUnix) sockaddrBytes() ([]byte, error) {
	name := sa.Name
	n := len(name)
	if n > len(sa.raw.Path) {
		return nil, EINVAL
	}
	abstract := n > 0 && (name[0] == '@' || name[0] == '\x00')
	if n == len(sa.raw.Path) && !abstract {
		return nil, EINVAL
	}
	sa.raw.Family = AF_UNIX
	// The SDK retains cached path bytes when Name becomes shorter.
	for i := 0; i < n; i++ {
		sa.raw.Path[i] = int8(name[i])
	}
	length := 2 + n
	if abstract {
		sa.raw.Path[0] = 0
	} else if n > 0 {
		length++
	}
	data := make([]byte, length)
	binary.NativeEndian.PutUint16(data, sa.raw.Family)
	for i := 2; i < len(data); i++ {
		data[i] = byte(sa.raw.Path[i-2])
	}
	return data, nil
}
