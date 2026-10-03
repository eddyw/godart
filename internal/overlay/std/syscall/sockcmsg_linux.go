// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

const cmsgAlignment = 8

func (header *Cmsghdr) SetLen(length int) { header.Len = uint64(length) }
