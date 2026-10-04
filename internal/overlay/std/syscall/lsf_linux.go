// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

// Deprecated: Use golang.org/x/net/bpf instead.
func LsfStmt(code, k int) *SockFilter {
	return &SockFilter{Code: uint16(code), K: uint32(k)}
}

// Deprecated: Use golang.org/x/net/bpf instead.
func LsfJump(code, k, jt, jf int) *SockFilter {
	return &SockFilter{Code: uint16(code), Jt: uint8(jt), Jf: uint8(jf), K: uint32(k)}
}

// Deprecated: Use golang.org/x/net/bpf instead.
func AttachLsf(fd int, filters []SockFilter) error {
	_ = filters[0]
	data := make([]byte, len(filters)*8)
	for i, filter := range filters {
		entry := data[i*8:]
		binary.NativeEndian.PutUint16(entry, filter.Code)
		entry[2] = filter.Jt
		entry[3] = filter.Jf
		binary.NativeEndian.PutUint32(entry[4:], filter.K)
	}
	if code := dart.SyscallAttachLsf(fd, data); code != 0 {
		return Errno(code)
	}
	return nil
}

// Deprecated: Use golang.org/x/net/bpf instead.
func DetachLsf(fd int) error {
	return SetsockoptString(fd, SOL_SOCKET, SO_DETACH_FILTER, string(make([]byte, 8)))
}
