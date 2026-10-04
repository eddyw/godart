// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "syscall/dart"

func Exchangedata(first, second string, options int) error {
	if code := dart.SyscallExchangedata(first, second, options); code != 0 {
		return Errno(code)
	}
	return nil
}
