//go:build dart

package syscall

import (
	errorspkg "errors"
	"syscall/dart"
)

func Getpid() int { return dart.Getpid() }

func Getppid() int { return dart.Getppid() }

func Getpgrp() int { return dart.Getpgrp() }

func Getpgid(pid int) (int, error) {
	group, code := dart.Getpgid(pid)
	if code == -1 {
		return group, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return group, Errno(code)
	}
	return group, nil
}

func Getuid() int { return dart.Getuid() }

func Geteuid() int { return dart.Geteuid() }

func Getgid() int { return dart.Getgid() }

func Getegid() int { return dart.Getegid() }

func Getpagesize() int { return dart.Getpagesize() }

func Getgroups() ([]int, error) {
	groups, code := dart.Getgroups()
	if code == -1 {
		return nil, errorspkg.ErrUnsupported
	}
	if code != 0 {
		return nil, Errno(code)
	}
	return groups, nil
}

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
