// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Package os gives the file access and the process state of the standard
// library.
//
// The standard Go source depends on OS-specific syscall code. This overlay
// uses the runtime adapters.
//
// File operations are synchronous and can delay other goroutines.
package os

import (
	"errors"
	"io"
	"io/fs"
	"syscall"
	"syscall/dart"
	"time"
)

// Args holds the name of the program and each argument of the command line.
var Args = args()

func args() []string {
	return append([]string{"gart"}, dart.Args()...)
}

// FileMode holds the permission bits of a path.
//
// A call that makes a file takes the argument and drops it. A Go source thus
// compiles the same way with the go command and with gart.
//
// Chmod of attr.go writes the mode through the command of the platform, and Stat
// still reports a fixed mode.
type FileMode = fs.FileMode

// The bits of a FileMode. Each one comes from the package io/fs.
const (
	ModeDir        = fs.ModeDir
	ModeAppend     = fs.ModeAppend
	ModeExclusive  = fs.ModeExclusive
	ModeTemporary  = fs.ModeTemporary
	ModeSymlink    = fs.ModeSymlink
	ModeDevice     = fs.ModeDevice
	ModeNamedPipe  = fs.ModeNamedPipe
	ModeSocket     = fs.ModeSocket
	ModeSetuid     = fs.ModeSetuid
	ModeSetgid     = fs.ModeSetgid
	ModeCharDevice = fs.ModeCharDevice
	ModeSticky     = fs.ModeSticky
	ModeIrregular  = fs.ModeIrregular
	ModeType       = fs.ModeType
	ModePerm       = fs.ModePerm
)

// The flag set of OpenFile.
const (
	O_RDONLY int = syscall.O_RDONLY
	O_WRONLY int = syscall.O_WRONLY
	O_RDWR   int = syscall.O_RDWR
	O_APPEND int = syscall.O_APPEND
	O_CREATE int = syscall.O_CREAT
	O_EXCL   int = syscall.O_EXCL
	O_TRUNC  int = syscall.O_TRUNC
)

// ErrNotExist is the error of a path that does not exist. Go compares an error
// with it through errors.Is.
var ErrNotExist = fs.ErrNotExist

// ErrClosed is the error of an operation on a closed file.
var ErrClosed = fs.ErrClosed

// PathError names the operation, the path, and the cause of a failure.
type PathError = fs.PathError

// pathError builds the error of an operation on a path. It gives nil when the
// cause is empty.
func pathError(op, path, cause string) error {
	if cause == "" {
		return nil
	}
	return &PathError{Op: op, Path: path, Err: causeError(cause)}
}

// sysError names a failure of the system. Go holds a syscall.Errno in a
// PathError, and the text of that value comes from the table of the system.
// The sentinel of the package writes another text: ErrNotExist gives "file
// does not exist" where the system gives "no such file or directory". Thus
// this type keeps the text of the system, and errors.Is still finds the
// sentinel, exactly as syscall.Errno does.
type sysError struct {
	text string
	kind error
}

func (e *sysError) Error() string { return e.text }

func (e *sysError) Is(target error) bool { return target == e.kind }

