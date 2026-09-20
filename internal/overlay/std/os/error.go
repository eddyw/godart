// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import "syscall"

func underlyingErrorIs(err, target error) bool {
	err = underlyingError(err)
	if err == target {
		return true
	}
	// Custom Is methods do not extend these legacy predicates.
	switch e := err.(type) {
	case syscall.Errno:
		return e.Is(target)
	case *sysError:
		return e.Is(target)
	case existsError:
		return e.Is(target)
	}
	return false
}
