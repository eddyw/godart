// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (amd64 || arm64)

package syscall

func (msghdr *Msghdr) SetControllen(length int) {
	msghdr.Controllen = uint32(length)
}
