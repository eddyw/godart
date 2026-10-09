// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"syscall/dart"
)

type fileAddr string

func (fileAddr) Network() string  { return "file+net" }
func (f fileAddr) String() string { return string(f) }

// FileListener duplicates a TCP listener descriptor on Linux and macOS.
// The caller must close both owners. Close on either owner does not close the other.
func FileListener(f *os.File) (Listener, error) {
	handle, address, port, code, operation := dart.FileListener(int(f.Fd()))
	runtime.KeepAlive(f)
	if code != 0 || operation != "" {
		var err error = syscall.Errno(code)
		if operation == "unsupported" {
			err = errors.ErrUnsupported
		} else if operation != "" {
			err = &os.SyscallError{Syscall: operation, Err: err}
		}
		return nil, &OpError{Op: "file", Net: "file+net", Addr: fileAddr(f.Name()), Err: err}
	}
	return newTCPListener(handle, "tcp", tcpSocketAddrOf(address, port)), nil
}
