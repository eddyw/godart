// Copyright 2009,2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (darwin || linux)

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Mkfifo(path string, mode uint32) error {
	code := dart.SyscallMkfifo(path, mode)
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}
