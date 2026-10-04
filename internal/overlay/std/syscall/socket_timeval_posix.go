// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"runtime"
	"syscall/dart"
)

func SetsockoptTimeval(fd, level, opt int, value *Timeval) error {
	var data []byte
	if value != nil {
		data = make([]byte, 16)
		binary.NativeEndian.PutUint64(data[:8], uint64(value.Sec))
		if runtime.GOOS == "darwin" {
			binary.NativeEndian.PutUint32(data[8:12], uint32(value.Usec))
		} else {
			binary.NativeEndian.PutUint64(data[8:], uint64(value.Usec))
		}
	}
	if code := dart.SyscallSetsockoptTimeval(fd, level, opt, data); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
