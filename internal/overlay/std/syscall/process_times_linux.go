// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func Gettid() int { return dart.SyscallGettid() }

func Times(tms *Tms) (uintptr, error) {
	var data []byte
	if tms != nil {
		data = make([]byte, 32)
		binary.NativeEndian.PutUint64(data, uint64(tms.Utime))
		binary.NativeEndian.PutUint64(data[8:], uint64(tms.Stime))
		binary.NativeEndian.PutUint64(data[16:], uint64(tms.Cutime))
		binary.NativeEndian.PutUint64(data[24:], uint64(tms.Cstime))
	}
	ticks, code := dart.SyscallTimes(data)
	if tms != nil {
		tms.Utime = int64(binary.NativeEndian.Uint64(data))
		tms.Stime = int64(binary.NativeEndian.Uint64(data[8:]))
		tms.Cutime = int64(binary.NativeEndian.Uint64(data[16:]))
		tms.Cstime = int64(binary.NativeEndian.Uint64(data[24:]))
	}
	if code != 0 {
		return uintptr(ticks), Errno(code)
	}
	return uintptr(ticks), nil
}
