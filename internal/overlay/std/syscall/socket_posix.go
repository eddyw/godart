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
		return fd, syscallErrno(code)
	}
	return fd, nil
}

type Sockaddr interface {
	sockaddrBytes() ([]byte, error)
}

type SockaddrInet4 struct {
	Port int
	Addr [4]byte
	raw  RawSockaddrInet4
}

type SockaddrInet6 struct {
	Port   int
	ZoneId uint32
	Addr   [16]byte
	raw    RawSockaddrInet6
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
	if code := dart.SyscallBind(fd, address); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func Connect(fd int, sa Sockaddr) error {
	address, err := sa.sockaddrBytes()
	if err != nil {
		return err
	}
	if code := dart.SyscallConnect(fd, address); code != 0 {
		return syscallErrno(code)
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
	if code := dart.SyscallSendto(fd, data, flags, address); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func SetsockoptString(fd, level, opt int, value string) error {
	if code := dart.SyscallSetsockoptString(fd, level, opt, value); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func GetsockoptInt(fd, level, opt int) (int, error) {
	value, code := dart.SyscallGetsockoptInt(fd, level, opt)
	if code != 0 {
		return value, syscallErrno(code)
	}
	return value, nil
}

func SetsockoptInt(fd, level, opt, value int) error {
	if code := dart.SyscallSetsockoptInt(fd, level, opt, value); code != 0 {
		return syscallErrno(code)
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
		return fd, syscallErrno(code)
	}
	return [2]int{first, second}, nil
}

func SetsockoptByte(fd, level, opt int, value byte) error {
	return SetsockoptString(fd, level, opt, string([]byte{value}))
}

func Shutdown(fd, how int) error {
	if code := dart.SyscallShutdown(fd, how); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func Listen(fd, how int) error {
	if code := dart.SyscallListen(fd, how); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func Getsockname(fd int) (Sockaddr, error) {
	data := make([]byte, SizeofSockaddrAny)
	if code := dart.SyscallGetsockname(fd, data); code != 0 {
		return nil, syscallErrno(code)
	}
	return sockaddrFromBytes(data)
}

func Getpeername(fd int) (Sockaddr, error) {
	data := make([]byte, SizeofSockaddrAny)
	if code := dart.SyscallGetpeername(fd, data); code != 0 {
		return nil, syscallErrno(code)
	}
	return sockaddrFromBytes(data)
}
