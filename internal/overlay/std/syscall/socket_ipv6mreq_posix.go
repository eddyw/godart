// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func GetsockoptIPv6Mreq(fd, level, opt int) (*IPv6Mreq, error) {
	var value IPv6Mreq
	data := make([]byte, SizeofIPv6Mreq)
	code := dart.SyscallGetsockoptIPv6Mreq(fd, level, opt, data)
	copy(value.Multiaddr[:], data[:16])
	value.Interface = binary.NativeEndian.Uint32(data[16:])
	if code != 0 {
		return &value, Errno(code)
	}
	return &value, nil
}

func SetsockoptIPv6Mreq(fd, level, opt int, value *IPv6Mreq) error {
	var data []byte
	if value != nil {
		data = make([]byte, SizeofIPv6Mreq)
		copy(data[:16], value.Multiaddr[:])
		binary.NativeEndian.PutUint32(data[16:], value.Interface)
	}
	if code := dart.SyscallSetsockoptIPv6Mreq(fd, level, opt, data); code != 0 {
		return Errno(code)
	}
	return nil
}
