// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func Accept(fd int) (nfd int, sa Sockaddr, err error) {
	address := make([]byte, SizeofSockaddrAny)
	nfd, length, code := dart.SyscallAccept(fd, address)
	if code != 0 {
		return nfd, nil, Errno(code)
	}
	if length > SizeofSockaddrAny {
		panic("RawSockaddrAny too small")
	}
	sa, err = sockaddrFromBytes(address)
	if err != nil {
		Close(nfd)
		nfd = 0
	}
	return
}
