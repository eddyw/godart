// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func GetsockoptIPv6MTUInfo(fd, level, opt int) (*IPv6MTUInfo, error) {
	var value IPv6MTUInfo
	data := make([]byte, SizeofIPv6MTUInfo)
	code := dart.SyscallGetsockoptICMPv6Filter(fd, level, opt, data)
	value.Addr.Family = binary.NativeEndian.Uint16(data[:2])
	value.Addr.Port = binary.NativeEndian.Uint16(data[2:4])
	value.Addr.Flowinfo = binary.NativeEndian.Uint32(data[4:8])
	copy(value.Addr.Addr[:], data[8:24])
	value.Addr.Scope_id = binary.NativeEndian.Uint32(data[24:28])
	value.Mtu = binary.NativeEndian.Uint32(data[28:])
	if code != 0 {
		return &value, syscallErrno(code)
	}
	return &value, nil
}
