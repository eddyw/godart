// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

func statfsFromData(fields []int64) Statfs_t {
	return Statfs_t{
		Type: fields[0], Bsize: fields[1],
		Blocks: uint64(fields[2]), Bfree: uint64(fields[3]), Bavail: uint64(fields[4]),
		Files: uint64(fields[5]), Ffree: uint64(fields[6]),
		Fsid:    Fsid{X__val: [2]int32{int32(fields[7]), int32(fields[8])}},
		Namelen: fields[9], Frsize: fields[10], Flags: fields[11],
		Spare: [4]int64{fields[12], fields[13], fields[14], fields[15]},
	}
}
