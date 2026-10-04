// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"io"
	"io/fs"
	"syscall/dart"
	"time"
)

// errPathEscapes names a path that leaves the root.
var errPathEscapes = errors.New("path escapes from parent")

// Root provides operations relative to an open directory.
type Root struct {
	*root
}

// OpenRoot opens a directory and gives the Root of it.
func OpenRoot(name string) (*Root, error) {
	fd, code := dart.RootOpen(name)
	if code != 0 {
		return nil, &PathError{Op: "open", Path: name, Err: nativeErrno(code)}
	}
	return newRoot(fd, name)
}

// OpenInRoot opens the file name inside the directory dir. It gives an error
// when the name leaves dir.
func OpenInRoot(dir, name string) (*File, error) {
	r, err := OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.Open(name)
}

// Name gives the name that OpenRoot read.
func (r *Root) Name() string { return r.name }

// Close releases the root directory descriptor.
func (r *Root) Close() error {
	return r.root.Close()
}

// Open opens a file of the root for reading.
func (r *Root) Open(name string) (*File, error) {
	return r.OpenFile(name, O_RDONLY, 0)
}

// OpenFile opens a file of the root with the flag and the permission.
func (r *Root) OpenFile(name string, flag int, perm FileMode) (*File, error) {
	if perm&0o777 != perm {
		return nil, &PathError{Op: "openat", Path: name, Err: errors.New("unsupported file mode")}
	}
	return rootOpenFile(r, name, flag, perm)
}

// Create makes a file of the root, or it truncates the file that stands there.
func (r *Root) Create(name string) (*File, error) {
	return r.OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)
}

// OpenRoot opens a directory of the root and gives the Root of it.
func (r *Root) OpenRoot(name string) (*Root, error) {
	return openRootInRoot(r, name)
}

// Stat reads the metadata of a name of the root.
func (r *Root) Stat(name string) (FileInfo, error) {
	return rootStat(r, name, false)
}

// Lstat reads the metadata of a name of the root, and it follows no symbolic
// link of the last element.
func (r *Root) Lstat(name string) (FileInfo, error) {
	return rootStat(r, name, true)
}

// ReadFile reads the whole content of a file of the root.
func (r *Root) ReadFile(name string) ([]byte, error) {
	f, err := r.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// WriteFile writes the data to a file of the root.
func (r *Root) WriteFile(name string, data []byte, perm FileMode) error {
	f, err := r.OpenFile(name, O_WRONLY|O_CREATE|O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

// Mkdir makes 1 directory of the root.
func (r *Root) Mkdir(name string, perm FileMode) error {
	if perm&0o777 != perm {
		return &PathError{Op: "mkdirat", Path: name, Err: errors.New("unsupported file mode")}
	}
	return rootMkdir(r, name, perm)
}

// Remove removes 1 name of the root.
func (r *Root) Remove(name string) error {
	return rootRemove(r, name)
}

// RemoveAll removes a name of the root and each name under it.
func (r *Root) RemoveAll(name string) error {
	return rootRemoveAll(r, name)
}

// Rename moves a name of the root to another name of the root.
func (r *Root) Rename(oldname, newname string) error {
	return rootRename(r, oldname, newname)
}

// Symlink makes a symbolic link of the root. The target reads as Go writes it,
// and the check of the root runs at each read of the link.
func (r *Root) Symlink(oldname, newname string) error {
	return rootSymlink(r, oldname, newname)
}

// Chmod writes the mode of a name of the root.
func (r *Root) Chmod(name string, mode FileMode) error {
	return rootChmod(r, name, mode)
}

// Chown writes the owner and the group of a name of the root.
func (r *Root) Chown(name string, uid, gid int) error {
	return rootChown(r, name, uid, gid)
}

// Lchown writes the owner and the group of a name of the root, and it takes a
// symbolic link itself.
func (r *Root) Lchown(name string, uid, gid int) error {
	return rootLchown(r, name, uid, gid)
}

// Chtimes writes the time of the last read and the time of the last change of
// a name of the root.
func (r *Root) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return rootChtimes(r, name, atime, mtime)
}

// Link creates a hard link to oldname. Linux and macOS preserve a final symlink.
func (r *Root) Link(oldname, newname string) error {
	return rootLink(r, oldname, newname)
}

// rootError writes the name of the caller and the operation of the root into
// the error of an operation.
func rootError(op, name string, err error) error {
	if err == nil {
		return nil
	}
	return &PathError{Op: op, Path: name, Err: underlyingError(err)}
}

// Readlink gives the target of a symbolic link of the root.
func (r *Root) Readlink(name string) (string, error) {
	return rootReadlink(r, name)
}

// FS gives the file system of the root. Each read of it holds the check of the
// root.
func (r *Root) FS() fs.FS { return rootFS{r} }

// MkdirAll makes a directory of the root and each parent of it.
func (r *Root) MkdirAll(name string, perm FileMode) error {
	if perm&0o777 != perm {
		return &PathError{Op: "mkdirat", Path: name, Err: errors.New("unsupported file mode")}
	}
	return rootMkdirAll(r, name, perm)
}
