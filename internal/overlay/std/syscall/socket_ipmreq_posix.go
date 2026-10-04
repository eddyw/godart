// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func GetsockoptIPMreq(fd, level, opt int) (*IPMreq, error) {
	var value IPMreq
	data := make([]byte, SizeofIPMreq)
	code := dart.SyscallGetsockoptIPMreq(fd, level, opt, data)
	copy(value.Multiaddr[:], data[:4])
	copy(value.Interface[:], data[4:])
	if code != 0 {
		return &value, Errno(code)
	}
	return &value, nil
}

func SetsockoptIPMreq(fd, level, opt int, value *IPMreq) error {
	var data []byte
	if value != nil {
		data = make([]byte, SizeofIPMreq)
		copy(data[:4], value.Multiaddr[:])
		copy(data[4:], value.Interface[:])
	}
	if code := dart.SyscallSetsockoptIPMreq(fd, level, opt, data); code != 0 {
		return Errno(code)
	}
	return nil
}
