// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (darwin || linux) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Exit(code int) { dart.Exit(int(int32(code))) }
