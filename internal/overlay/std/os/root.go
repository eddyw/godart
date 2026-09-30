// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"syscall"
	"syscall/dart"
	"time"
)

// errPathEscapes names a path that leaves the root.
var errPathEscapes = errors.New("path escapes from parent")

// Root provides path operations below a directory. It retains an absolute
// path, which does not track directory renames. Validation and filesystem
// access are separate operations. Concurrent path changes can cause access
// outside the root.
type Root struct {
	*root
}

type root struct {
	name string
	// real anchors operations to the absolute path after symlink resolution.
	real   string
	closed bool
}

// OpenRoot opens a directory and gives the Root of it.
func OpenRoot(name string) (*Root, error) {
	info, err := Stat(name)
	if err != nil {
		return nil, &PathError{Op: "open", Path: name, Err: underlyingError(err)}
	}
	if !info.IsDir() {
		return nil, &PathError{Op: "open", Path: name, Err: errors.New("not a directory")}
	}
	real, cause := dart.ResolveSymlinks(name)
	if cause != "" {
		return nil, &PathError{Op: "open", Path: name, Err: errors.New(cause)}
	}
	return &Root{&root{name: name, real: real}}, nil
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

// Close releases the root. The surface holds no directory handle, thus the
// call only marks the root.
func (r *Root) Close() error {
	r.closed = true
	return nil
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
	full, err := r.resolve("openat", name)
	if err != nil {
		return nil, err
	}
	f, cause := OpenFile(full, flag, perm)
	return r.openIn(f, cause, name)
}

// Create makes a file of the root, or it truncates the file that stands there.
func (r *Root) Create(name string) (*File, error) {
	return r.OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)
}

// OpenRoot opens a directory of the root and gives the Root of it.
func (r *Root) OpenRoot(name string) (*Root, error) {
	full, err := r.resolve("openroot", name)
	if err != nil {
		return nil, err
	}
	out, cause := OpenRoot(full)
	if cause != nil {
		return nil, rename(cause, name)
	}
	out.name = joinPath(r.name, name)
	return out, nil
}

// Stat reads the metadata of a name of the root.
func (r *Root) Stat(name string) (FileInfo, error) {
	full, err := r.resolve("statat", name)
	if err != nil {
		return nil, err
	}
	info, cause := Stat(full)
	if cause != nil {
		return nil, rootError("statat", name, cause)
	}
	return info, nil
}

// Lstat reads the metadata of a name of the root, and it follows no symbolic
// link of the last element.
func (r *Root) Lstat(name string) (FileInfo, error) {
	full, err := r.resolveNoFollow("statat", name)
	if err != nil {
		return nil, err
	}
	info, cause := Lstat(full)
	if cause != nil {
		return nil, rootError("statat", name, cause)
	}
	return info, nil
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
	full, err := r.resolve("mkdir", name)
	if err != nil {
		return err
	}
	return rename(Mkdir(full, perm), name)
}

// MkdirAll makes a directory of the root and each parent of it.
func (r *Root) MkdirAll(name string, perm FileMode) error {
	if perm&0o777 != perm {
		return &PathError{Op: "mkdirat", Path: name, Err: errors.New("unsupported file mode")}
	}
	full, err := r.resolve("mkdirat", name)
	if err != nil {
		if underlyingError(err) != syscall.ENOTDIR {
			return err
		}
		full = joinPath(r.real, name)
	}
	if err := MkdirAll(full, perm); err != nil {
		if pe, ok := err.(*PathError); ok {
			pe.Path = strings.TrimPrefix(pe.Path, r.real+string(PathSeparator))
		}
		return err
	}
	return nil
}

// Remove removes 1 name of the root.
func (r *Root) Remove(name string) error {
	full, err := r.resolveNoFollow("remove", name)
	if err != nil {
		return err
	}
	return rename(Remove(full), name)
}

// RemoveAll removes a name of the root and each name under it.
func (r *Root) RemoveAll(name string) error {
	for len(name) > 0 && IsPathSeparator(name[len(name)-1]) {
		name = name[:len(name)-1]
	}
	if endsWithDot(name) {
		return &PathError{Op: "RemoveAll", Path: name, Err: syscall.EINVAL}
	}
	full, err := r.resolveNoFollow("removeall", name)
	if err != nil {
		return err
	}
	return rename(RemoveAll(full), name)
}

