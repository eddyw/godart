//go:build dart

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sys

import "math/bits"

func TrailingZeros32(x uint32) int { return bits.TrailingZeros32(x) }
func TrailingZeros64(x uint64) int { return bits.TrailingZeros64(x) }
func Bswap32(x uint32) uint32      { return bits.ReverseBytes32(x) }
func Bswap64(x uint64) uint64      { return bits.ReverseBytes64(x) }
