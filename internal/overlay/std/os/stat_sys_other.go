// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (!(linux || darwin) || !(amd64 || arm64))

package os

func nativeFileSys(data []int64) any { return nil }
func nativeSameFile(first, second *fileStat) bool {
	return first.device == second.device && first.inode == second.inode
}
