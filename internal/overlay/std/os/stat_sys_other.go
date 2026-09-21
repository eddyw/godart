// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (!(linux || darwin) || !(amd64 || arm64))

package os

import (
	"syscall"
	"syscall/dart"
)

func nativeSameFile(first, second *fileStat) bool {
	return first.device == second.device && first.inode == second.inode
}

func nativeStat(name string, follow bool) (FileInfo, error) {
	data, errno := dart.NativeStat(name, follow)
	op := "stat"
	if !follow {
		op = "lstat"
	}
	return nativeFileInfo(name, op, data, errno)
}
func nativeFstat(name string, fd int) (FileInfo, error) {
	data, errno := dart.NativeFstat(fd)
	return nativeFileInfo(name, "stat", data, errno)
}
func nativeFileInfo(name, op string, data []int64, errno int) (FileInfo, error) {
	if errno != 0 {
		return nil, &PathError{Op: op, Path: name, Err: syscall.Errno(errno)}
	}
	return &fileStat{name: base(name), size: data[0], mode: nativeFileMode(uint32(data[1])), seconds: data[2], nanos: data[3], device: uint64(data[4]), inode: uint64(data[5])}, nil
}
