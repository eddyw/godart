//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getpid() int { return dart.Getpid() }

func Kill(pid int, sig Signal) error {
	code := dart.SendSignal(pid, int(sig))
	if code < 0 {
		return errorspkg.ErrUnsupported
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}
