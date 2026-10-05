// Copyright 2009,2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "syscall/dart"

// Sendfile copies up to count bytes from file descriptor infd to file descriptor outfd.
//
// It wraps the sendfile system call. The behavior varies by operating system,
// particularly regarding the offset argument and how partial writes are reported.
//
// On Linux, if offset is nil, Sendfile uses and updates the current file
// position of infd. If offset is non-nil, the current file position is
// unchanged, and the offset pointer is updated to reflect the bytes written.
// A non-nil error typically implies that no bytes were written.
//
// On BSD-derived systems (including macOS), if offset is nil, Sendfile panics.
// The offset argument is not updated by the system call; the caller must manually
// update the offset using the number of bytes written. These systems may
// return a non-zero byte count together with an error (for example, EAGAIN).
//
// For precise semantics, see the system documentation (e.g., man 2 sendfile).
func Sendfile(outfd, infd int, offset *int64, count int) (written int, err error) {
	written, code := dart.SyscallDarwinSendfile(outfd, infd, *offset, count)
	if code != 0 {
		return written, syscallErrno(code)
	}
	return written, nil
}
