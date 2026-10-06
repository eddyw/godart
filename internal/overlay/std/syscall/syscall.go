// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Package syscall provides system constants and supported native operations.
package syscall

import "strconv"

var (
	Stdin  = 0
	Stdout = 1
	Stderr = 2
)

// Signal identifies an operating system signal.
type Signal int

func (s Signal) Signal() {}

func (s Signal) String() string {
	if 0 <= int(s) && int(s) < len(signals) {
		if text := signals[s]; text != "" {
			return text
		}
	}
	return "signal " + strconv.Itoa(int(s))
}
