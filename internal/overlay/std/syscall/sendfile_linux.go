// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func Sendfile(outfd, infd int, offset *int64, count int) (written int, err error) {
	var data []byte
	if offset != nil {
		data = make([]byte, 8)
		binary.NativeEndian.PutUint64(data, uint64(*offset))
	}
	written, code := dart.SyscallSendfile(outfd, infd, data, count)
	if offset != nil {
		*offset = int64(binary.NativeEndian.Uint64(data))
	}
	if code != 0 {
		return written, Errno(code)
	}
	return written, nil
}
