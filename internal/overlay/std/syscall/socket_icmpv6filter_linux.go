// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func GetsockoptICMPv6Filter(fd, level, opt int) (*ICMPv6Filter, error) {
	var value ICMPv6Filter
	data := make([]byte, SizeofICMPv6Filter)
	code := dart.SyscallGetsockoptICMPv6Filter(fd, level, opt, data)
	for i := range value.Data {
		value.Data[i] = binary.NativeEndian.Uint32(data[i*4:])
	}
	if code != 0 {
		return &value, syscallErrno(code)
	}
	return &value, nil
}

func SetsockoptICMPv6Filter(fd, level, opt int, value *ICMPv6Filter) error {
	var data []byte
	if value != nil {
		data = make([]byte, SizeofICMPv6Filter)
		for i, word := range value.Data {
			binary.NativeEndian.PutUint32(data[i*4:], word)
		}
	}
	if code := dart.SyscallSetsockoptICMPv6Filter(fd, level, opt, data); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
