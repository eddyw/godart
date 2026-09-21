// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package os

import "syscall"

func nativeSameFile(first, second *fileStat) bool {
	a := first.sys.(*syscall.Stat_t)
	b := second.sys.(*syscall.Stat_t)
	return a.Dev == b.Dev && a.Ino == b.Ino
}
