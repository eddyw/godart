// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func Pipe2(p []int, flags int) error {
	if len(p) != 2 {
		return EINVAL
	}
	first, second, code := dart.SyscallPipe2(flags)
	if code != 0 {
		return Errno(code)
	}
	p[0], p[1] = first, second
	return nil
}

func Splice(rfd int, roff *int64, wfd int, woff *int64, length, flags int) (int64, error) {
	var read, write []byte
	if roff != nil {
		read = make([]byte, 8)
		binary.NativeEndian.PutUint64(read, uint64(*roff))
	}
	if woff != nil {
		if woff == roff {
			write = read
		} else {
			write = make([]byte, 8)
			binary.NativeEndian.PutUint64(write, uint64(*woff))
		}
	}
	n, code := dart.SyscallSplice(rfd, read, wfd, write, length, flags)
	if roff != nil {
		*roff = int64(binary.NativeEndian.Uint64(read))
	}
	if woff != nil {
		*woff = int64(binary.NativeEndian.Uint64(write))
	}
	if code != 0 {
		return n, Errno(code)
	}
	return n, nil
}

func Tee(rfd, wfd, length, flags int) (int64, error) {
	n, code := dart.SyscallTee(rfd, wfd, length, flags)
	if code != 0 {
		return n, Errno(code)
	}
	return n, nil
}
