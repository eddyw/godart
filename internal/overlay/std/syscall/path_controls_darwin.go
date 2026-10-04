// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "syscall/dart"

func Revoke(path string) error {
	if _, err := BytePtrFromString(path); err != nil {
		return err
	}
	if code := dart.SyscallRevoke(path); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func Undelete(path string) error {
	if _, err := BytePtrFromString(path); err != nil {
		return err
	}
	if code := dart.SyscallUndelete(path); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
