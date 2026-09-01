//go:build dart

package os

import (
	"errors"
	"io/fs"
	"syscall/dart"
)

// The file system view of a directory.
//
// DirFS gives the tree under 1 directory as an fs.FS. It holds no protection
// against a symbolic link that leaves the directory, exactly as Go states.
// Root holds that protection. Read docs/parity.md.

// Readlink gives the target of a symbolic link.
func Readlink(name string) (string, error) {
	target, cause := dart.ReadLink(name)
	if cause != "" {
		return "", &PathError{Op: "readlink", Path: name, Err: errors.New(cause)}
	}
	return target, nil
}

// DirFS gives the file system of the tree under dir.
//
// The result holds no protection against a path that leaves dir through a
// symbolic link, or against a change of dir under the program. Use Root for a
// tree that must stay closed.
func DirFS(dir string) fs.FS { return dirFS(dir) }

type dirFS string

func (dir dirFS) Open(name string) (fs.File, error) {
	full, err := dir.join(name)
	if err != nil {
		return nil, &PathError{Op: "open", Path: name, Err: err}
	}
	f, err := Open(full)
	if err != nil {
		return nil, rename(err, name)
	}
	return f, nil
}

func (dir dirFS) ReadFile(name string) ([]byte, error) {
	full, err := dir.join(name)
	if err != nil {
		return nil, &PathError{Op: "readfile", Path: name, Err: err}
	}
	b, err := ReadFile(full)
	if err != nil {
		return nil, rename(err, name)
	}
	return b, nil
}

func (dir dirFS) ReadDir(name string) ([]DirEntry, error) {
	full, err := dir.join(name)
	if err != nil {
		return nil, &PathError{Op: "readdir", Path: name, Err: err}
	}
	entries, err := ReadDir(full)
	if err != nil {
		return nil, rename(err, name)
	}
	return entries, nil
}

func (dir dirFS) Stat(name string) (fs.FileInfo, error) {
	full, err := dir.join(name)
	if err != nil {
		return nil, &PathError{Op: "stat", Path: name, Err: err}
	}
	info, err := Stat(full)
	if err != nil {
		return nil, rename(err, name)
	}
	return info, nil
}

func (dir dirFS) Lstat(name string) (fs.FileInfo, error) {
	full, err := dir.join(name)
	if err != nil {
		return nil, &PathError{Op: "lstat", Path: name, Err: err}
	}
	info, err := Lstat(full)
	if err != nil {
		return nil, rename(err, name)
	}
	return info, nil
}

func (dir dirFS) ReadLink(name string) (string, error) {
	full, err := dir.join(name)
	if err != nil {
		return "", &PathError{Op: "readlink", Path: name, Err: err}
	}
	target, err := Readlink(full)
	if err != nil {
		return "", rename(err, name)
	}
	return target, nil
}

// join gives the path of name inside dir.
func (dir dirFS) join(name string) (string, error) {
	if dir == "" {
		return "", errors.New("os: DirFS with empty root")
	}
	local, err := localize(name)
	if err != nil {
		return "", err
	}
	if IsPathSeparator(dir[len(dir)-1]) {
		return string(dir) + local, nil
	}
	return string(dir) + string(PathSeparator) + local, nil
}

// localize turns a slash separated name into a path of the platform. The
// separator of the platform is always the slash, thus the name changes only
// when it holds no valid form.
//
// os cannot read path/filepath, because that package reads os.
func localize(name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", ErrInvalid
	}
	for i := 0; i < len(name); i++ {
		if name[i] == 0 {
			return "", ErrInvalid
		}
	}
	return name, nil
}

// rename writes the name of the caller into a path error. DirFS joins the root
// and the name, and the report must hold the name alone.
func rename(err error, name string) error {
	var pe *PathError
	if errors.As(err, &pe) {
		pe.Path = name
	}
	return err
}
