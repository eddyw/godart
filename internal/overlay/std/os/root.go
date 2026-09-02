//go:build dart

package os

import (
	"errors"
	"io/fs"
	"syscall/dart"
	"time"
)

// Root holds a directory, and each operation of it stays inside that
// directory.
//
// Go opens the directory 1 time and reaches each name under it with openat,
// thus no path can leave the tree. dart:io holds no such call, and the check
// below resolves each symbolic link of the path and compares the result with
// the root.
//
// The 2 forms differ in 1 point: the check of this package is not atomic. A
// rename of a directory of the path, between the check and the operation, can
// still reach a name outside the root. Read docs/parity.md.

// errPathEscapes names a path that leaves the root.
var errPathEscapes = errors.New("path escapes from parent")

// A Root holds an open directory.
type Root struct {
	name string
	// real holds the name after each symbolic link resolved. The check of each
	// operation compares against it.
	real   string
	closed bool
}

// OpenRoot opens a directory and gives the Root of it.
func OpenRoot(name string) (*Root, error) {
	info, err := Stat(name)
	if err != nil {
		return nil, &PathError{Op: "open", Path: name, Err: err}
	}
	if !info.IsDir() {
		return nil, &PathError{Op: "open", Path: name, Err: errors.New("not a directory")}
	}
	real, cause := dart.ResolveSymlinks(name)
	if cause != "" {
		return nil, &PathError{Op: "open", Path: name, Err: errors.New(cause)}
	}
	return &Root{name: name, real: real}, nil
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
	full, err := r.resolve("open", name)
	if err != nil {
		return nil, err
	}
	f, cause := Open(full)
	return openIn(f, cause, name)
}

// OpenFile opens a file of the root with the flag and the permission.
func (r *Root) OpenFile(name string, flag int, perm FileMode) (*File, error) {
	full, err := r.resolve("open", name)
	if err != nil {
		return nil, err
	}
	f, cause := OpenFile(full, flag, perm)
	return openIn(f, cause, name)
}

// Create makes a file of the root, or it truncates the file that stands there.
func (r *Root) Create(name string) (*File, error) {
	full, err := r.resolve("create", name)
	if err != nil {
		return nil, err
	}
	f, cause := Create(full)
	return openIn(f, cause, name)
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
	return out, nil
}

// Stat reads the metadata of a name of the root.
func (r *Root) Stat(name string) (FileInfo, error) {
	full, err := r.resolve("stat", name)
	if err != nil {
		return nil, err
	}
	info, cause := Stat(full)
	if cause != nil {
		return nil, rename(cause, name)
	}
	return info, nil
}

// Lstat reads the metadata of a name of the root, and it follows no symbolic
// link of the last element.
func (r *Root) Lstat(name string) (FileInfo, error) {
	full, err := r.resolve("lstat", name)
	if err != nil {
		return nil, err
	}
	info, cause := Lstat(full)
	if cause != nil {
		return nil, rename(cause, name)
	}
	return info, nil
}

// ReadFile reads the whole content of a file of the root.
func (r *Root) ReadFile(name string) ([]byte, error) {
	full, err := r.resolve("readfile", name)
	if err != nil {
		return nil, err
	}
	b, cause := ReadFile(full)
	if cause != nil {
		return nil, rename(cause, name)
	}
	return b, nil
}

// WriteFile writes the data to a file of the root.
func (r *Root) WriteFile(name string, data []byte, perm FileMode) error {
	full, err := r.resolve("writefile", name)
	if err != nil {
		return err
	}
	return rename(WriteFile(full, data, perm), name)
}

// Mkdir makes 1 directory of the root.
func (r *Root) Mkdir(name string, perm FileMode) error {
	full, err := r.resolve("mkdir", name)
	if err != nil {
		return err
	}
	return rename(Mkdir(full, perm), name)
}

// MkdirAll makes a directory of the root and each parent of it.
func (r *Root) MkdirAll(name string, perm FileMode) error {
	full, err := r.resolve("mkdirall", name)
	if err != nil {
		return err
	}
	return rename(MkdirAll(full, perm), name)
}

