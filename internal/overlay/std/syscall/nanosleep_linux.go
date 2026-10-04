// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func nanosleepData(value *Timespec) []byte {
	if value == nil {
		return nil
	}
	data := make([]byte, 16)
	binary.NativeEndian.PutUint64(data, uint64(value.Sec))
	binary.NativeEndian.PutUint64(data[8:], uint64(value.Nsec))
	return data
}

func Nanosleep(request, leftover *Timespec) error {
	input, output := nanosleepData(request), nanosleepData(leftover)
	if request == leftover {
		output = input
	}
	code := dart.SyscallNanosleep(input, output)
	if leftover != nil {
		leftover.Sec = int64(binary.NativeEndian.Uint64(output))
		leftover.Nsec = int64(binary.NativeEndian.Uint64(output[8:]))
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}
