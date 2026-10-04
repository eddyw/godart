// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func selectSetData(set *FdSet) []byte {
	if set == nil {
		return nil
	}
	data := make([]byte, 128)
	for i, word := range set.Bits {
		binary.NativeEndian.PutUint64(data[i*8:], uint64(word))
	}
	return data
}

func selectLoadSet(set *FdSet, data []byte) {
	if set == nil {
		return
	}
	for i := range set.Bits {
		set.Bits[i] = int64(binary.NativeEndian.Uint64(data[i*8:]))
	}
}

func Select(nfd int, r, w, e *FdSet, timeout *Timeval) (int, error) {
	if nfd > 1024 && (r != nil || w != nil || e != nil) {
		return -1, EINVAL
	}
	read, write, except := selectSetData(r), selectSetData(w), selectSetData(e)
	if w == r {
		write = read
	}
	if e == r {
		except = read
	} else if e == w {
		except = write
	}
	var timeval []byte
	if timeout != nil {
		timeval = make([]byte, 16)
		binary.NativeEndian.PutUint64(timeval, uint64(timeout.Sec))
		binary.NativeEndian.PutUint64(timeval[8:], uint64(timeout.Usec))
	}
	count, code := dart.SyscallSelect(nfd, read, write, except, timeval)
	selectLoadSet(r, read)
	selectLoadSet(w, write)
	selectLoadSet(e, except)
	if timeout != nil {
		timeout.Sec = int64(binary.NativeEndian.Uint64(timeval))
		timeout.Usec = int64(binary.NativeEndian.Uint64(timeval[8:]))
	}
	if code != 0 {
		return count, Errno(code)
	}
	return count, nil
}
