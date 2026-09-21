// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (amd64 || arm64)

package os

import "syscall"

func nativeModTime(stat *syscall.Stat_t) (int64, int64) {
	return stat.Mtim.Sec, stat.Mtim.Nsec
}
