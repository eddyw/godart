//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Access(path string, mode uint32) error {
	if errno := dart.SyscallAccess(path, mode); errno != 0 {
		return syscallErrno(errno)
	}
	return nil
}
