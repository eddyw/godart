// Copyright 2009,2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Sysctl(name string) (string, error) {
	value, code := dart.SyscallSysctl(name)
	if code == -1 {
		return "", errorspkg.ErrUnsupported
	}
	if code != 0 {
		return "", Errno(code)
	}
	return value, nil
}

func SysctlUint32(name string) (uint32, error) {
	value, code := dart.SyscallSysctlUint32(name)
	if code == -1 {
		return 0, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return 0, Errno(code)
	}
	return value, nil
}
