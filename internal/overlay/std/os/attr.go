// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"syscall"
	"syscall/dart"
	"time"
)

// The 3 bits of a POSIX mode that stand above the 9 permission bits.
const (
	sISUID = 0o4000
	sISGID = 0o2000
	sISVTX = 0o1000
)

// syscallMode gives the POSIX mode of a FileMode. Go carries the same 12 bits
// to the system.
func syscallMode(mode FileMode) int {
	out := int(mode.Perm())
	if mode&ModeSetuid != 0 {
		out |= sISUID
	}
	if mode&ModeSetgid != 0 {
		out |= sISGID
	}
	if mode&ModeSticky != 0 {
		out |= sISVTX
	}
	return out
}

// stdError reports the 3 files that the process holds at its start. The surface
// holds no descriptor of them, thus an operation that needs 1 fails.
//
// The test of the receiver stands first, exactly as each method of Go starts
// with checkValid: a nil file gives ErrInvalid, and a closed file gives
// ErrClosed.
func (f *File) stdError(op string) error {
	if err := f.checkValid(op); err != nil {
		return err
	}
	if f.std == stdNone {
		return nil
	}
	return &PathError{Op: op, Path: f.name, Err: errors.New("bad file descriptor")}
}

// underlyingError unwraps only the error types that the os predicates accept.
func underlyingError(err error) error {
	switch e := err.(type) {
	case *PathError:
		return e.Err
	case *LinkError:
		return e.Err
	case *SyscallError:
		return e.Err
	}
	return err
}

