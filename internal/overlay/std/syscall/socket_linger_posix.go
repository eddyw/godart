// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func SetsockoptLinger(fd, level, opt int, value *Linger) error {
	var data []byte
	if value != nil {
		data = make([]byte, SizeofLinger)
		binary.NativeEndian.PutUint32(data[:4], uint32(value.Onoff))
		binary.NativeEndian.PutUint32(data[4:], uint32(value.Linger))
	}
	if code := dart.SyscallSetsockoptIPMreq(fd, level, opt, data); code != 0 {
		return Errno(code)
	}
	return nil
}
