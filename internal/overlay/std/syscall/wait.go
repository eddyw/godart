//go:build dart

// The exit state of a child process.
//
// os.ProcessState.Sys gives a value of this type, exactly as it does on a Unix
// system of Go. dart:io reports 2 numbers of a child that ended: the exit code,
// and the number of the signal that ended it. The overlay writes those 2 into
// the word of a Unix wait state, thus each read below gives the answer of Go.
//
// 2 parts of the word hold no source. dart:io reports no core dump, thus
// CoreDump is always false. dart:io reports no child that a signal stopped,
// thus Stopped is always false and StopSignal always gives -1. Read
// docs/parity.md.

package syscall

// The parts of the word of a wait state, as a Unix system writes them.
const (
	waitMask    = 0x7F
	waitCore    = 0x80
	waitExited  = 0x00
	waitStopped = 0x7F
	waitShift   = 8
)

// A WaitStatus holds the exit state of a child process.
type WaitStatus uint32

// Exited reports whether the child ended with a call of exit.
func (w WaitStatus) Exited() bool { return w&waitMask == waitExited }

// Signaled reports whether a signal ended the child.
func (w WaitStatus) Signaled() bool {
	return w&waitMask != waitStopped && w&waitMask != waitExited
}

// Stopped reports whether a signal stopped the child. dart:io reports no such
// child, thus the answer is always false.
func (w WaitStatus) Stopped() bool { return w&0xFF == waitStopped }

// Continued reports whether a signal started the child again. dart:io reports
// no such child, thus the answer is always false.
func (w WaitStatus) Continued() bool { return w == 0xFFFF }

// CoreDump reports whether the child wrote a core file. dart:io reports no such
// flag, thus the answer is always false.
func (w WaitStatus) CoreDump() bool { return w.Signaled() && w&waitCore != 0 }

// ExitStatus gives the code of the exit, and -1 for a child that a signal
// ended.
func (w WaitStatus) ExitStatus() int {
	if !w.Exited() {
		return -1
	}
	return int(w>>waitShift) & 0xFF
}

// Signal gives the signal that ended the child, and -1 for a child that ended
// with a call of exit.
func (w WaitStatus) Signal() Signal {
	sig := Signal(w & waitMask)
	if sig == waitStopped || sig == 0 {
		return -1
	}
	return sig
}

// StopSignal gives the signal that stopped the child. dart:io reports no such
// child, thus the answer is always -1.
func (w WaitStatus) StopSignal() Signal {
	if !w.Stopped() {
		return -1
	}
	return Signal(w>>waitShift) & 0xFF
}
