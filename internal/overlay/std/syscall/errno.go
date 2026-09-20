//go:build dart

package syscall

import (
	"internal/oserror"
	"strconv"
)

// Errno holds the number of a failure of the system, exactly as the package of
// Go does. A PathError of os holds such a value in its field Err.
//
// The table below holds each error that macOS and Linux number the same way
// and name with the same text. The 2 systems disagree about each other error:
// ENOTEMPTY is 66 on macOS and 39 on Linux, EAGAIN is 35 and 11, and the text
// of ERANGE is "result too large" and "numerical result out of range". A
// program that runs the Dart output reads no table of the system, thus such an
// error stays out and the os package keeps its own error value for it. Read
// docs/parity.md.
type Errno uintptr

func (e Errno) Error() string {
	if text := errorText(e); text != "" {
		return text
	}
	return "errno " + strconv.Itoa(int(e))
}

// Is gives the answer of the sentinel group of the standard library. The rule
// of Go also names ENOTEMPTY under ErrExist and ENOSYS, ENOTSUP, and
// EOPNOTSUPP under ErrUnsupported. The 4 stay out of the table above, thus no
// value of this type can hold them.
func (e Errno) Is(target error) bool {
	switch target {
	case oserror.ErrPermission:
		return e == EACCES || e == EPERM
	case oserror.ErrExist:
		return e == EEXIST
	case oserror.ErrNotExist:
		return e == ENOENT
	}
	return false
}

// The errors that macOS and Linux number the same way and name the same way.
const (
	EPERM   Errno = 1
	ENOENT  Errno = 2
	ESRCH   Errno = 3
	EINTR   Errno = 4
	EIO     Errno = 5
	E2BIG   Errno = 7
	ENOEXEC Errno = 8
	EBADF   Errno = 9
	ECHILD  Errno = 10
	ENOMEM  Errno = 12
	EACCES  Errno = 13
	EFAULT  Errno = 14
	ENOTBLK Errno = 15
	EEXIST  Errno = 17
	ENOTDIR Errno = 20
	EISDIR  Errno = 21
	EINVAL  Errno = 22
	ENFILE  Errno = 23
	EMFILE  Errno = 24
	ENOTTY  Errno = 25
	ETXTBSY Errno = 26
	EFBIG   Errno = 27
	ENOSPC  Errno = 28
	ESPIPE  Errno = 29
	EROFS   Errno = 30
	EMLINK  Errno = 31
	EPIPE   Errno = 32
	EDOM    Errno = 33
)

func errorText(e Errno) string {
	switch e {
	case EPERM:
		return "operation not permitted"
	case ENOENT:
		return "no such file or directory"
	case ESRCH:
		return "no such process"
	case EINTR:
		return "interrupted system call"
	case EIO:
		return "input/output error"
	case E2BIG:
		return "argument list too long"
	case ENOEXEC:
		return "exec format error"
	case EBADF:
		return "bad file descriptor"
	case ECHILD:
		return "no child processes"
	case ENOMEM:
		return "cannot allocate memory"
	case EACCES:
		return "permission denied"
	case EFAULT:
		return "bad address"
	case ENOTBLK:
		return "block device required"
	case EEXIST:
		return "file exists"
	case ENOTDIR:
		return "not a directory"
	case EISDIR:
		return "is a directory"
	case EINVAL:
		return "invalid argument"
	case ENFILE:
		return "too many open files in system"
	case EMFILE:
		return "too many open files"
	case ENOTTY:
		return "inappropriate ioctl for device"
	case ETXTBSY:
		return "text file busy"
	case EFBIG:
		return "file too large"
	case ENOSPC:
		return "no space left on device"
	case ESPIPE:
		return "illegal seek"
	case EROFS:
		return "read-only file system"
	case EMLINK:
		return "too many links"
	case EPIPE:
		return "broken pipe"
	case EDOM:
		return "numerical argument out of domain"
	}
	return ""
}
