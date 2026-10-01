// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getdirentries(fd int, buf []byte, basep *uintptr) (int, error) {
	if dart.OS() == "js" {
		return 0, errorspkg.ErrUnsupported
	}
	// Go's Darwin cursor uses the descriptor offset and ignores basep.
	return ReadDirent(fd, buf)
}
