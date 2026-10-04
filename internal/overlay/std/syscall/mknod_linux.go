// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func Mknod(path string, mode uint32, dev int) error {
	return Mknodat(-100, path, mode, dev)
}

func Mknodat(dirfd int, path string, mode uint32, dev int) error {
	if code := dart.SyscallMknodat(dirfd, path, mode, dev); code != 0 {
		return Errno(code)
	}
	return nil
}
