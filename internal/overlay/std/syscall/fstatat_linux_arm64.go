// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import "syscall/dart"

func Fstatat(fd int, path string, stat *Stat_t, flags int) error {
	return fstatat(fd, path, stat, flags)
}

func fstatat(fd int, path string, stat *Stat_t, flags int) error {
	data, errno := dart.NativeFstatat(fd, path, stat != nil, flags)
	return storeStat(data, errno, stat)
}
