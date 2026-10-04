// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func Recvfrom(fd int, data []byte, flags int) (n int, from Sockaddr, err error) {
	address := make([]byte, SizeofSockaddrAny)
	n, code := dart.SyscallRecvfrom(fd, data, flags, address)
	if code != 0 {
		return n, nil, syscallErrno(code)
	}
	if binary.NativeEndian.Uint16(address) != AF_UNSPEC {
		from, err = sockaddrFromBytes(address)
	}
	return
}
