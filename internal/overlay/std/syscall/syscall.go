//go:build dart

// Package syscall gives the signal type and the signal numbers that the
// surface holds. The package of Go holds the whole system call interface, and
// the surface holds no such interface: a Go program reaches the system through
// os, through net, and through the interop package syscall/dart.
//
// The numbers below are the numbers that macOS and Linux agree on. A signal
// whose number differs between the 2, for example SIGUSR1, stays out. Read
// docs/parity.md.
package syscall

import "strconv"

// Signal names an operating system signal.
type Signal int

// Signal marks the type as a signal of the package os.
func (s Signal) Signal() {}

func (s Signal) String() string {
	if name := signalName(s); name != "" {
		return name
	}
	return "signal " + strconv.Itoa(int(s))
}

// The signals that macOS and Linux number the same way.
const (
	SIGHUP  Signal = 1
	SIGINT  Signal = 2
	SIGQUIT Signal = 3
	SIGILL  Signal = 4
	SIGTRAP Signal = 5
	SIGABRT Signal = 6
	SIGFPE  Signal = 8
	SIGKILL Signal = 9
	SIGSEGV Signal = 11
	SIGPIPE Signal = 13
	SIGALRM Signal = 14
	SIGTERM Signal = 15
)

func signalName(s Signal) string {
	switch s {
	case SIGHUP:
		return "hangup"
	case SIGINT:
		return "interrupt"
	case SIGQUIT:
		return "quit"
	case SIGILL:
		return "illegal instruction"
	case SIGTRAP:
		return "trace/BPT trap"
	case SIGABRT:
		return "abort trap"
	case SIGFPE:
		return "floating point exception"
	case SIGKILL:
		return "killed"
	case SIGSEGV:
		return "segmentation fault"
	case SIGPIPE:
		return "broken pipe"
	case SIGALRM:
		return "alarm clock"
	case SIGTERM:
		return "terminated"
	}
	return ""
}
