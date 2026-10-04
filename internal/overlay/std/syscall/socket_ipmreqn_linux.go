// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func GetsockoptIPMreqn(fd, level, opt int) (*IPMreqn, error) {
	var value IPMreqn
	data := make([]byte, SizeofIPMreqn)
	code := dart.SyscallGetsockoptIPMreqn(fd, level, opt, data)
	copy(value.Multiaddr[:], data[:4])
	copy(value.Address[:], data[4:8])
	value.Ifindex = int32(binary.NativeEndian.Uint32(data[8:]))
	if code != 0 {
		return &value, Errno(code)
	}
	return &value, nil
}

func SetsockoptIPMreqn(fd, level, opt int, value *IPMreqn) error {
	var data []byte
	if value != nil {
		data = make([]byte, SizeofIPMreqn)
		copy(data[:4], value.Multiaddr[:])
		copy(data[4:8], value.Address[:])
		binary.NativeEndian.PutUint32(data[8:], uint32(value.Ifindex))
	}
	if code := dart.SyscallSetsockoptIPMreqn(fd, level, opt, data); code != 0 {
		return Errno(code)
	}
	return nil
}
