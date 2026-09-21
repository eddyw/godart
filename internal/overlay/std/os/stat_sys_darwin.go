// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import "syscall"

func nativeFileSys(data []int64) any {
	return &syscall.Stat_t{
		Size: data[0], Mode: uint16(data[1]),
		Mtimespec: syscall.Timespec{Sec: data[2], Nsec: data[3]},
		Dev:       int32(data[4]), Ino: uint64(data[5]),
		Nlink: uint16(data[6]), Uid: uint32(data[7]), Gid: uint32(data[8]),
		Rdev: int32(data[9]), Blksize: int32(data[10]), Blocks: data[11],
		Atimespec:     syscall.Timespec{Sec: data[12], Nsec: data[13]},
		Ctimespec:     syscall.Timespec{Sec: data[14], Nsec: data[15]},
		Birthtimespec: syscall.Timespec{Sec: data[16], Nsec: data[17]},
		Flags:         uint32(data[18]), Gen: uint32(data[19]),
	}
}
