// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && arm64

package syscall

func utimes(path string, tv []Timeval) error {
	return futimesat(-100, path, tv)
}

func futimesat(dirfd int, path string, tv []Timeval) error {
	ts := []Timespec{
		NsecToTimespec(TimevalToNsec(tv[0])),
		NsecToTimespec(TimevalToNsec(tv[1])),
	}
	return utimensat(dirfd, path, ts)
}
