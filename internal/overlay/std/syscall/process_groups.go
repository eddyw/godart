// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Setpgid(pid, pgid int) error {
	code := dart.Setpgid(pid, pgid)
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}

func Setsid() (int, error) {
	pid, code := dart.Setsid()
	if code == -1 {
		return pid, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return pid, Errno(code)
	}
	return pid, nil
}
