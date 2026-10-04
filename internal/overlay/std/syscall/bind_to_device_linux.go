// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

func BindToDevice(fd int, device string) error {
	return SetsockoptString(fd, SOL_SOCKET, SO_BINDTODEVICE, device)
}
