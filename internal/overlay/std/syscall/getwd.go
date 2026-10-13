//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

const ImplementsGetwd = true

func Getwd() (string, error) {
	path, code := dart.SyscallGetwd()
	if code != 0 {
		return "", syscallErrno(code)
	}
	return path, nil
}