// Remove removes 1 name of the root.
func (r *Root) Remove(name string) error {
	full, err := r.resolve("remove", name)
	if err != nil {
		return err
	}
	return rename(Remove(full), name)
}

// RemoveAll removes a name of the root and each name under it.
func (r *Root) RemoveAll(name string) error {
	full, err := r.resolve("removeall", name)
	if err != nil {
		return err
	}
	return rename(RemoveAll(full), name)
}

// Rename moves a name of the root to another name of the root.
func (r *Root) Rename(oldname, newname string) error {
	from, err := r.resolve("rename", oldname)
	if err != nil {
		return err
	}
	to, err := r.resolve("rename", newname)
	if err != nil {
		return err
	}
	return Rename(from, to)
}

// Symlink makes a symbolic link of the root. The target reads as Go writes it,
// and the check of the root runs at each read of the link.
func (r *Root) Symlink(oldname, newname string) error {
	full, err := r.resolve("symlink", newname)
	if err != nil {
		return err
	}
	return rename(Symlink(oldname, full), newname)
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

// Link makes newname of the root as a hard link of oldname of the root.
//
// A symbolic link at oldname gives an error. The platform of the surface holds
// no link of a link.
func (r *Root) Link(oldname, newname string) error {
	from, err := r.resolveNoFollow("linkat", oldname)
	if err != nil {
		return err
	}
	if info, cause := Lstat(from); cause == nil && info.Mode()&ModeSymlink != 0 {
		return &PathError{Op: "linkat", Path: oldname,
			Err: errors.New("cannot create a hard link to a symlink")}
	}
	to, err := r.resolveNoFollow("linkat", newname)
	if err != nil {
		return err
	}
	if cause := Link(from, to); cause != nil {
		return &LinkError{Op: "linkat", Old: oldname, New: newname,
			Err: underlyingError(cause)}
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
	full, err := r.resolveNoFollow("readlink", name)
	if err != nil {
		return "", err
	}
	target, cause := Readlink(full)
	if cause != nil {
		return "", rename(cause, name)
	}
	return target, nil
}

// FS gives the file system of the root. Each read of it holds the check of the
// root.
func (r *Root) FS() fs.FS { return rootFS{r} }

type rootFS struct{ root *Root }

func (f rootFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}
	if name == "." {
		return f.root.Open(".")
	}
	return f.root.Open(name)
}

func (f rootFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "stat", Path: name, Err: ErrInvalid}
	}
	return f.root.Stat(name)
}

func (f rootFS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &PathError{Op: "readfile", Path: name, Err: ErrInvalid}
	}
	return f.root.ReadFile(name)
}

// openIn writes the name of the caller into the error of an open.
func openIn(f *File, err error, name string) (*File, error) {
	if err != nil {
		return nil, rename(err, name)
	}
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
	if r.closed {
		return "", &PathError{Op: op, Path: name, Err: ErrClosed}
	}
	if name == "." {
		return r.name, nil
	}
	local, err := localize(name)
	if err != nil {
		return "", &PathError{Op: op, Path: name, Err: err}
	}
	full := r.name + string(PathSeparator) + local
	probe := full
	if !follow {
		probe = parentOf(full)
	}
	real, cause := dart.ResolveSymlinks(probe)
	if cause != "" {
		// The name does not exist yet. The parent of it must stand inside the
		// root, thus a write through a link that leaves the root fails.
		real, cause = dart.ResolveSymlinks(parentOf(probe))
		if cause != "" {
			return "", &PathError{Op: op, Path: name, Err: ErrNotExist}
		}
	}
	if !inside(r.real, real) {
		return "", &PathError{Op: op, Path: name, Err: errPathEscapes}
	}
	return full, nil
}

// parentOf gives the path of the directory that holds the path.
func parentOf(p string) string {
	for i := len(p) - 1; i > 0; i-- {
		if IsPathSeparator(p[i]) {
			return p[:i]
		}
	}
	return string(PathSeparator)
}

// inside reports whether the path stands under the root.
func inside(root, p string) bool {
	if p == root {
		return true
	}
	if len(p) <= len(root) || !IsPathSeparator(p[len(root)]) {
		return false
	}
	return p[:len(root)] == root
}
