// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package net

import (
	"errors"
	"os"
	"syscall"
	"syscall/dart"
	"time"
)

type unixgram struct {
	handle        int
	local, remote *UnixAddr
	closed        bool
}

func (a *UnixAddr) opAddr() Addr {
	if a == nil {
		return nil
	}
	return a
}

func (g *unixgram) opError(op string, err error) error {
	return &OpError{Op: op, Net: "unixgram", Source: g.local.opAddr(), Addr: g.remote.opAddr(), Err: err}
}

func (g *unixgram) readMsg(data, control []byte) (n, oobn, flags int, from *UnixAddr, err error) {
	if g.closed {
		return 0, 0, 0, nil, g.opError("read", ErrClosed)
	}
	n, oobn, flags, name, cause, code, expired := dart.UnixgramRead(g.handle, data, control)
	if cause != "" {
		var err error = socketSyscallCause("recvmsg", cause, code)
		if expired {
			err = os.ErrDeadlineExceeded
		}
		return n, oobn, flags, nil, g.opError("read", err)
	}
	if name != "" {
		from = &UnixAddr{Name: name, Net: "unixgram"}
	}
	return n, oobn, flags, from, nil
}

func (g *unixgram) writeMsg(data, control []byte, destination *UnixAddr, sendTo bool) (int, int, error) {
	if g.closed {
		return 0, 0, g.opError("write", ErrClosed)
	}
	if sendTo && g.remote != nil {
		return 0, 0, g.opError("write", ErrWriteToConnected)
	}
	if sendTo && destination == nil {
		return 0, 0, g.opError("write", errMissingAddress)
	}
	name := ""
	if destination != nil {
		if destination.Net != "unixgram" {
			return 0, 0, g.opError("write", syscall.EAFNOSUPPORT)
		}
		name = destination.Name
	}
	n, oobn, cause, code, expired := dart.UnixgramWrite(g.handle, data, control, name, sendTo)
	if cause != "" {
		var err error = socketSyscallCause("sendmsg", cause, code)
		if expired {
			err = os.ErrDeadlineExceeded
		}
		return n, oobn, g.opError("write", err)
	}
	return n, oobn, nil
}

func (g *unixgram) close() error {
	if g.closed {
		return g.opError("close", ErrClosed)
	}
	g.closed = true
	if cause, code := dart.UnixgramClose(g.handle); cause != "" {
		return g.opError("close", socketSyscallCause("close", cause, code))
	}
	return nil
}

func (g *unixgram) deadline(t time.Time, mode int) error {
	if g.closed {
		return g.opError("set", ErrClosed)
	}
	clear := t.IsZero()
	var timeout int64
	if !clear {
		timeout = time.Until(t).Nanoseconds()
	}
	if cause, code := dart.UnixgramDeadline(g.handle, timeout, mode, clear); cause != "" {
		return g.opError("set", socketSyscallCause("set", cause, code))
	}
	return nil
}

func (g *unixgram) buffer(read bool, size int) error {
	if g.closed {
		return g.opError("set", ErrClosed)
	}
	if cause, code := dart.UnixgramBuffer(g.handle, read, size); cause != "" {
		return g.opError("set", socketSyscallCause("setsockopt", cause, code))
	}
	return nil
}

func (g *unixgram) shutdown(direction int) error {
	if g.closed {
		return g.opError("close", ErrClosed)
	}
	if cause, code := dart.UnixgramShutdown(g.handle, direction); cause != "" {
		return g.opError("close", socketSyscallCause("shutdown", cause, code))
	}
	return nil
}

func openUnixgram(op string, local, remote *UnixAddr) (*UnixConn, error) {
	localName, remoteName := "", ""
	if local != nil {
		localName = local.Name
	}
	if remote != nil {
		remoteName = remote.Name
	}
	handle, actualLocal, actualRemote, cause, code := dart.UnixgramOpen(localName, remoteName, local != nil, remote != nil)
	if cause != "" {
		return nil, &OpError{Op: op, Net: "unixgram", Source: local.opAddr(), Addr: remote.opAddr(), Err: socketSyscallCause(op, cause, code)}
	}
	g := &unixgram{handle: handle, local: &UnixAddr{Name: actualLocal, Net: "unixgram"}}
	if remote != nil {
		g.remote = &UnixAddr{Name: actualRemote, Net: "unixgram"}
	}
	return &UnixConn{gram: g}, nil
}

func ListenUnixgram(network string, local *UnixAddr) (*UnixConn, error) {
	if network != "unixgram" {
		return nil, &OpError{Op: "listen", Net: network, Addr: local.opAddr(), Err: UnknownNetworkError(network)}
	}
	if local == nil || local.Name == "" {
		return nil, &OpError{Op: "listen", Net: network, Addr: local.opAddr(), Err: errMissingAddress}
	}
	return openUnixgram("listen", local, nil)
}

func (u *UnixConn) ReadMsgUnix(data, control []byte) (int, int, int, *UnixAddr, error) {
	if u.gram == nil && u.c.network == "" {
		return 0, 0, 0, nil, syscall.EINVAL
	}
	if u.gram == nil {
		return 0, 0, 0, nil, &OpError{Op: "read", Net: "unix", Err: errors.ErrUnsupported}
	}
	return u.gram.readMsg(data, control)
}

func (u *UnixConn) WriteMsgUnix(data, control []byte, destination *UnixAddr) (int, int, error) {
	if u.gram == nil && u.c.network == "" {
		return 0, 0, syscall.EINVAL
	}
	if u.gram == nil {
		return 0, 0, &OpError{Op: "write", Net: "unix", Err: errors.ErrUnsupported}
	}
	if u.gram.remote != nil {
		return 0, 0, &OpError{Op: "write", Net: "unixgram", Source: u.gram.local.opAddr(), Addr: destination.opAddr(), Err: ErrWriteToConnected}
	}
	n, oobn, err := u.gram.writeMsg(data, control, destination, destination != nil)
	if err != nil {
		err.(*OpError).Addr = destination.opAddr()
	}
	return n, oobn, err
}

func (u *UnixConn) WriteToUnix(data []byte, destination *UnixAddr) (int, error) {
	if u.gram == nil && u.c.network == "" {
		return 0, syscall.EINVAL
	}
	if u.gram == nil {
		return 0, &OpError{Op: "write", Net: "unix", Err: errors.ErrUnsupported}
	}
	n, _, err := u.gram.writeMsg(data, nil, destination, true)
	if err != nil {
		err.(*OpError).Addr = destination.opAddr()
	}
	return n, err
}

func (u *UnixConn) WriteTo(data []byte, destination Addr) (int, error) {
	if u.gram == nil && u.c.network == "" {
		return 0, syscall.EINVAL
	}
	addr, ok := destination.(*UnixAddr)
	if !ok {
		return 0, &OpError{Op: "write", Net: "unixgram", Addr: destination, Err: syscall.EINVAL}
	}
	return u.WriteToUnix(data, addr)
}
