//go:build dart

// Package os gives the file access and the process state of the standard
// library.
//
// The compiler cannot read the real source. That source reaches the kernel
// through the package syscall, and the Dart VM gives no such access. Each
// operation below calls the interop package, and the Dart side uses dart:io.
//
// Each operation is synchronous. Go blocks the goroutine at the same point, and
// a synchronous call keeps the block analysis of stage 3 simple.
package os

import (
	"errors"
	"syscall/dart"
)

// Args holds the name of the program and each argument of the command line.
var Args = args()

func args() []string {
	return append([]string{"gart"}, dart.Args()...)
}

// FileMode holds the permission bits of a path.
//
// The Dart side gives no control of the permission. Each function below takes
// the argument and drops it, thus a Go source compiles the same way with the go
// command and with gart.
type FileMode uint32

// The permission bits that a Go program writes most.
const (
	ModePerm FileMode = 0o777
	ModeDir  FileMode = 1 << 31
)

func (m FileMode) IsDir() bool { return m&ModeDir != 0 }

func (m FileMode) Perm() FileMode { return m & 0o777 }

// The flag set of OpenFile.
const (
	O_RDONLY = 0
	O_WRONLY = 1
	O_RDWR   = 2
	O_APPEND = 1 << 10
	O_CREATE = 1 << 6
	O_EXCL   = 1 << 7
	O_TRUNC  = 1 << 9
)

// ErrNotExist is the error of a path that does not exist. Go compares an error
// with it through errors.Is.
var ErrNotExist = errors.New("file does not exist")

// ErrClosed is the error of an operation on a closed file.
var ErrClosed = errors.New("file already closed")

// PathError names the operation, the path, and the cause of a failure.
type PathError struct {
	Op   string
	Path string
	Err  error
}

func (e *PathError) Error() string { return e.Op + " " + e.Path + ": " + e.Err.Error() }

func (e *PathError) Unwrap() error { return e.Err }

// pathError builds the error of an operation on a path. It gives nil when the
// cause is empty.
func pathError(op, path, cause string) error {
	if cause == "" {
		return nil
	}
	return &PathError{Op: op, Path: path, Err: causeError(cause)}
}

// causeError maps the text of the Dart side to an error value. A text that
// names a missing path gives ErrNotExist, thus errors.Is finds it.
func causeError(cause string) error {
	if contains(cause, "No such file or directory") ||
		contains(cause, "no such file or directory") ||
		contains(cause, "cannot find the file") {
		return ErrNotExist
	}
	if contains(cause, "file already closed") {
		return ErrClosed
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
func IsNotExist(err error) bool { return errors.Is(err, ErrNotExist) }

// File is an open file. The zero value is not usable.
type File struct {
	handle int
	name   string
	// std marks standard output or standard error. Such a file holds no handle.
	std int
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
	Stdin  = &File{std: stdInFile, name: "/dev/stdin"}
	Stdout = &File{std: stdOut, name: "/dev/stdout"}
	Stderr = &File{std: stdErr, name: "/dev/stderr"}
)

func (f *File) Name() string { return f.name }

// Write writes the bytes and gives the count.
func (f *File) Write(p []byte) (int, error) {
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
	n, cause := dart.FileWrite(f.handle, p)
	return n, pathError("write", f.name, cause)
}

// WriteString writes the text and gives the count of bytes.
func (f *File) WriteString(s string) (int, error) { return f.Write([]byte(s)) }

// Read reads bytes into the slice and gives the count. It gives io.EOF at the
// end of the file, and the caller compares the error with that value.
func (f *File) Read(p []byte) (int, error) {
	if f.std == stdInFile {
		n, cause := dart.StdinRead(p)
		if cause != "" {
			return n, pathError("read", f.name, cause)
		}
		if n == 0 {
			return 0, endOfFile
		}
		return n, nil
	}
	if f.std != stdNone {
		return 0, &PathError{Op: "read", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	n, cause := dart.FileRead(f.handle, p)
	if cause != "" {
		return n, pathError("read", f.name, cause)
	}
	if n == 0 && len(p) > 0 {
		return 0, endOfFile
	}
	return n, nil
}

// ReadAt reads at the offset. It does not move the read position of the file,
// thus the call reads the position first and writes it again.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if f.std != stdNone {
		return 0, &PathError{Op: "read", Path: f.name, Err: errors.New("bad file descriptor")}
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
			err = endOfFile
			break
		}
		read += n
	}
	if _, cause := dart.FileSeek(f.handle, pos, dart.SeekStart); cause != "" && err == nil {
		err = pathError("readat", f.name, cause)
	}
	return read, err
}

// endOfFile is the error of the end of a file. The package io holds the same
// value under the name EOF.
var endOfFile = errors.New("EOF")

// EOF is the error of the end of a file.
var EOF = endOfFile

// Seek moves the read position and gives the new offset from the start.
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.std != stdNone {
		return 0, &PathError{Op: "seek", Path: f.name, Err: errors.New("bad file descriptor")}
	}
	pos, cause := dart.FileSeek(f.handle, offset, whence)
	return pos, pathError("seek", f.name, cause)
}

func (f *File) Close() error {
	if f.std != stdNone {
		return nil
	}
	return pathError("close", f.name, dart.FileClose(f.handle))
}

