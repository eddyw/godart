// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

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
	sockaddrBytes() ([]byte, error)
}

type SockaddrUnix struct {
	Name string
	raw  RawSockaddrUnix
}

func Bind(fd int, sa Sockaddr) error {
	address, err := sa.sockaddrBytes()
	if err != nil {
		return err
	}
	if code := dart.SyscallUnixBind(fd, address); code != 0 {
		return Errno(code)
	}
	return nil
}

func Connect(fd int, sa Sockaddr) error {
	address, err := sa.sockaddrBytes()
	if err != nil {
		return err
	}
	if code := dart.SyscallUnixConnect(fd, address); code != 0 {
		return Errno(code)
	}
	return nil
}

func Sendto(fd int, data []byte, flags int, destination Sockaddr) error {
	var address []byte
	if destination != nil {
		var err error
		address, err = destination.sockaddrBytes()
		if err != nil {
			return err
		}
	}
	if code := dart.SyscallUnixSendto(fd, data, flags, address); code != 0 {
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

func GetsockoptInt(fd, level, opt int) (int, error) {
	value, code := dart.SyscallGetsockoptInt(fd, level, opt)
	if code != 0 {
		return value, Errno(code)
	}
	return value, nil
}

func SetsockoptInt(fd, level, opt, value int) error {
	if code := dart.SyscallSetsockoptInt(fd, level, opt, value); code != 0 {
		return Errno(code)
	}
	return nil
}

func GetsockoptInet4Addr(fd, level, opt int) (value [4]byte, err error) {
	n, err := GetsockoptInt(fd, level, opt)
	binary.NativeEndian.PutUint32(value[:], uint32(n))
	return value, err
}

func SetsockoptInet4Addr(fd, level, opt int, value [4]byte) error {
	return SetsockoptInt(fd, level, opt, int(int32(binary.NativeEndian.Uint32(value[:]))))
}

func Socketpair(domain, typ, proto int) (fd [2]int, err error) {
	first, second, code := dart.SyscallSocketpair(domain, typ, proto)
	if code != 0 {
		return fd, Errno(code)
	}
	return [2]int{first, second}, nil
}

func SetsockoptByte(fd, level, opt int, value byte) error {
	return SetsockoptString(fd, level, opt, string([]byte{value}))
}

func Shutdown(fd, how int) error {
	if code := dart.SyscallShutdown(fd, how); code != 0 {
		return Errno(code)
	}
	return nil
}

func Listen(fd, how int) error {
	if code := dart.SyscallListen(fd, how); code != 0 {
		return Errno(code)
	}
	return nil
}
