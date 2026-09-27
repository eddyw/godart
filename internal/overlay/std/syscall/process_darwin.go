//go:build dart && darwin

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getsid(pid int) (int, error) {
	session, code := dart.Getsid(pid)
	if code == -1 {
		return session, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return session, Errno(code)
	}
	return session, nil
}
