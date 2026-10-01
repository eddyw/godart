// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux

package syscall

func Time(t *Time_t) (Time_t, error) {
	var tv Timeval
	err := Gettimeofday(&tv)
	if err != nil {
		return 0, err
	}
	if t != nil {
		*t = Time_t(tv.Sec)
	}
	return Time_t(tv.Sec), nil
}
