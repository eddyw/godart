// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getpriority(which, who int) (int, error) {
	value, code := dart.Getpriority(which, who)
	if code == -1 {
		return value, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return value, Errno(code)
	}
	return value, nil
}

func Setpriority(which, who, priority int) error {
	code := dart.Setpriority(which, who, priority)
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}
