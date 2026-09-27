//go:build dart

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import "context"

var testHookLookupIP = func(ctx context.Context, fn func(context.Context, string, string) ([]IPAddr, error), network, host string) ([]IPAddr, error) {
	return fn(ctx, network, host)
}
