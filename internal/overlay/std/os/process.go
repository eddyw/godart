//go:build dart

package os

import (
	"errors"
	"strconv"
	"syscall"
	"syscall/dart"
)

// Process is a child process that os/exec started.
type Process struct {
	Pid int
}

// FindProcess gives the process of a process id. The runtime names a child by
// its id, thus the result works only for a child of this program.
func FindProcess(pid int) (*Process, error) {
	return &Process{Pid: pid}, nil
}

// Kill ends the process with SIGKILL.
func (p *Process) Kill() error {
	if cause := dart.ProcKill(p.Pid, 9); cause != "" {
		return errors.New(cause)
	}
	return nil
}

// Signal sends a signal to the process. A signal that the platform cannot
// send gives an error.
func (p *Process) Signal(sig Signal) error {
	number, ok := sig.(syscall.Signal)
	if !ok {
		return errors.New("os: unsupported signal type")
	}
	if cause := dart.KillPid(p.Pid, int(number)); cause != "" {
		return errors.New(cause)
	}
	return nil
}

// Wait waits for the end of the process and gives its state.
func (p *Process) Wait() (*ProcessState, error) {
	code, cause := dart.ProcWait(p.Pid)
	if cause != "" {
		return nil, errors.New(cause)
	}
	return &ProcessState{pid: p.Pid, code: code}, nil
}

// Release drops the resources of the process. The runtime holds the child in
// a table by id, thus there is nothing to drop.
func (p *Process) Release() error { return nil }

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

func (p *ProcessState) String() string {
	if p.code < 0 {
		return "signal: " + signalName(-p.code)
	}
	return "exit status " + strconv.Itoa(p.code)
}

func signalName(sig int) string {
	switch sig {
	case 2:
		return "interrupt"
	case 9:
		return "killed"
	case 15:
		return "terminated"
	}
	return "signal " + strconv.Itoa(sig)
}