// causeError maps the text of the Dart side to an error value. The text of
// the Dart side holds the message of the system with a capital first letter,
// thus the map writes the text of Go.
func causeError(cause string) error {
	switch {
	case contains(cause, "No such file or directory") ||
		contains(cause, "no such file or directory"):
		return &sysError{text: "no such file or directory", kind: ErrNotExist}
	case contains(cause, "cannot find the file"):
		return &sysError{
			text: "The system cannot find the file specified.",
			kind: ErrNotExist,
		}
	case contains(cause, "Not a directory") ||
		contains(cause, "not a directory"):
		return syscall.ENOTDIR
	case contains(cause, "Is a directory") || contains(cause, "is a directory"):
		return &sysError{text: "is a directory"}
	case contains(cause, "Directory not empty") ||
		contains(cause, "directory not empty"):
		return &sysError{text: "directory not empty", kind: ErrExist}
	case contains(cause, "Invalid argument") ||
		contains(cause, "invalid argument"):
		return &sysError{text: "invalid argument", kind: ErrInvalid}
	case contains(cause, "Bad file descriptor") ||
		contains(cause, "bad file descriptor"):
		return &sysError{text: "bad file descriptor"}
	case contains(cause, "file already closed"):
		return ErrClosed
	case contains(cause, "Operation not permitted") ||
		contains(cause, "operation not permitted"):
		return &sysError{text: "operation not permitted", kind: ErrPermission}
	case contains(cause, "Permission denied") ||
		contains(cause, "permission denied"):
		// The text of ErrPermission and the text of EACCES agree, thus the
		// sentinel itself fits here.
		return ErrPermission
	case contains(cause, "File exists") || contains(cause, "file exists"):
		return errExist
	}
	return errors.New(cause)
}

func contains(text, part string) bool {
	if len(part) > len(text) {
		return false
	}
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return true
		}
	}
	return false
}

// IsNotExist reports whether the error names a path that does not exist.
func IsNotExist(err error) bool { return underlyingErrorIs(err, ErrNotExist) }

// File is an open file. The zero value is not usable.
type File struct {
	*file
}

type file struct {
	handle int
	name   string
	// std marks standard output or standard error. Such a file holds no handle.
	std int
	// closed marks a file after Close, thus a later call reports ErrClosed.
	closed bool
	// Directory enumeration still uses the path. dirFD owns the native descriptor.
	dir   bool
	dirFD int
	// dirPos counts the entries that ReadDir gave.
	dirPos int
}

// The value of the field std.
const (
	stdNone   = 0
	stdOut    = 1
	stdErr    = 2
	stdInFile = 3
)

// Stdin, Stdout, and Stderr are the 3 files that the process holds at its start.
var (
	Stdin  = &File{&file{std: stdInFile, name: "/dev/stdin"}}
	Stdout = &File{&file{std: stdOut, name: "/dev/stdout"}}
	Stderr = &File{&file{std: stdErr, name: "/dev/stderr"}}
)

// Pipe gives a connected pair of files. The Dart platform holds no pipe, thus
// the call gives an error.
func Pipe() (*File, *File, error) {
	return nil, nil, &PathError{Op: "pipe", Err: errors.New("not supported")}
}

// Name gives the name that the caller of Open gave. Go reads the field with no
// test of the receiver, thus a nil pointer panics here too.
func (f *File) Name() string { return f.name }

// Fd returns the native descriptor. The caller must not close it.
func (f *File) Fd() uintptr {
	if f == nil || f.closed {
		return ^uintptr(0)
	}
	if f.dir {
		return uintptr(dart.DirectoryFd(f.dirFD))
	}
	return uintptr(dart.FileFd(f.handle, f.std))
}

// checkValid gives ErrInvalid for a nil receiver. Go tests the receiver of
// each method of a File, thus a call on a nil pointer gives an error and no
// panic.
func (f *File) checkValid(op string) error {
	if f == nil {
		return ErrInvalid
	}
	if f.closed {
		return &PathError{Op: op, Path: f.name, Err: ErrClosed}
	}
	return nil
}