// Chmod writes the mode of the named file. It reads the 9 permission bits, and
// the 3 bits of setuid, of setgid, and of the sticky flag.
func Chmod(name string, mode FileMode) error {
	if contains(name, "\x00") {
		return &PathError{Op: "chmod", Path: name, Err: syscall.EINVAL}
	}
	if system := dart.OS(); system == "linux" || system == "darwin" {
		if errno := dart.NativeChmod(name, syscallMode(mode)); errno != 0 {
			return &PathError{Op: "chmod", Path: name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	return pathError("chmod", name, dart.Chmod(name, syscallMode(mode)))
}

// Chmod writes the mode of the file.
func (f *File) Chmod(mode FileMode) error {
	if system := dart.OS(); system == "linux" || system == "darwin" {
		if err := f.checkValid("chmod"); err != nil {
			return err
		}
		if errno := dart.FileChmod(int(f.Fd()), syscallMode(mode)); errno != 0 {
			return &PathError{Op: "chmod", Path: f.name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	if err := f.stdError("chmod"); err != nil {
		return err
	}
	return pathError("chmod", f.name, dart.Chmod(f.name, syscallMode(mode)))
}

// Chown writes the owner and the group of the named file. A number of -1 leaves
// that 1 of the 2 alone. It reads the target of a symbolic link.
func Chown(name string, uid, gid int) error {
	if contains(name, "\x00") {
		return &PathError{Op: "chown", Path: name, Err: syscall.EINVAL}
	}
	if system := dart.OS(); system == "linux" || system == "darwin" {
		if errno := dart.NativeChown(name, uid, gid, true); errno != 0 {
			return &PathError{Op: "chown", Path: name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	return pathError("chown", name, dart.Chown(name, uid, gid, true))
}

// Lchown writes the owner and the group of the named file, and it takes a
// symbolic link itself.
func Lchown(name string, uid, gid int) error {
	if contains(name, "\x00") {
		return &PathError{Op: "lchown", Path: name, Err: syscall.EINVAL}
	}
	if system := dart.OS(); system == "linux" || system == "darwin" {
		if errno := dart.NativeChown(name, uid, gid, false); errno != 0 {
			return &PathError{Op: "lchown", Path: name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	return pathError("lchown", name, dart.Chown(name, uid, gid, false))
}

// Chown writes the owner and the group of the file.
func (f *File) Chown(uid, gid int) error {
	if system := dart.OS(); system == "linux" || system == "darwin" {
		if err := f.checkValid("chown"); err != nil {
			return err
		}
		if errno := dart.FileChown(int(f.Fd()), uid, gid); errno != 0 {
			return &PathError{Op: "chown", Path: f.name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	if err := f.stdError("chown"); err != nil {
		return err
	}
	return pathError("chown", f.name, dart.Chown(f.name, uid, gid, true))
}

// Chtimes writes the time of the last read and the time of the last change of
// the named file. A zero time leaves that 1 of the 2 alone.
func Chtimes(name string, atime time.Time, mtime time.Time) error {
	if contains(name, "\x00") {
		return &PathError{Op: "chtimes", Path: name, Err: syscall.EINVAL}
	}
	if system := dart.OS(); system == "linux" || system == "darwin" {
		errno := dart.NativeChtimes(name, atime.UnixNano(), !atime.IsZero(), mtime.UnixNano(), !mtime.IsZero())
		if errno != 0 {
			return &PathError{Op: "chtimes", Path: name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	cause := dart.Chtimes(name, atime.UnixNano(), !atime.IsZero(),
		mtime.UnixNano(), !mtime.IsZero())
	return pathError("chtimes", name, cause)
}

// Truncate writes the size of the named file. It does not move the read
// position of an open file.
func Truncate(name string, size int64) error {
	// The write mode of the Dart side makes a file that does not exist yet, and
	// Go reports such a path instead.
	if _, err := Stat(name); err != nil {
		return &PathError{Op: "truncate", Path: name, Err: underlyingError(err)}
	}
	handle, cause := dart.FileOpen(name, dart.OpenReadWrite)
	if cause != "" {
		return pathError("truncate", name, cause)
	}
	cause = dart.FileTruncate(handle, size)
	dart.FileClose(handle)
	return pathError("truncate", name, cause)
}

// Truncate writes the size of the file. It does not move the read position.
func (f *File) Truncate(size int64) error {
	if err := f.stdError("truncate"); err != nil {
		return err
	}
	return pathError("truncate", f.name, dart.FileTruncate(f.handle, size))
}

// Chdir writes the working directory of the program to the file, and the file
// must be a directory.
func (f *File) Chdir() error {
	if err := f.checkValid("chdir"); err != nil {
		return err
	}
	if system := dart.OS(); system == "linux" || system == "darwin" {
		if errno := dart.DirectoryChdir(int(f.Fd())); errno != 0 {
			return &PathError{Op: "chdir", Path: f.name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	return pathError("chdir", f.name, dart.Chdir(f.name))
}

// Link makes newname as a hard link of the file oldname.
func Link(oldname, newname string) error {
	if contains(oldname, "\x00") || contains(newname, "\x00") {
		return &LinkError{Op: "link", Old: oldname, New: newname, Err: syscall.EINVAL}
	}
	if dart.HasNativeStat() {
		if errno := dart.NativeLink(oldname, newname); errno != 0 {
			return &LinkError{Op: "link", Old: oldname, New: newname, Err: syscall.Errno(errno)}
		}
		return nil
	}
	cause := dart.Link(oldname, newname)
	if cause == "" {
		return nil
	}
	return &LinkError{Op: "link", Old: oldname, New: newname, Err: causeError(cause)}
}

// SetDeadline gives ErrNoDeadline.
//
// A read of the Dart side is 1 synchronous call, and no deadline stops it. Go
// gives the same error for an ordinary file of a platform whose poller holds no
// such file. The test of the receiver stands first, exactly as in Go.
func (f *File) SetDeadline(t time.Time) error {
	if err := f.checkValid("SetDeadline"); err != nil {
		return err
	}
	return ErrNoDeadline
}

// SetReadDeadline gives ErrNoDeadline, as SetDeadline does.
func (f *File) SetReadDeadline(t time.Time) error {
	if err := f.checkValid("SetReadDeadline"); err != nil {
		return err
	}
	return ErrNoDeadline
}

// SetWriteDeadline gives ErrNoDeadline, as SetDeadline does.
func (f *File) SetWriteDeadline(t time.Time) error {
	if err := f.checkValid("SetWriteDeadline"); err != nil {
		return err
	}
	return ErrNoDeadline
}
