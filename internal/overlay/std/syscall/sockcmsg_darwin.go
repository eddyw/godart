// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

// Darwin uses 4-byte control-message alignment on both supported 64-bit ABIs.
const cmsgAlignment = 4

func (header *Cmsghdr) SetLen(length int) { header.Len = uint32(length) }
