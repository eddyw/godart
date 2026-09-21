// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package os

import (
	"syscall"
	"syscall/dart"
)

func nativeFileInfo(name, op string, size int64, mode uint32, seconds, nanos int64, device, inode uint64, errno int) (FileInfo, error) {
	if errno != 0 {
		return nil, &PathError{Op: op, Path: name, Err: syscall.Errno(errno)}
	}
	m := FileMode(mode & 0777)
	switch mode & 0170000 {
	case 0060000, 0160000:
		m |= ModeDevice
	case 0020000:
		m |= ModeDevice | ModeCharDevice
	case 0040000:
		m |= ModeDir
	case 0010000:
		m |= ModeNamedPipe
	case 0120000:
		m |= ModeSymlink
	case 0140000:
		m |= ModeSocket
	}
	if mode&sISGID != 0 {
		m |= ModeSetgid
	}
	if mode&sISUID != 0 {
		m |= ModeSetuid
	}
	if mode&sISVTX != 0 {
		m |= ModeSticky
	}
	return &fileStat{name: base(name), size: size, mode: m, seconds: seconds, nanos: nanos, device: device, inode: inode}, nil
}

// SameFile compares descriptions from this package. Supported POSIX hosts
// compare native device and inode values. Other hosts compare metadata.
func SameFile(fi1, fi2 FileInfo) bool {
	fs1, ok1 := fi1.(*fileStat)
	fs2, ok2 := fi2.(*fileStat)
	if !ok1 || !ok2 {
		return false
	}
	if dart.HasNativeStat() {
		return fs1.device == fs2.device && fs1.inode == fs2.inode
	}
	return fs1.size == fs2.size && fs1.mode == fs2.mode &&
		fs1.ModTime().Equal(fs2.ModTime())
}