// Write writes the bytes and gives the count.
func (f *File) Write(p []byte) (int, error) {
	if err := f.checkValid("write"); err != nil {
		return 0, err
	}
	switch f.std {
	case stdOut:
		dart.Print(string(p))
		return len(p), nil
	case stdErr:
		dart.Stderr(string(p))
		return len(p), nil
	case stdInFile:
		return 0, &PathError{Op: "write", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	if f.dir {
		return 0, &PathError{Op: "write", Path: f.name, Err: errBadDescriptor}
	}
	n, cause := dart.FileWrite(f.handle, p)
	return n, pathError("write", f.name, cause)
}

var errBadDescriptor = &sysError{text: "bad file descriptor"}

var errIsDir = &sysError{text: "is a directory"}

// WriteString writes the text and gives the count of bytes.
func (f *File) WriteString(s string) (int, error) { return f.Write([]byte(s)) }

// Read reads bytes into the slice and gives the count. It gives io.EOF at the
// end of the file, and the caller compares the error with that value.
func (f *File) Read(p []byte) (int, error) {
	if err := f.checkValid("read"); err != nil {
		return 0, err
	}
	if f.std == stdInFile {
		n, cause := dart.StdinRead(p)
		if cause != "" {
			return n, pathError("read", f.name, cause)
		}
		if n == 0 {
			return 0, io.EOF
		}
		return n, nil
	}
	if f.std != stdNone {
		return 0, &PathError{Op: "read", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	if f.dir {
		return 0, &PathError{Op: "read", Path: f.name, Err: errIsDir}
	}
	n, cause := dart.FileRead(f.handle, p)
	if cause != "" {
		return n, pathError("read", f.name, cause)
	}
	if n == 0 && len(p) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

// ReadAt reads at the offset. It does not move the read position of the file,
// thus the call reads the position first and writes it again.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if err := f.checkValid("readat"); err != nil {
		return 0, err
	}
	if f.std != stdNone {
		return 0, &PathError{Op: "read", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	if f.dir {
		return 0, &PathError{Op: "read", Path: f.name, Err: errIsDir}
	}
	if off < 0 {
		return 0, &PathError{Op: "readat", Path: f.name, Err: errors.New("negative offset")}
	}
	pos, cause := dart.FileSeek(f.handle, 0, dart.SeekCurrent)
	if cause != "" {
		return 0, pathError("readat", f.name, cause)
	}
	if _, cause := dart.FileSeek(f.handle, off, dart.SeekStart); cause != "" {
		return 0, pathError("readat", f.name, cause)
	}
	read := 0
	var err error
	for read < len(p) {
		n, cause := dart.FileRead(f.handle, p[read:])
		if cause != "" {
			err = pathError("readat", f.name, cause)
			break
		}
		if n == 0 {
			err = io.EOF
			break
		}
		read += n
	}
	if _, cause := dart.FileSeek(f.handle, pos, dart.SeekStart); cause != "" && err == nil {
		err = pathError("readat", f.name, cause)
	}
	return read, err
}

// WriteAt writes at the offset. It does not move the write position of the
// file, thus the call reads the position first and writes it again.
func (f *File) WriteAt(p []byte, off int64) (int, error) {
	if err := f.checkValid("writeat"); err != nil {
		return 0, err
	}
	if f.std != stdNone {
		return 0, &PathError{Op: "write", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	if f.dir {
		return 0, &PathError{Op: "write", Path: f.name, Err: errBadDescriptor}
	}
	if off < 0 {
		return 0, &PathError{Op: "writeat", Path: f.name, Err: errors.New("negative offset")}
	}
	pos, cause := dart.FileSeek(f.handle, 0, dart.SeekCurrent)
	if cause != "" {
		return 0, pathError("writeat", f.name, cause)
	}
	if _, cause := dart.FileSeek(f.handle, off, dart.SeekStart); cause != "" {
		return 0, pathError("writeat", f.name, cause)
	}
	written, cause := dart.FileWrite(f.handle, p)
	var err error
	if cause != "" {
		err = pathError("writeat", f.name, cause)
	}
	if _, cause := dart.FileSeek(f.handle, pos, dart.SeekStart); cause != "" && err == nil {
		err = pathError("writeat", f.name, cause)
	}
	return written, err
}

// ReadFrom writes each byte of the reader to the file and gives the count.
//
// Go reaches the copy call of the kernel here. The loop below is the fallback
// that Go also takes when that call does not fit.
func (f *File) ReadFrom(r io.Reader) (int64, error) {
	if err := f.checkValid("readfrom"); err != nil {
		return 0, err
	}
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, rerr := r.Read(buf)
		if n > 0 {
			written, werr := f.Write(buf[:n])
			total += int64(written)
			if werr != nil {
				return total, werr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				return total, nil
			}
			return total, rerr
		}
	}
}

// WriteTo writes each byte of the file to the writer and gives the count.
func (f *File) WriteTo(w io.Writer) (int64, error) {
	if err := f.checkValid("writeto"); err != nil {
		return 0, err
	}
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			written, werr := w.Write(buf[:n])
			total += int64(written)
			if werr != nil {
				return total, werr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				return total, nil
			}
			return total, rerr
		}
	}
}

// EOF is the error of the end of a file. The package io holds the same value,
// and a caller compares an error with either name.
var EOF = io.EOF

// Seek moves the read position and gives the new offset from the start.
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if err := f.checkValid("seek"); err != nil {
		return 0, err
	}
	if f.std != stdNone {
		return 0, &PathError{Op: "seek", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	pos, cause := dart.FileSeek(f.handle, offset, whence)
	return pos, pathError("seek", f.name, cause)
}

func (f *File) Close() error {
	if f == nil {
		return ErrInvalid
	}
	if f.file == nil {
		return syscall.EINVAL
	}
	if f.std != stdNone {
		return nil
	}
	if f.closed {
		return &PathError{Op: "close", Path: f.name, Err: ErrClosed}
	}
	f.closed = true
	if f.dir {
		if errno := dart.DirectoryClose(f.dirFD); errno != 0 {
			return &PathError{Op: "close", Path: f.name, Err: syscall.Errno(errno)}
		}
		return nil
	}
	return pathError("close", f.name, dart.FileClose(f.handle))
}

// Sync writes each held byte to the storage.
func (f *File) Sync() error {
	if err := f.checkValid("sync"); err != nil {
		return err
	}
	if f.std != stdNone {
		return nil
	}
	return pathError("sync", f.name, dart.FileSync(f.handle))
}

// Stat reads the metadata of the file.
func (f *File) Stat() (FileInfo, error) {
	if err := f.checkValid("stat"); err != nil {
		return nil, err
	}
	return Stat(f.name)
}

// Open opens a file or directory for a read.
func Open(name string) (*File, error) {
	return openMode(name, dart.OpenRead, "open")
}

// Create makes a file, or it replaces the content of an existing file.
func Create(name string) (*File, error) {
	return openMode(name, dart.OpenWrite, "open")
}

// existsError names a path that already exists. Go writes the text of the
// system, and ErrExist writes another text.
type existsError struct{}

func (existsError) Error() string { return "file exists" }

func (existsError) Is(target error) bool { return target == ErrExist }

var errExist error = existsError{}

// OpenFile opens a file with a flag set. The permission argument reaches no
// operation of the Dart side.
//
// The flag set selects 1 of the 4 modes of the Dart side: a read, a write that
// replaces the content, a write at the end, or a read and a write that keeps
// the content. Only O_TRUNC replaces the content, exactly as in Go.
//
// O_EXCL takes a read of the path before the open. The 2 steps are not 1
// operation of the system. A file that another program makes between them thus
// still passes.
func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	if contains(name, "\x00") {
		return nil, &PathError{Op: "open", Path: name, Err: syscall.EINVAL}
	}
	if flag&O_CREATE != 0 && flag&O_EXCL != 0 {
		if _, err := Lstat(name); err == nil {
			return nil, &PathError{Op: "open", Path: name, Err: errExist}
		}
	}
	// A create gives the file the bits of perm without the bits of the mask.
	// dart:io writes no mode, thus the chmod stands after the open, and it
	// runs only for a path that the call itself makes.
	fresh := false
	if _, err := Lstat(name); err != nil {
		if flag&O_CREATE == 0 {
			// Each write mode of dart:io makes a file that does not exist,
			// and Go makes 1 with O_CREATE alone.
			return nil, &PathError{Op: "open", Path: name, Err: underlyingError(err)}
		}
		fresh = true
	}
	mode := dart.OpenRead
	switch {
	case flag&O_APPEND != 0:
		mode = dart.OpenAppend
	case flag&O_TRUNC != 0:
		mode = dart.OpenWrite
	case flag&(O_WRONLY|O_RDWR) != 0:
		mode = dart.OpenReadWrite
	}
	f, err := openMode(name, mode, "open")
	if err != nil || !fresh {
		return f, err
	}
	applyPerm(name, perm)
	return f, nil
}

// applyPerm writes the permission bits of a fresh file. It drops each bit of
// the mask of the process, exactly as a create of Go does. A default mode
// needs no command, thus the common path runs no process.
func applyPerm(name string, perm FileMode) {
	bits := syscallMode(perm) &^ dart.Umask()
	if bits == 0o644 {
		return
	}
	dart.Chmod(name, bits)
}

func openMode(name string, mode int, op string) (*File, error) {
	if contains(name, "\x00") {
		return nil, &PathError{Op: op, Path: name, Err: syscall.EINVAL}
	}
	if mode == dart.OpenRead {
		if info, err := Stat(name); err == nil && info.IsDir() {
			fd, errno := dart.DirectoryOpen(name)
			if errno != 0 {
				return nil, &PathError{Op: op, Path: name, Err: syscall.Errno(errno)}
			}
			return &File{&file{handle: -1, name: name, dir: true, dirFD: fd}}, nil
		}
	}
	handle, cause := dart.FileOpen(name, mode)
	if cause != "" {
		return nil, pathError(op, name, cause)
	}
	return &File{&file{handle: handle, name: name}}, nil
}

// ReadFile reads a whole file.
func ReadFile(name string) ([]byte, error) {
	if contains(name, "\x00") {
		return nil, &PathError{Op: "open", Path: name, Err: syscall.EINVAL}
	}
	data, cause := dart.ReadFile(name)
	return data, pathError("open", name, cause)
}

// WriteFile writes a whole file. It replaces the content of an existing file,
// and it gives a fresh file the permission bits of perm without the bits of
// the mask of the process.
func WriteFile(name string, data []byte, perm FileMode) error {
	if contains(name, "\x00") {
		return &PathError{Op: "open", Path: name, Err: syscall.EINVAL}
	}
	fresh := false
	if _, err := Lstat(name); err != nil {
		fresh = true
	}
	if cause := dart.WriteFile(name, data); cause != "" {
		return pathError("open", name, cause)
	}
	if fresh {
		applyPerm(name, perm)
	}
	return nil
}

// FileInfo holds the metadata of a path.
type FileInfo = fs.FileInfo

// DirEntry holds 1 member of a directory.
type DirEntry = fs.DirEntry

// fileStat holds the metadata that the Dart runtime gives.
//
// The permission bits reach no operation. Dart names no such bit, thus each
// file reads 0644 and each directory reads 0755.
type fileStat struct {
	name  string
	size  int64
	dir   bool
	link  bool
	nanos int64
}

func (i *fileStat) Name() string { return i.name }

func (i *fileStat) Size() int64 { return i.size }

func (i *fileStat) IsDir() bool { return i.dir }

func (i *fileStat) Sys() any { return nil }

func (i *fileStat) Mode() FileMode {
	if i.dir {
		return ModeDir | 0o755
	}
	if i.link {
		return ModeSymlink | 0o755
	}
	return 0o644
}

func (i *fileStat) ModTime() time.Time { return time.Unix(0, i.nanos) }

// ModNanos gives the time of the last change, in nanoseconds since the start of
// the Unix epoch. Go holds no such method, and ModTime gives the same instant.
func (i *fileStat) ModNanos() int64 { return i.nanos }

// Stat reads the metadata of a path.
func Stat(name string) (FileInfo, error) {
	if contains(name, "\x00") {
		return nil, &PathError{Op: "stat", Path: name, Err: syscall.EINVAL}
	}
	size, dir, nanos, cause := dart.Stat(name)
	if cause != "" {
		return nil, pathError("stat", name, cause)
	}
	return &fileStat{name: base(name), size: size, dir: dir, nanos: nanos}, nil
}

// Lstat reads the metadata of a path, and it follows no symbolic link. The
// size of a link is the length of its target, and the time of a link is 0,
// because dart:io gives no metadata of a link itself.
func Lstat(name string) (FileInfo, error) {
	if contains(name, "\x00") {
		return nil, &PathError{Op: "lstat", Path: name, Err: syscall.EINVAL}
	}
	size, dir, link, nanos, cause := dart.Lstat(name)
	if cause != "" {
		return nil, pathError("lstat", name, cause)
	}
	return &fileStat{name: base(name), size: size, dir: dir, link: link, nanos: nanos}, nil
}

// LinkError names the operation, the 2 paths, and the cause of a failure of a
// link operation.
type LinkError struct {
	Op  string
	Old string
	New string
	Err error
}

func (e *LinkError) Error() string {
	return e.Op + " " + e.Old + " " + e.New + ": " + e.Err.Error()
}

func (e *LinkError) Unwrap() error { return e.Err }

// Symlink makes a symbolic link at newname, and the link points at oldname.
func Symlink(oldname, newname string) error {
	if contains(oldname, "\x00") || contains(newname, "\x00") {
		return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: syscall.EINVAL}
	}
	cause := dart.Symlink(oldname, newname)
	if cause == "" {
		return nil
	}
	return &LinkError{Op: "symlink", Old: oldname, New: newname, Err: causeError(cause)}
}

// dirEntry holds 1 member of a directory. It reads the metadata only when a
// caller asks for it.
type dirEntry struct {
	dir  string
	name string
	stat *fileStat
}

func (d *dirEntry) Name() string { return d.name }

func (d *dirEntry) IsDir() bool {
	info, err := d.Info()
	return err == nil && info.IsDir()
}

func (d *dirEntry) Type() FileMode {
	info, err := d.Info()
	if err != nil {
		return 0
	}
	return info.Mode().Type()
}

func (d *dirEntry) Info() (FileInfo, error) {
	if d.stat != nil {
		return d.stat, nil
	}
	path := d.dir
	if path != "" && path[len(path)-1] != '/' {
		path += "/"
	}
	// Go reads the metadata of the entry itself, and it follows no link there.
	info, err := Lstat(path + d.name)
	if err != nil {
		return nil, err
	}
	d.stat = info.(*fileStat)
	return d.stat, nil
}

// ReadDir gives each member of the directory, ordered by name.
func ReadDir(name string) ([]DirEntry, error) {
	names, err := ReadNames(name)
	if err != nil {
		return nil, err
	}
	out := make([]DirEntry, len(names))
	for i, n := range names {
		out[i] = &dirEntry{dir: name, name: n}
	}
	return out, nil
}

// ReadDir gives the next members of the directory of the file. A count that
// is not positive gives each member that stays, and a later call gives none. A
// positive count gives at most that many, and io.EOF at the end.
func (f *File) ReadDir(n int) ([]DirEntry, error) {
	if err := f.checkValid("readdir"); err != nil {
		return nil, err
	}
	all, err := ReadDir(f.name)
	if err != nil {
		return nil, err
	}
	if f.dirPos > len(all) {
		f.dirPos = len(all)
	}
	rest := all[f.dirPos:]
	if n <= 0 {
		f.dirPos = len(all)
		return rest, nil
	}
	if len(rest) == 0 {
		return nil, io.EOF
	}
	if n < len(rest) {
		rest = rest[:n]
	}
	f.dirPos += len(rest)
	return rest, nil
}

// Readdir reads the directory and gives the metadata of each entry. A count of
// 0 or less gives each entry. Go marks the method as older than ReadDir, and
// http.File reads it.
func (f *File) Readdir(n int) ([]FileInfo, error) {
	if err := f.checkValid("readdir"); err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(n)
	if err != nil {
		return nil, err
	}
	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return out, err
		}
		out = append(out, info)
	}
	return out, nil
}

// Readdirnames reads the directory and gives the name of each entry.
func (f *File) Readdirnames(n int) ([]string, error) {
	if err := f.checkValid("readdirnames"); err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(n)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out, nil
}

// base gives the last part of a path. A trailing separator does not count,
// exactly as in Go.
func base(path string) string {
	sep := dart.PathSeparator()
	if sep == "" || path == "" {
		return path
	}
	c := sep[0]
	i := len(path) - 1
	for i > 0 && path[i] == c {
		path = path[:i]
		i--
	}
	for i--; i >= 0; i-- {
		if path[i] == c {
			return path[i+1:]
		}
	}
	return path
}

func Remove(name string) error {
	if contains(name, "\x00") {
		return &PathError{Op: "remove", Path: name, Err: syscall.EINVAL}
	}
	return pathError("remove", name, dart.Remove(name))
}

// RemoveAll removes a path and each path inside it. It gives no error when the
// path does not exist.
func RemoveAll(name string) error {
	if name == "" {
		return nil
	}
	for len(name) > 1 && IsPathSeparator(name[len(name)-1]) {
		name = name[:len(name)-1]
	}
	if endsWithDot(name) || contains(name, "\x00") {
		return &PathError{Op: "RemoveAll", Path: name, Err: syscall.EINVAL}
	}
	op, path, cause := dart.RemoveAll(name)
	return pathError(op, path, cause)
}

// Mkdir makes 1 directory. The permission argument reaches no operation of the
// Dart side. A path that exists gives ErrExist, because the Dart side accepts
// such a path with no report.
func Mkdir(name string, perm FileMode) error {
	if contains(name, "\x00") {
		return &PathError{Op: "mkdir", Path: name, Err: syscall.EINVAL}
	}
	if _, err := Lstat(name); err == nil {
		return &PathError{Op: "mkdir", Path: name, Err: errExist}
	}
	return pathError("mkdir", name, dart.Mkdir(name, false))
}

// MkdirAll makes a directory and each parent directory of it. The logic comes
// from the file path.go of Go, thus a parent that is a file gives the error of
// Go with the path of that parent.
func MkdirAll(name string, perm FileMode) error {
	info, err := Stat(name)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return &PathError{Op: "mkdir", Path: name, Err: &sysError{text: "not a directory"}}
	}
	i := len(name)
	for i > 0 && IsPathSeparator(name[i-1]) {
		i--
	}
	j := i
	for j > 0 && !IsPathSeparator(name[j-1]) {
		j--
	}
	if j > 1 {
		if err := MkdirAll(name[:j-1], perm); err != nil {
			return err
		}
	}
	err = Mkdir(name, perm)
	if err != nil {
		if info, err1 := Lstat(name); err1 == nil && info.IsDir() {
			return nil
		}
		return err
	}
	return nil
}

