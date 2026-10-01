// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package os

import (
	"syscall"
	"syscall/dart"
)

func nativeSameFile(first, second *fileStat) bool {
	a := first.sys.(*syscall.Stat_t)
	b := second.sys.(*syscall.Stat_t)
	return a.Dev == b.Dev && a.Ino == b.Ino
}

func nativeStat(name string, follow bool) (FileInfo, error) {
	stat := new(syscall.Stat_t)
	var err error
	op := "stat"
	if follow {
		err = syscall.Stat(name, stat)
	} else {
		op = "lstat"
		err = syscall.Lstat(name, stat)
	}
	if err != nil {
		return nil, &PathError{Op: op, Path: name, Err: err}
	}
	return nativeFileInfo(name, stat), nil
}

func nativeFstat(name string, fd int) (FileInfo, error) {
	stat := new(syscall.Stat_t)
	if err := syscall.Fstat(fd, stat); err != nil {
		return nil, &PathError{Op: "stat", Path: name, Err: err}
	}
	return nativeFileInfo(name, stat), nil
}

func nativeFileInfo(name string, stat *syscall.Stat_t) FileInfo {
	seconds, nanos := nativeModTime(stat)
	return &fileStat{name: base(name), size: stat.Size, mode: nativeFileMode(uint32(stat.Mode)), seconds: seconds, nanos: nanos, device: uint64(stat.Dev), inode: uint64(stat.Ino), sys: stat}
}

func lstatatWithName(parent int, original, name string) (FileInfo, error) {
	data, code := dart.NativeLstatat(parent, name)
	if code != 0 {
		return nil, syscall.Errno(code)
	}
	stat := directoryStatFromData(data)
	return nativeFileInfo(original, &stat), nil
}
