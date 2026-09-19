//go:build dart

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testenv

import (
	"os"
	"testing"
)

func SetGODEBUG(t testing.TB, v string) {
	t.Helper()
	t.Setenv("GODEBUG", os.Getenv("GODEBUG")+","+v)
}