// Sync writes each held byte to the storage.
func (f *File) Sync() error {
	if f.std != stdNone {
		return nil
	}
	return pathError("sync", f.name, dart.FileSync(f.handle))
}

// Stat reads the metadata of the file.
func (f *File) Stat() (FileInfo, error) { return Stat(f.name) }

// Open opens a file for a read.
//
// Go also opens a directory this way. The Dart side gives no handle of a
// directory, thus such a call fails. Read a directory with ReadNames.
func Open(name string) (*File, error) {
	return openMode(name, dart.OpenRead, "open")
}

// Create makes a file, or it replaces the content of an existing file.
func Create(name string) (*File, error) {
	return openMode(name, dart.OpenWrite, "open")
}

// OpenFile opens a file with a flag set. The permission argument reaches no
// operation of the Dart side.
//
// The flag set selects 1 of the 3 modes of the Dart side: a read, a write that
// replaces the content, or a write at the end.
func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	mode := dart.OpenRead
	switch {
	case flag&O_APPEND != 0:
		mode = dart.OpenAppend
	case flag&(O_WRONLY|O_RDWR) != 0:
		mode = dart.OpenWrite
	}
	return openMode(name, mode, "open")
}

func openMode(name string, mode int, op string) (*File, error) {
	handle, cause := dart.FileOpen(name, mode)
	if cause != "" {
		return nil, pathError(op, name, cause)
	}
	return &File{handle: handle, name: name}, nil
}

// ReadFile reads a whole file.
func ReadFile(name string) ([]byte, error) {
	data, cause := dart.ReadFile(name)
	return data, pathError("open", name, cause)
}

// WriteFile writes a whole file. It replaces the content of an existing file.
// The permission argument reaches no operation of the Dart side.
func WriteFile(name string, data []byte, perm FileMode) error {
	return pathError("open", name, dart.WriteFile(name, data))
}

// FileInfo holds the metadata of a path.
type FileInfo struct {
	name  string
	size  int64
	dir   bool
	nanos int64
}

func (i FileInfo) Name() string { return i.name }

func (i FileInfo) Size() int64 { return i.size }

func (i FileInfo) IsDir() bool { return i.dir }

// ModNanos gives the time of the last change, in nanoseconds since the start of
// the Unix epoch.
//
// Go gives a time.Time from a method named ModTime. This package holds no
// import of time, because time would then import os.
func (i FileInfo) ModNanos() int64 { return i.nanos }

// Stat reads the metadata of a path.
func Stat(name string) (FileInfo, error) {
	size, dir, nanos, cause := dart.Stat(name)
	if cause != "" {
		return FileInfo{}, pathError("stat", name, cause)
	}
	return FileInfo{name: base(name), size: size, dir: dir, nanos: nanos}, nil
}

// base gives the last part of a path.
func base(path string) string {
	sep := dart.PathSeparator()
	if sep == "" || path == "" {
		return path
	}
	c := sep[0]
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == c {
			return path[i+1:]
		}
	}
	return path
}

func Remove(name string) error {
	return pathError("remove", name, dart.Remove(name))
}

// RemoveAll removes a path and each path inside it. It gives no error when the
// path does not exist.
func RemoveAll(name string) error {
	return pathError("removeall", name, dart.RemoveAll(name))
}

// Mkdir makes 1 directory. The permission argument reaches no operation of the
// Dart side.
func Mkdir(name string, perm FileMode) error {
	return pathError("mkdir", name, dart.Mkdir(name, false))
}

// MkdirAll makes a directory and each parent directory of it.
func MkdirAll(name string, perm FileMode) error {
	return pathError("mkdir", name, dart.Mkdir(name, true))
}

func Rename(from, to string) error {
	return pathError("rename", from, dart.Rename(from, to))
}

// ReadNames reads the names inside a directory. Each name holds no directory
// part, and the list is in the order of the name.
//
// Go holds ReadDir with a slice of DirEntry. A DirEntry holds 4 methods, and 1
// of them reads the metadata again. This function replaces it, thus a source
// that calls it does not compile with the go command.
func ReadNames(name string) ([]string, error) {
	names, cause := dart.ReadDir(name)
	return names, pathError("open", name, cause)
}

func Getenv(name string) string {
	value, _ := dart.Getenv(name)
	return value
}

// LookupEnv reads 1 variable of the environment. The flag is false when the
// environment holds no such name.
func LookupEnv(name string) (string, bool) { return dart.Getenv(name) }

// Environ reads each variable of the environment, in the form name=value.
func Environ() []string { return dart.Environ() }

func Getwd() (string, error) {
	path, cause := dart.Getwd()
	if cause != "" {
		return "", errors.New(cause)
	}
	return path, nil
}

func Chdir(path string) error {
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
	for i := 0; i < len(pattern); i++ {
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
	for i := 0; i < 100; i++ {
		name := tempName(dir, pattern)
		if _, err := Stat(name); err == nil {
			continue
		}
		if err := Mkdir(name, 0o700); err != nil {
			return "", err
		}
		return name, nil
	}
	return "", pathError("mkdirtemp", dir, "no free name")
}

// Exit stops the program with a status.
func Exit(code int) { dart.Exit(code) }
