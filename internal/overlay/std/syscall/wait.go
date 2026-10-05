// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

// The parts of the word of a wait state, as a Unix system writes them.
const (
	waitMask    = 0x7F
	waitCore    = 0x80
	waitExited  = 0x00
	waitStopped = 0x7F
	waitShift   = 8
)

// WaitStatus holds a Unix status word.
// Dart process state supplies only exit and termination signal values.
// Native stop, continue, core dump and ptrace observations remain unsupported.
type WaitStatus uint32

// Exited reports whether the child ended with a call of exit.
func (w WaitStatus) Exited() bool { return w&waitMask == waitExited }

// Signaled reports whether a signal ended the child.
func (w WaitStatus) Signaled() bool {
	return w&waitMask != waitStopped && w&waitMask != waitExited
}

// CoreDump reports whether the status word holds a core dump flag.
func (w WaitStatus) CoreDump() bool { return w.Signaled() && w&waitCore != 0 }

// Signal gives the signal that ended the child, and -1 for a child that ended
// with a call of exit.
func (w WaitStatus) Signal() Signal {
	sig := Signal(w & waitMask)
	if sig == waitStopped || sig == 0 {
		return -1
	}
	return sig
}

// StopSignal gives the signal in a stopped status word.
func (w WaitStatus) StopSignal() Signal {
	if !w.Stopped() {
		return -1
	}
	return Signal(w>>waitShift) & 0xFF
}
