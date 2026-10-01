// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Getrusage(who int, rusage *Rusage) error {
	data, code := dart.SyscallGetrusage(who, rusage != nil)
	if code != 0 {
		return Errno(code)
	}
	if rusage != nil {
		*rusage = Rusage{
			Utime:  setTimeval(data[0], data[1]),
			Stime:  setTimeval(data[2], data[3]),
			Maxrss: data[4], Ixrss: data[5], Idrss: data[6], Isrss: data[7],
			Minflt: data[8], Majflt: data[9], Nswap: data[10],
			Inblock: data[11], Oublock: data[12], Msgsnd: data[13], Msgrcv: data[14],
			Nsignals: data[15], Nvcsw: data[16], Nivcsw: data[17],
		}
	}
	return nil
}
