// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"runtime"
	"syscall"
	"syscall/dart"
)

var errRawFileClosing = errors.New("use of closed file")

type rawConn struct {
	file *File
}

// SyscallConn returns a raw file. File implements syscall.Conn.
// The descriptor remains valid for the complete callback.
func (f *File) SyscallConn() (syscall.RawConn, error) {
	if f == nil {
		return nil, ErrInvalid
	}
	if system := dart.OS(); system != "linux" && system != "darwin" {
		return nil, syscall.ENOSYS
	}
	return &rawConn{f}, nil
}

func (c *rawConn) acquire() error {
	if c.file.closed {
		return errRawFileClosing
	}
	c.file.rawRefs++
	return nil
}

func (c *rawConn) release() {
	c.file.rawRefs--
	if c.file.rawRefs == 0 {
		if c.file.rawCloseDeferred {
			c.file.closeNative()
		} else if c.file.rawDone != nil {
			close(c.file.rawDone)
		}
	}
	runtime.KeepAlive(c.file)
}

func (c *rawConn) Control(callback func(uintptr)) error {
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	callback(c.file.Fd())
	return nil
}

func rawFileError(code, kind int) error {
	switch kind {
	case 1:
		return errRawFileClosing
	case 2:
		return ErrDeadlineExceeded
	case 3:
		return errors.New("waiting for unsupported file type")
	}
	if code != 0 {
		return syscall.Errno(code)
	}
	return nil
}

func (c *rawConn) Read(callback func(uintptr) bool) error {
	c.file.rawReadMu.Lock()
	defer c.file.rawReadMu.Unlock()
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	if c.file.std == stdNone && !c.file.dir {
		if err := rawFileError(0, dart.FileRawPrepare(c.file.handle, 1)); err != nil {
			return err
		}
	}
	fd := c.file.Fd()
	for !callback(fd) {
		if c.file.std != stdNone || c.file.dir {
			return rawFileError(0, 3)
		}
		code, kind := dart.FileRawWait(c.file.handle, 1)
		if err := rawFileError(code, kind); err != nil {
			return err
		}
	}
	return nil
}

func (c *rawConn) Write(callback func(uintptr) bool) error {
	c.file.rawWriteMu.Lock()
	defer c.file.rawWriteMu.Unlock()
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	if c.file.std == stdNone && !c.file.dir {
		if err := rawFileError(0, dart.FileRawPrepare(c.file.handle, 2)); err != nil {
			return err
		}
	}
	fd := c.file.Fd()
	for !callback(fd) {
		if c.file.std != stdNone || c.file.dir {
			return rawFileError(0, 3)
		}
		code, kind := dart.FileRawWait(c.file.handle, 2)
		if err := rawFileError(code, kind); err != nil {
			return err
		}
	}
	return nil
}
