// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (amd64 || arm64)

package syscall

func (w WaitStatus) Stopped() bool { return w&0xFF == waitStopped }

func (w WaitStatus) Continued() bool { return w == 0xFFFF }

func (w WaitStatus) ExitStatus() int {
	if !w.Exited() {
		return -1
	}
	return int(w>>waitShift) & 0xFF
}

func (w WaitStatus) TrapCause() int {
	if w.StopSignal() != SIGTRAP {
		return -1
	}
	return int(w>>waitShift) >> 8
}
