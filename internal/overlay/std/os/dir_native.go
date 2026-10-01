// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package os

import (
	"errors"
	"internal/byteorder"
	"io"
	"runtime"
	"syscall"
	"syscall/dart"
	"unsafe"
)

var errDirectoryClosed = errors.New("use of closed file")

func (f *File) readDirNames(n int) ([]string, []FileMode, error) {
	defer runtime.KeepAlive(f)
	names := make([]string, 0)
	types := make([]FileMode, 0)
	if f.closed {
		op := "readdirent"
		if dart.OS() == "darwin" {
			op = ""
		}
		return names, types, &PathError{Op: op, Path: f.name, Err: errDirectoryClosed}
	}
	remaining := n
	if remaining <= 0 {
		remaining = -1
	}
	for remaining != 0 {
		if f.dirBufPos >= len(f.dirBuf) {
			f.dirBuf = make([]byte, 8192)
			f.dirBufPos = 0
			count, err := syscall.ReadDirent(int(f.Fd()), f.dirBuf)
			for err == syscall.EINTR {
				count, err = syscall.ReadDirent(int(f.Fd()), f.dirBuf)
			}
			if err != nil {
				f.dirBuf = nil
				op := "readdirent"
				if dart.OS() == "darwin" && err == syscall.ENOTDIR {
					op = "fdopendir"
				}
				return names, types, &PathError{Op: op, Path: f.name, Err: err}
			}
			f.dirBuf = f.dirBuf[:count]
			if count == 0 {
				break
			}
		}
		record := f.dirBuf[f.dirBufPos:]
		const lengthOffset = unsafe.Offsetof(syscall.Dirent{}.Reclen)
		const typeOffset = unsafe.Offsetof(syscall.Dirent{}.Type)
		if len(record) < int(lengthOffset)+2 {
			break
		}
		length := int(byteorder.LEUint16(record[lengthOffset:]))
		if length <= int(typeOffset) || length > len(record) {
			break
		}
		record = record[:length]
		f.dirBufPos += length
		_, _, parsed := syscall.ParseDirent(record, 1, nil)
		if len(parsed) == 0 {
			continue
		}
		names = append(names, parsed[0])
		types = append(types, directoryEntryType(record[typeOffset]))
		if remaining > 0 {
			remaining--
		}
	}
	if n > 0 && len(names) == 0 {
		return names, types, io.EOF
	}
	return names, types, nil
}

func directoryEntryType(kind byte) FileMode {
	switch kind {
	case syscall.DT_REG:
		return 0
	case syscall.DT_DIR:
		return ModeDir
	case syscall.DT_LNK:
		return ModeSymlink
	case syscall.DT_BLK:
		return ModeDevice
	case syscall.DT_CHR:
		return ModeDevice | ModeCharDevice
	case syscall.DT_FIFO:
		return ModeNamedPipe
	case syscall.DT_SOCK:
		return ModeSocket
	}
	return ^FileMode(0)
}

func (f *File) directoryInfo(name string) (FileInfo, error) {
	defer runtime.KeepAlive(f)
	data, code := dart.NativeLstatat(f.dirFD, name)
	if code != 0 {
		return nil, &PathError{Op: "fstatat", Path: f.name + "/" + name, Err: syscall.Errno(code)}
	}
	stat := directoryStatFromData(data)
	return nativeFileInfo(name, &stat), nil
}

func openDirectory(name string) (*File, error) {
	return OpenFile(name, O_RDONLY|syscall.O_DIRECTORY, 0)
}
