// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"runtime"
	"syscall"
	"syscall/dart"
)

type rawTCPListener struct {
	listener *TCPListener
}

func (l *TCPListener) SyscallConn() (syscall.RawConn, error) {
	if l == nil || l.l.network == "" {
		return nil, syscall.EINVAL
	}
	if system := dart.OS(); system != "linux" && system != "darwin" {
		return nil, syscall.ENOSYS
	}
	return &rawTCPListener{l}, nil
}

func (c *rawTCPListener) Control(callback func(uintptr)) error {
	fd, kind := dart.ListenRawAcquire(c.listener.l.handle)
	if kind != 0 {
		var cause error = ErrClosed
		if kind == 2 {
			cause = syscall.ENOSYS
		}
		return &OpError{Op: "raw-control", Net: c.listener.l.network, Addr: c.listener.l.addr, Err: cause}
	}
	defer dart.ListenRawRelease(fd)
	defer runtime.KeepAlive(c.listener)
	callback(uintptr(fd))
	return nil
}

func (c *rawTCPListener) Read(func(uintptr) bool) error {
	return syscall.EINVAL
}

func (c *rawTCPListener) Write(func(uintptr) bool) error {
	return syscall.EINVAL
}
