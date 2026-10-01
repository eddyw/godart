// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Uname(buf *Utsname) error {
	data, code := dart.SyscallUname(buf != nil)
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	fields := []*[65]int8{&buf.Sysname, &buf.Nodename, &buf.Release, &buf.Version, &buf.Machine, &buf.Domainname}
	for i, field := range fields {
		for j := range field {
			field[j] = int8(data[i*65+j])
		}
	}
	return nil
}
