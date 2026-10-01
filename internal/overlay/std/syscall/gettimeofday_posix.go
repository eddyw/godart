// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin)

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Gettimeofday(tv *Timeval) error {
	sec, usec, code := dart.SyscallGettimeofday(tv != nil)
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	if tv != nil {
		value := setTimeval(sec, usec)
		tv.Sec, tv.Usec = value.Sec, value.Usec
	}
	return nil
}