// Rename moves a path. A failure gives a LinkError that names the 2 paths,
// exactly as Go does.
func Rename(from, to string) error {
	if contains(from, "\x00") || contains(to, "\x00") {
		return &LinkError{Op: "rename", Old: from, New: to, Err: syscall.EINVAL}
	}
	if info, err := Lstat(to); err == nil && info.IsDir() {
		if _, err := Lstat(from); err != nil {
			return &LinkError{Op: "rename", Old: from, New: to, Err: underlyingError(err)}
		}
		same, cause := dart.FileIdentical(from, to)
		if cause != "" {
			return &LinkError{Op: "rename", Old: from, New: to, Err: causeError(cause)}
		}
		if from == to || !same {
			return &LinkError{Op: "rename", Old: from, New: to, Err: syscall.EEXIST}
		}
	}
	cause := dart.Rename(from, to)
	if cause == "" {
		return nil
	}
	return &LinkError{Op: "rename", Old: from, New: to, Err: causeError(cause)}
}

// ReadNames reads the names inside a directory. Each name holds no directory
// part, and the list is in the order of the name.
//
// Go holds ReadDir with a slice of DirEntry. A DirEntry holds 4 methods, and 1
// of them reads the metadata again. This function replaces it, thus a source
// that calls it does not compile with the go command.
func ReadNames(name string) ([]string, error) {
	if contains(name, "\x00") {
		return nil, &PathError{Op: "open", Path: name, Err: syscall.EINVAL}
	}
	names, cause := dart.ReadDir(name)
	return names, pathError("open", name, cause)
}

