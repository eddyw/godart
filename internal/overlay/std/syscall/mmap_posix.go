// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Mmap(fd int, offset int64, length int, prot, flags int) ([]byte, error) {
	if length <= 0 {
		return nil, EINVAL
	}
	data, code := dart.SyscallMmap(fd, offset, length, prot, flags)
	if code != 0 {
		return nil, syscallErrno(code)
	}
	return data, nil
}

func Munmap(data []byte) error {
	if len(data) == 0 || len(data) != cap(data) {
		return EINVAL
	}
	if code := dart.SyscallMunmap(data); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
