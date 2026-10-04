// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && arm64

package syscall

const epollEventSize = 16
const epollDataOffset = 8
