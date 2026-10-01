// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getfsstat(buf []Statfs_t, flags int) (int, error) {
	n, data, code := dart.SyscallGetfsstat(len(buf), flags)
	if code == -1 {
		return n, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return n, Errno(code)
	}
	if len(buf) == 0 {
		return n, nil
	}
	for i := 0; i < n; i++ {
		fields := data[i*2085 : (i+1)*2085]
		buf[i] = statfsFromData(fields)
	}
	return n, nil
}
