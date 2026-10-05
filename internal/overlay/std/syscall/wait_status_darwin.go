// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (amd64 || arm64)

package syscall

func (w WaitStatus) Stopped() bool {
	return w&waitMask == waitStopped && Signal(w>>waitShift) != SIGSTOP
}

// The BSD Go SDK uses SIGSTOP to mark a continued status word.
func (w WaitStatus) Continued() bool {
	return w&waitMask == waitStopped && Signal(w>>waitShift) == SIGSTOP
}

func (w WaitStatus) ExitStatus() int {
	if w&waitMask != waitExited {
		return -1
	}
	return int(w >> waitShift)
}

func (w WaitStatus) TrapCause() int { return -1 }
