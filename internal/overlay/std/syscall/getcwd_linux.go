// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "syscall/dart"

func Getcwd(buf []byte) (int, error) {
	n, code := dart.SyscallGetcwd(buf)
	if code != 0 {
		return n, Errno(code)
	}
	return n, nil
}
