// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package syscall

import "internal/bytealg"

// StringByteSlice converts a string to a NUL-terminated []byte,
// If s contains a NUL byte this function panics instead of
// returning an error.
//
// Deprecated: Use ByteSliceFromString instead.
func StringByteSlice(s string) []byte {
	a, err := ByteSliceFromString(s)
	if err != nil {
		panic("syscall: string with NUL passed to StringByteSlice")
	}
	return a
}

// ByteSliceFromString returns a NUL-terminated slice of bytes
// containing the text of s. If s contains a NUL byte at any
// location, it returns (nil, [EINVAL]).
func ByteSliceFromString(s string) ([]byte, error) {
	if bytealg.IndexByteString(s, 0) != -1 {
		return nil, EINVAL
	}
	a := make([]byte, len(s)+1)
	copy(a, s)
	return a, nil
}

// StringBytePtr returns a pointer to a NUL-terminated array of bytes.
// If s contains a NUL byte this function panics instead of returning
// an error.
//
// Deprecated: Use [BytePtrFromString] instead.
func StringBytePtr(s string) *byte { return &StringByteSlice(s)[0] }

// BytePtrFromString returns a pointer to a NUL-terminated array of
// bytes containing the text of s. If s contains a NUL byte at any
// location, it returns (nil, [EINVAL]).
func BytePtrFromString(s string) (*byte, error) {
	a, err := ByteSliceFromString(s)
	if err != nil {
		return nil, err
	}
	return &a[0], nil
}

// StringSlicePtr converts a slice of strings to a slice of pointers
// to NUL-terminated byte arrays. If any string contains a NUL byte
// this function panics instead of returning an error.
//
// Deprecated: Use [SlicePtrFromStrings] instead.
func StringSlicePtr(ss []string) []*byte {
	bb := make([]*byte, len(ss)+1)
	for i := 0; i < len(ss); i++ {
		bb[i] = StringBytePtr(ss[i])
	}
	bb[len(ss)] = nil
	return bb
}

// SlicePtrFromStrings converts a slice of strings to a slice of
// pointers to NUL-terminated byte arrays. If any string contains
// a NUL byte, it returns (nil, [EINVAL]).
func SlicePtrFromStrings(ss []string) ([]*byte, error) {
	n := 0
	for _, s := range ss {
		if bytealg.IndexByteString(s, 0) != -1 {
			return nil, EINVAL
		}
		n += len(s) + 1 // +1 for NUL
	}
	bb := make([]*byte, len(ss)+1)
	b := make([]byte, n)
	n = 0
	for i, s := range ss {
		bb[i] = &b[n]
		copy(b[n:], s)
		n += len(s) + 1
	}
	return bb, nil
}
