// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import "syscall/dart"

// RouteRIB returns routing information base, as known as RIB,
// which consists of network facility information, states and
// parameters.
//
// Deprecated: Use golang.org/x/net/route instead.
func RouteRIB(facility, param int) ([]byte, error) {
	tab, code := dart.SyscallRouteRIB(int32(facility), int32(param))
	if code != 0 {
		return nil, syscallErrno(code)
	}
	return tab, nil
}
