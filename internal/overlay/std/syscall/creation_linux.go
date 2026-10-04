// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

func Creat(path string, mode uint32) (int, error) {
	return Open(path, O_CREAT|O_WRONLY|O_TRUNC, mode)
}

func Pipe(p []int) error { return Pipe2(p, 0) }
