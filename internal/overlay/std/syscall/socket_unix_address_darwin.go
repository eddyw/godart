// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

func (sa *SockaddrUnix) sockaddrBytes() ([]byte, error) {
	name := sa.Name
	n := len(name)
	if n >= len(sa.raw.Path) || n == 0 {
		return nil, EINVAL
	}
	sa.raw.Len = byte(3 + n)
	sa.raw.Family = AF_UNIX
	// The SDK retains cached path bytes when Name becomes shorter.
	for i := 0; i < n; i++ {
		sa.raw.Path[i] = int8(name[i])
	}
	data := make([]byte, int(sa.raw.Len))
	data[0], data[1] = sa.raw.Len, sa.raw.Family
	for i := 2; i < len(data); i++ {
		data[i] = byte(sa.raw.Path[i-2])
	}
	return data, nil
}
