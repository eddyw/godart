// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package os

import (
	"io"
	"syscall"
)

func (f *File) readDirNames(n int) ([]string, error) {
	names := make([]string, 0)
	remaining := n
	if remaining <= 0 {
		remaining = -1
	}
	for remaining != 0 {
		if f.dirBufPos >= len(f.dirBuf) {
			f.dirBuf = make([]byte, 8192)
			f.dirBufPos = 0
			count, err := syscall.ReadDirent(int(f.Fd()), f.dirBuf)
			for err == syscall.EINTR {
				count, err = syscall.ReadDirent(int(f.Fd()), f.dirBuf)
			}
			if err != nil {
				f.dirBuf = nil
				return names, &PathError{Op: "readdirent", Path: f.name, Err: err}
			}
			f.dirBuf = f.dirBuf[:count]
			if count == 0 {
				break
			}
		}
		consumed, count, next := syscall.ParseDirent(f.dirBuf[f.dirBufPos:], remaining, names)
		f.dirBufPos += consumed
		names = next
		if remaining > 0 {
			remaining -= count
		}
	}
	if n > 0 && len(names) == 0 {
		return names, io.EOF
	}
	return names, nil
}
