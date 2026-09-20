//go:build dart

package os

import (
	"errors"
	"io/fs"
	"syscall"
	"syscall/dart"
)

// The 4 error values of Go that the overlay did not hold. Each one comes from
// the package io/fs, thus errors.Is finds it.
var (
	// ErrInvalid names an argument that no operation accepts.
	ErrInvalid = fs.ErrInvalid
	// ErrPermission names an operation that the permission of the path stops.
	ErrPermission = fs.ErrPermission
	// ErrExist names a path that already exists.
	ErrExist = fs.ErrExist
	// ErrDeadlineExceeded names a read or a write that ran out of time.
	ErrDeadlineExceeded = errors.New("i/o timeout")
	// ErrNoDeadline names a file that takes no deadline.
	ErrNoDeadline = errors.New("file type does not support deadline")
	// ErrProcessDone names a process that already ended.
	ErrProcessDone = errors.New("os: process already finished")
	// ErrNoHandle names a Process that holds no handle of the system.
	ErrNoHandle = errors.New("os: process has no handle")
)

// IsExist reports whether the error names a path that already exists.
func IsExist(err error) bool { return underlyingErrorIs(err, ErrExist) }

// IsPermission reports whether the error names a permission that stops the
// operation.
func IsPermission(err error) bool { return underlyingErrorIs(err, ErrPermission) }

// IsTimeout reports whether the error names an operation that ran out of time.
func IsTimeout(err error) bool {
	t, ok := underlyingError(err).(interface{ Timeout() bool })
	return ok && t.Timeout()
}

// PathSeparator and PathListSeparator name the 2 separators of the platform.
// The Dart side names the platform at run time, thus the 2 hold the values of
// a Unix system. docs/parity.md holds that gap.
const (
	PathSeparator     = '/'
	PathListSeparator = ':'
)

// IsPathSeparator reports whether c separates 2 parts of a path.
func IsPathSeparator(c uint8) bool { return c == '/' }

// DevNull names the file that drops each write.
const DevNull = "/dev/null"

// The 3 modes of Seek. Go marks each one as deprecated, and io holds the same
// values under another name.
const (
	SEEK_SET = 0
	SEEK_CUR = 1
	SEEK_END = 2
)

// O_SYNC asks for a write that reaches the storage before it gives control
// back. The Dart side holds no such flag, thus a write drops it.
const O_SYNC = 1 << 12

// Clearenv drops each variable of the environment of the process.
func Clearenv() { syscall.Clearenv() }

// Expand replaces each `${name}` and each `$name` of s with the value that
// mapping gives.
func Expand(s string, mapping func(string) string) string {
	out := make([]byte, 0, 2*len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 >= len(s) {
			out = append(out, s[i])
			continue
		}
		name, width := shellName(s[i+1:])
		if name == "" && width > 0 {
			// A bad form such as `${` drops its characters, exactly as Go.
		} else if name == "" {
			out = append(out, s[i])
			continue
		} else {
			out = append(out, mapping(name)...)
		}
		i += width
	}
	return string(out)
}

// ExpandEnv replaces each `${name}` and each `$name` of s with the value of the
// variable of that name.
func ExpandEnv(s string) string { return Expand(s, Getenv) }

// shellName reads the name of a variable at the start of s, and it gives the
// count of the bytes that the name takes.
func shellName(s string) (string, int) {
	if len(s) == 0 {
		return "", 0
	}
	if s[0] == '{' {
		for i := 1; i < len(s); i++ {
			if s[i] == '}' {
				if i == 1 {
					return "", 2
				}
				return s[1:i], i + 1
			}
		}
		return "", 1
	}
	if isShellSpecial(s[0]) {
		return s[0:1], 1
	}
	i := 0
	for i < len(s) && isNameByte(s[i]) {
		i++
	}
	return s[:i], i
}

// isShellSpecial reports a name of 1 character that a shell holds, for example
// `$1` or `$*`.
func isShellSpecial(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6',
		'7', '8', '9':
		return true
	}
	return false
}

func isNameByte(c uint8) bool {
	return c == '_' || ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') ||
		('A' <= c && c <= 'Z')
}

// Getpid gives the process id of the program.
func Getpid() int { return dart.Getpid() }

// Getppid gives the process id of the parent. The Dart side names no parent,
// thus the result is 0. docs/parity.md holds that gap.
func Getppid() int { return 0 }

// The 5 identity functions below give -1. The Dart side names no user and no
// group of the process, and Go also gives -1 on a system that holds none.
func Getuid() int  { return -1 }
func Geteuid() int { return -1 }
func Getgid() int  { return -1 }
func Getegid() int { return -1 }

// Getgroups gives each group of the process. The Dart side names none.
func Getgroups() ([]int, error) { return nil, nil }

// Getpagesize gives the count of bytes of 1 page of memory. The Dart side names
// no page, thus the result is the common value of 4096.
func Getpagesize() int { return 4096 }

// Hostname gives the name of the host of the program.
func Hostname() (string, error) {
	name, cause := dart.Hostname()
	if cause != "" {
		return "", errors.New(cause)
	}
	return name, nil
}

// Executable gives the path of the program that runs.
//
// The Dart VM runs the emitted Dart, thus the path names the VM. An ahead of
// time build names the program itself. docs/parity.md holds that gap.
func Executable() (string, error) {
	path, cause := dart.Executable()
	if cause != "" {
		return "", errors.New(cause)
	}
	return path, nil
}

// UserHomeDir gives the home directory of the user.
func UserHomeDir() (string, error) {
	if home := Getenv("HOME"); home != "" {
		return home, nil
	}
	return "", errors.New("$HOME is not defined")
}

// UserConfigDir gives the directory of the settings of the user.
func UserConfigDir() (string, error) {
	if dir := Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	home, err := UserHomeDir()
	if err != nil {
		return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
	}
	return home + "/.config", nil
}

// UserCacheDir gives the directory of the cache of the user.
func UserCacheDir() (string, error) {
	if dir := Getenv("XDG_CACHE_HOME"); dir != "" {
		return dir, nil
	}
	home, err := UserHomeDir()
	if err != nil {
		return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
	}
	return home + "/.cache", nil
}

// A SyscallError names the operation of a call of the system and its cause.
type SyscallError struct {
	Syscall string
	Err     error
}

func (e *SyscallError) Error() string { return e.Syscall + ": " + e.Err.Error() }

func (e *SyscallError) Unwrap() error { return e.Err }

// Timeout reports whether the cause names an operation that ran out of time.
func (e *SyscallError) Timeout() bool {
	t, ok := e.Err.(interface{ Timeout() bool })
	return ok && t.Timeout()
}

// NewSyscallError gives a SyscallError, and it gives nil for a nil cause.
func NewSyscallError(syscall string, err error) error {
	if err == nil {
		return nil
	}
	return &SyscallError{Syscall: syscall, Err: err}
}

// SameFile reports whether 2 descriptions name 1 file. The Dart side names no
// inode, thus the test reads the size, the mode, and the time of the last
// change. docs/parity.md holds that gap.
func SameFile(fi1, fi2 FileInfo) bool {
	if fi1 == nil || fi2 == nil {
		return false
	}
	return fi1.Size() == fi2.Size() && fi1.Mode() == fi2.Mode() &&
		fi1.ModTime().Equal(fi2.ModTime())
}

// Signal names an operating system signal. The concrete type is
// syscall.Signal, as it is in Go.
type Signal interface {
	String() string
	Signal()
}

// The 2 signals that each platform holds.
var (
	Interrupt Signal = syscall.SIGINT
	Kill      Signal = syscall.SIGKILL
)
