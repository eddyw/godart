// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Mprotect(data []byte, prot int) error {
	if code := dart.SyscallMprotect(data, prot); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
