// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Mlock(data []byte) error {
	if code := dart.SyscallMlock(data); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func Munlock(data []byte) error {
	if code := dart.SyscallMunlock(data); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
