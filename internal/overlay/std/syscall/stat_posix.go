// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Stat(path string, stat *Stat_t) (err error) {
	data, errno := dart.NativeStat(path, true)
	return storeStat(data, errno, stat)
}

func Lstat(path string, stat *Stat_t) (err error) {
	data, errno := dart.NativeStat(path, false)
	return storeStat(data, errno, stat)
}

func Fstat(fd int, stat *Stat_t) (err error) {
	data, errno := dart.NativeFstat(fd)
	return storeStat(data, errno, stat)
}

func storeStat(data []int64, errno int, stat *Stat_t) error {
	if errno != 0 {
		return Errno(errno)
	}
	if stat == nil {
		return EFAULT
	}
	*stat = statFromData(data)
	return nil
}
