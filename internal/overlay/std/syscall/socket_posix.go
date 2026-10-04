// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

var SocketDisableIPv6 bool

func Socket(domain, typ, proto int) (fd int, err error) {
	if domain == AF_INET6 && SocketDisableIPv6 {
		return -1, EAFNOSUPPORT
	}
	fd, code := dart.SyscallSocket(domain, typ, proto)
	if code != 0 {
		return fd, Errno(code)
	}
	return fd, nil
}

type Sockaddr interface {
	sockaddrUnix() string
}

type SockaddrUnix struct {
	Name string
}

func (a *SockaddrUnix) sockaddrUnix() string { return a.Name }

func Sendto(fd int, data []byte, flags int, destination Sockaddr) error {
	name := ""
	if destination != nil {
		name = destination.sockaddrUnix()
	}
	if code := dart.SyscallUnixSendto(fd, data, flags, name, destination != nil); code != 0 {
		return Errno(code)
	}
	return nil
}

func SetsockoptString(fd, level, opt int, value string) error {
	if code := dart.SyscallSetsockoptString(fd, level, opt, value); code != 0 {
		return Errno(code)
	}
	return nil
}
