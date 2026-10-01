// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Statfs(path string, stat *Statfs_t) error {
	data, code := dart.SyscallStatfs(path, stat != nil)
	return storeStatfs(data, code, stat)
}

func Fstatfs(fd int, stat *Statfs_t) error {
	data, code := dart.SyscallFstatfs(fd, stat != nil)
	return storeStatfs(data, code, stat)
}

func storeStatfs(data []int64, code int, stat *Statfs_t) error {
	if code == -1 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	*stat = statfsFromData(data)
	return nil
}

func statfsFromData(fields []int64) Statfs_t {
	stat := Statfs_t{
		Bsize: uint32(fields[0]), Iosize: int32(fields[1]),
		Blocks: uint64(fields[2]), Bfree: uint64(fields[3]), Bavail: uint64(fields[4]),
		Files: uint64(fields[5]), Ffree: uint64(fields[6]),
		Fsid:  Fsid{Val: [2]int32{int32(fields[7]), int32(fields[8])}},
		Owner: uint32(fields[9]), Type: uint32(fields[10]), Flags: uint32(fields[11]), Fssubtype: uint32(fields[12]),
	}
	for j := range stat.Fstypename {
		stat.Fstypename[j] = int8(fields[13+j])
	}
	for j := range stat.Mntonname {
		stat.Mntonname[j] = int8(fields[29+j])
	}
	for j := range stat.Mntfromname {
		stat.Mntfromname[j] = int8(fields[1053+j])
	}
	for j := range stat.Reserved {
		stat.Reserved[j] = uint32(fields[2077+j])
	}
	return stat
}
