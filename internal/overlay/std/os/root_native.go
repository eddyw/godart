// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"syscall/dart"
	"time"
)

type sysfdType = int

type root struct {
	name    string
	mu      sync.Mutex
	fd      int
	refs    int
	closed  bool
	cleanup runtime.Cleanup
}

func closeRootFD(fd int) { dart.SyscallClose(fd) }

func newRoot(fd int, name string) (*Root, error) {
	data, code := dart.NativeFstat(fd)
	if code != 0 {
		closeRootFD(fd)
		return nil, &PathError{Op: "open", Path: name, Err: syscall.Errno(code)}
	}
	if nativeFileMode(uint32(data[1]))&ModeDir == 0 {
		closeRootFD(fd)
		return nil, &PathError{Op: "open", Path: name, Err: errors.New("not a directory")}
	}
	r := &root{name: name, fd: fd}
	r.cleanup = runtime.AddCleanup(r, closeRootFD, fd)
	return &Root{r}, nil
}

func (r *root) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed && r.refs == 0 {
		closeRootFD(r.fd)
	}
	r.closed = true
	r.cleanup.Stop()
	return nil
}

func (r *root) incref() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	r.refs++
	return nil
}

func (r *root) decref() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refs--
	if r.refs == 0 && r.closed {
		closeRootFD(r.fd)
	}
}

func rootErrno(code int) error {
	if code == 0 {
		return nil
	}
	return syscall.Errno(code)
}

func openRootInRoot(r *Root, name string) (*Root, error) {
	fd, err := doInRoot(r, name, 0, nil, func(parent int, name string, endsInSlash bool) (int, error) {
		fd, code := dart.RootOpenAt(parent, name)
		err := rootErrno(code)
		if err == syscall.ELOOP {
			err = checkSymlink(parent, name, err)
		}
		return fd, err
	})
	if err != nil {
		return nil, rootError("openat", name, err)
	}
	return newRoot(fd, joinPath(r.name, name))
}

func rootOpenFile(r *Root, name string, flag int, perm FileMode) (*File, error) {
	f, err := doInRoot(r, name, 0, nil, func(parent int, path string, endsInSlash bool) (*File, error) {
		handle, dirFD, code := dart.RootOpenFile(parent, path, flag, int(syscallMode(perm)))
		if code != 0 {
			err := rootErrno(code)
			isCreateExcl := flag&(O_CREATE|O_EXCL) == (O_CREATE | O_EXCL)
			if !isCreateExcl && (err == syscall.ELOOP || err == syscall.ENOTDIR) {
				err = checkSymlink(parent, path, err)
			}
			return nil, err
		}
		return fileWithCleanup(&file{handle: handle, name: joinPath(r.name, name), dir: dirFD >= 0, inRoot: true, dirFD: dirFD, appendMode: flag&O_APPEND != 0}), nil
	})
	if err != nil {
		return nil, rootError("openat", name, err)
	}
	return f, nil
}

func rootOpenDir(parent int, name string) (int, error) {
	fd, code := dart.RootOpenDir(parent, name)
	err := rootErrno(code)
	if err == syscall.ELOOP || err == syscall.ENOTDIR {
		err = checkSymlink(parent, name, err)
	} else if err == syscall.ENOTSUP || err == syscall.EOPNOTSUPP {
		err = syscall.ENOTDIR
	}
	return fd, err
}

func rootStat(r *Root, name string, lstat bool) (FileInfo, error) {
	fi, err := doInRoot(r, name, 0, nil, func(parent int, path string, endsInSlash bool) (FileInfo, error) {
		fi, err := lstatatWithName(parent, name, path)
		if err != nil {
			return nil, err
		}
		if !lstat && fi.Mode()&ModeSymlink != 0 {
			return nil, checkSymlink(parent, path, syscall.ELOOP)
		}
		return fi, nil
	})
	if err != nil {
		return nil, rootError("statat", name, err)
	}
	return fi, nil
}

func rootSymlink(r *Root, oldname, newname string) error {
	_, err := doInRoot(r, newname, 0, nil, func(parent int, name string, endsInSlash bool) (struct{}, error) {
		return struct{}{}, rootErrno(dart.RootSymlink(oldname, parent, name))
	})
	if err != nil {
		return &LinkError{Op: "symlinkat", Old: oldname, New: newname, Err: err}
	}
	return nil
}

func afterResolvingSymlink(parent int, name string, f func() error) error {
	if err := checkSymlink(parent, name, nil); err != nil {
		return err
	}
	return f()
}

func chmodat(parent int, name string, mode FileMode) error {
	return afterResolvingSymlink(parent, name, func() error {
		return rootErrno(dart.RootChmod(parent, name, int(syscallMode(mode))))
	})
}

func chownat(parent int, name string, uid, gid int) error {
	return afterResolvingSymlink(parent, name, func() error {
		return rootErrno(dart.RootChown(parent, name, uid, gid))
	})
}

func lchownat(parent int, name string, uid, gid int) error {
	return rootErrno(dart.RootChown(parent, name, uid, gid))
}

func chtimesat(parent int, name string, atime, mtime time.Time) error {
	return afterResolvingSymlink(parent, name, func() error {
		return rootErrno(dart.RootChtimes(parent, name, atime.UnixNano(), !atime.IsZero(), mtime.UnixNano(), !mtime.IsZero()))
	})
}

func mkdirat(parent int, name string, mode FileMode) error {
	return rootErrno(dart.RootMkdir(parent, name, int(syscallMode(mode))))
}

func removeat(parent int, name string) error {
	return rootErrno(dart.RootRemove(parent, name))
}

func removeAllFrom(parent int, name string) error {
	return rootErrno(dart.RootRemoveAll(parent, name))
}

func renameat(oldParent int, oldname string, newParent int, newname string) error {
	return rootErrno(dart.RootRename(oldParent, oldname, newParent, newname))
}

func linkat(oldParent int, oldname string, newParent int, newname string) error {
	return rootErrno(dart.RootLink(oldParent, oldname, newParent, newname))
}

func lstatat(parent int, name string) (FileInfo, error) {
	return lstatatWithName(parent, name, name)
}

func checkSymlink(parent int, name string, origError error) error {
	link, err := readlinkat(parent, name)
	if err != nil {
		return origError
	}
	return errSymlink(link)
}

func readlinkat(parent int, name string) (string, error) {
	target, code := dart.RootReadlink(parent, name)
	return target, rootErrno(code)
}
