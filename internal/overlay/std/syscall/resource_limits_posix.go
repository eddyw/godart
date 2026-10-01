// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Getrlimit(resource int, rlim *Rlimit) error {
	cur, max, code := dart.SyscallGetrlimit(resource, rlim != nil)
	if code != 0 {
		return Errno(code)
	}
	if rlim != nil {
		rlim.Cur, rlim.Max = cur, max
	}
	return nil
}

func Setrlimit(resource int, rlim *Rlimit) error {
	var cur, max uint64
	if rlim != nil {
		cur, max = rlim.Cur, rlim.Max
	}
	if code := dart.SyscallSetrlimit(resource, rlim != nil, cur, max); code != 0 {
		return Errno(code)
	}
	return nil
}
