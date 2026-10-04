// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func GetsockoptUcred(fd, level, opt int) (*Ucred, error) {
	pid, uid, gid, code := dart.SyscallGetsockoptUcred(fd, level, opt)
	value := &Ucred{Pid: int32(pid), Uid: uint32(uid), Gid: uint32(gid)}
	if code != 0 {
		return value, Errno(code)
	}
	return value, nil
}
