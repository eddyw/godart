//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Getwd() (string, error) {
	path, code := dart.SyscallGetwd()
	if code != 0 {
		return "", syscallErrno(code)
	}
	return path, nil
}
