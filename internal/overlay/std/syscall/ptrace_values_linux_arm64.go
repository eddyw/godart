// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && arm64

package syscall

func (r *PtraceRegs) PC() uint64 { return r.Pc }

func (r *PtraceRegs) SetPC(pc uint64) { r.Pc = pc }
