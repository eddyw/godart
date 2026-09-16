// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package poly1305

// The Dart target uses the portable MAC in place of the assembly wrapper.
type mac struct{ macGeneric }
