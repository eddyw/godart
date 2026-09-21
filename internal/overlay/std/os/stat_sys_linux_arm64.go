// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import "syscall"

func nativeFileSys(data []int64) any {
	return &syscall.Stat_t{
		Size: data[0], Mode: uint32(data[1]),
		Mtim: syscall.Timespec{Sec: data[2], Nsec: data[3]},
		Dev:  uint64(data[4]), Ino: uint64(data[5]),
		Nlink: uint32(data[6]), Uid: uint32(data[7]), Gid: uint32(data[8]),
		Rdev: uint64(data[9]), Blksize: int32(data[10]), Blocks: data[11],
		Atim: syscall.Timespec{Sec: data[12], Nsec: data[13]},
		Ctim: syscall.Timespec{Sec: data[14], Nsec: data[15]},
	}
}
