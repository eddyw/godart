// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && ((!netgo && ((cgo && unix) || darwin)) || windows || plan9)

package net

const cgoAvailable = true