func Getenv(name string) string {
	value, _ := syscall.Getenv(name)
	return value
}

func LookupEnv(name string) (string, bool) { return syscall.Getenv(name) }

func Setenv(name, value string) error {
	return NewSyscallError("setenv", syscall.Setenv(name, value))
}

func Unsetenv(name string) error { return syscall.Unsetenv(name) }

func Environ() []string { return syscall.Environ() }

func Getwd() (string, error) {
	path, cause := dart.Getwd()
	if cause != "" {
		return "", errors.New(cause)
	}
	return path, nil
}

func Chdir(path string) error {
	if contains(path, "\x00") {
		return &PathError{Op: "chdir", Path: path, Err: syscall.EINVAL}
	}
	return pathError("chdir", path, dart.Chdir(path))
}

// TempDir gives the directory that the system holds for a temporary file.
func TempDir() string { return dart.TempDir() }

// tempSeq numbers each temporary name of the process.
var tempSeq int

// tempName builds a name that no other call of the process gives. Go reads a
// random number, and 1 isolate runs each goroutine, thus a counter is enough.
func tempName(dir, pattern string) string {
	tempSeq++
	if dir == "" {
		dir = TempDir()
	}
	prefix, suffix := pattern, ""
	for i := len(pattern) - 1; i >= 0; i-- {
		if pattern[i] == '*' {
			prefix, suffix = pattern[:i], pattern[i+1:]
			break
		}
	}
	return dir + dart.PathSeparator() + prefix + itoa(dart.NowNanos()) + itoa(int64(tempSeq)) + suffix
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

// CreateTemp makes a new file in the directory. An empty directory reads the
// directory of the system. The last star of the pattern holds the place of the
// random part, and a pattern with no star takes the part at the end.
func CreateTemp(dir, pattern string) (*File, error) {
	for i := 0; i < len(pattern); i++ {
		if IsPathSeparator(pattern[i]) {
			return nil, &PathError{Op: "createtemp", Path: pattern, Err: errors.New("pattern contains path separator")}
		}
	}
	for i := 0; i < 100; i++ {
		name := tempName(dir, pattern)
		if _, err := Stat(name); err == nil {
			continue
		}
		return Create(name)
	}
	return nil, pathError("createtemp", dir, "no free name")
}

// MkdirTemp makes a new directory. It reads the pattern of CreateTemp.
func MkdirTemp(dir, pattern string) (string, error) {
	if dir == "" {
		dir = TempDir()
	}
	for i := 0; i < len(pattern); i++ {
		if IsPathSeparator(pattern[i]) {
			return "", &PathError{Op: "mkdirtemp", Path: pattern, Err: errors.New("pattern contains path separator")}
		}
	}
	for i := 0; i < 100; i++ {
		name := tempName(dir, pattern)
		if _, err := Stat(name); err == nil {
			continue
		}
		if err := Mkdir(name, 0o700); err != nil {
			if IsNotExist(err) {
				if _, dirErr := Stat(dir); IsNotExist(dirErr) {
					return "", dirErr
				}
			}
			return "", err
		}
		return name, nil
	}
	return "", pathError("mkdirtemp", dir, "no free name")
}

// Exit stops the program with a status.
func Exit(code int) { dart.Exit(code) }