// Rename moves a name of the root to another name of the root.
func (r *Root) Rename(oldname, newname string) error {
	from, err := r.resolveNoFollow("renameat", oldname)
	if err != nil {
		return &LinkError{Op: "renameat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	to, err := r.resolveNoFollow("renameat", newname)
	if err != nil {
		return &LinkError{Op: "renameat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	if err := Rename(from, to); err != nil {
		return &LinkError{Op: "renameat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	return nil
}

// Symlink makes a symbolic link of the root. The target reads as Go writes it,
// and the check of the root runs at each read of the link.
func (r *Root) Symlink(oldname, newname string) error {
	full, err := r.resolveNoFollow("symlinkat", newname)
	if err != nil {
		return &LinkError{Op: "symlinkat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	if err := Symlink(oldname, full); err != nil {
		return &LinkError{Op: "symlinkat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	return nil
}

// Chmod writes the mode of a name of the root.
func (r *Root) Chmod(name string, mode FileMode) error {
	full, err := r.resolve("chmodat", name)
	if err != nil {
		return err
	}
	return rootError("chmodat", name, Chmod(full, mode))
}

// Chown writes the owner and the group of a name of the root.
func (r *Root) Chown(name string, uid, gid int) error {
	full, err := r.resolve("chownat", name)
	if err != nil {
		return err
	}
	return rootError("chownat", name, Chown(full, uid, gid))
}

// Lchown writes the owner and the group of a name of the root, and it takes a
// symbolic link itself.
func (r *Root) Lchown(name string, uid, gid int) error {
	full, err := r.resolveNoFollow("lchownat", name)
	if err != nil {
		return err
	}
	return rootError("lchownat", name, Lchown(full, uid, gid))
}

// Chtimes writes the time of the last read and the time of the last change of
// a name of the root.
func (r *Root) Chtimes(name string, atime time.Time, mtime time.Time) error {
	full, err := r.resolve("chtimesat", name)
	if err != nil {
		return err
	}
	return rootError("chtimesat", name, Chtimes(full, atime, mtime))
}

// Link creates a hard link to oldname. Linux and macOS preserve a final symlink.
func (r *Root) Link(oldname, newname string) error {
	from, err := r.resolveNoFollow("linkat", oldname)
	if err != nil {
		return &LinkError{Op: "linkat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	to, err := r.resolveNoFollow("linkat", newname)
	if err != nil {
		return &LinkError{Op: "linkat", Old: oldname, New: newname, Err: underlyingError(err)}
	}
	if dart.HasNativeStat() {
		if code := dart.NativeLinkNoFollow(from, to); code != 0 {
			return &LinkError{Op: "linkat", Old: oldname, New: newname, Err: syscall.Errno(code)}
		}
		return nil
	}
	if info, cause := Lstat(from); cause == nil && info.Mode()&ModeSymlink != 0 {
		return &LinkError{Op: "linkat", Old: oldname, New: newname, Err: errors.New("cannot create a hard link to a symlink")}
	}
	if cause := Link(from, to); cause != nil {
		return &LinkError{Op: "linkat", Old: oldname, New: newname, Err: underlyingError(cause)}
	}
	return nil
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
	full, err := r.resolveNoFollow("readlinkat", name)
	if err != nil {
		return "", err
	}
	target, cause := Readlink(full)
	if cause != nil {
		return "", rootError("readlinkat", name, cause)
	}
	return target, nil
}

// FS gives the file system of the root. Each read of it holds the check of the
// root.
func (r *Root) FS() fs.FS { return rootFS{r} }

// openIn preserves the caller's names for File.Name and PathError.
func (r *Root) openIn(f *File, err error, name string) (*File, error) {
	if err != nil {
		return nil, rootError("openat", name, err)
	}
	f.name = joinPath(r.name, name)
	return f, nil
}

// resolve gives the path of a name inside the root, and it rejects a name that
// leaves the root.
func (r *Root) resolve(op, name string) (string, error) {
	return r.check(op, name, true)
}

// resolveNoFollow does the same, and it leaves the last element of the path
// alone. A read of a symbolic link needs the link itself.
func (r *Root) resolveNoFollow(op, name string) (string, error) {
	return r.check(op, name, false)
}

func (r *Root) check(op, name string, follow bool) (string, error) {
	if err := checkPathEscapesInternal(r, name, !follow); err != nil {
		return "", &PathError{Op: op, Path: name, Err: err}
	}
	return joinPath(r.real, name), nil
}
