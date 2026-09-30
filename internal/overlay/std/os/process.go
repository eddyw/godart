//go:build dart

package os

import (
	"errors"
	"strconv"
	"syscall"
	"syscall/dart"
)

// Process identifies a process by its id.
type Process struct {
	Pid  int
	done bool
}

// FindProcess gives a Process for a process id. Wait requires a child process.
func FindProcess(pid int) (*Process, error) {
	return &Process{Pid: pid}, nil
}

// Kill ends the process with SIGKILL.
func (p *Process) Kill() error {
	return p.Signal(Kill)
}

// processError maps the text of the Dart side to an error value. A process that
// already ended gives ErrProcessDone, thus errors.Is finds it.
func processError(cause string) error {
	if cause == ErrProcessDone.Error() {
		return ErrProcessDone
	}
	return errors.New(cause)
}

// Signal sends a signal to the process. A signal that the platform cannot
// send gives an error.
func (p *Process) Signal(sig Signal) error {
	number, ok := sig.(syscall.Signal)
	if !ok {
		return errors.New("os: unsupported signal type")
	}
	if p.Pid == 0 {
		return errors.New("os: process not initialized")
	}
	if p.Pid == -1 {
		return errors.New("os: process already released")
	}
	if p.done {
		return ErrProcessDone
	}
	if number == 0 {
		err := syscall.Kill(p.Pid, number)
		if err == syscall.ESRCH {
			return ErrProcessDone
		}
		return err
	}
	if cause := dart.KillPid(p.Pid, int(number)); cause != "" {
		return processError(cause)
	}
	return nil
}

// Wait waits for the end of the process and gives its state.
func (p *Process) Wait() (*ProcessState, error) {
	if p.Pid == -1 {
		return nil, syscall.EINVAL
	}
	if p.done {
		return nil, NewSyscallError("wait", syscall.ECHILD)
	}
	code, consumed, cause := dart.ProcWait(p.Pid)
	if cause != "" {
		return nil, errors.New(cause)
	}
	if consumed {
		return nil, NewSyscallError("wait", syscall.ECHILD)
	}
	p.done = true
	return &ProcessState{pid: p.Pid, code: code}, nil
}

// Release invalidates this Process. The runtime owns the child and its IO.
func (p *Process) Release() error {
	p.Pid = -1
	return nil
}

// WithHandle runs f with a handle of the process.
//
// dart:io names a child by its process id and gives no handle of the system.
// Thus the call runs f no time and gives ErrNoHandle, exactly as Go does on a
// platform that holds no such handle.
func (p *Process) WithHandle(f func(handle uintptr)) error { return ErrNoHandle }

// ProcessState holds the exit state of a process. A negative code names the
// signal that ended the process, as the Dart side gives it.
type ProcessState struct {
	pid  int
	code int
}

func (p *ProcessState) Pid() int { return p.pid }

// ExitCode gives the exit code, or -1 when a signal ended the process.
func (p *ProcessState) ExitCode() int {
	if p.code < 0 {
		return -1
	}
	return p.code
}

func (p *ProcessState) Exited() bool { return p.code >= 0 }

func (p *ProcessState) Success() bool { return p.code == 0 }

// Sys gives the exit state of the system. The value holds the type
// syscall.WaitStatus, exactly as it does on a Unix system of Go.
//
// dart:io reports the exit code and the number of the signal that ended the
// child. The word below carries the 2, thus Exited, ExitStatus, Signaled, and
// Signal each give the answer of Go. Read docs/parity.md for the 2 parts of
// the word that hold no source.
func (p *ProcessState) Sys() any { return p.status() }

// status gives the word of the wait state of the process.
func (p *ProcessState) status() syscall.WaitStatus {
	if p.code < 0 {
		return syscall.WaitStatus(-p.code) & 0x7F
	}
	return syscall.WaitStatus(p.code) << 8
}

func (p *ProcessState) String() string {
	if p == nil {
		return "<nil>"
	}
	if p.code < 0 {
		return "signal: " + syscall.Signal(-p.code).String()
	}
	return "exit status " + strconv.Itoa(p.code)
}
