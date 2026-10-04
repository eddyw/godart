// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func EpollCreate(size int) (int, error) {
	if size <= 0 {
		return -1, EINVAL
	}
	return EpollCreate1(0)
}

func EpollCreate1(flag int) (int, error) {
	fd, code := dart.SyscallEpollCreate1(flag)
	if code != 0 {
		return fd, Errno(code)
	}
	return fd, nil
}

func putEpollEvent(data []byte, event EpollEvent) {
	binary.NativeEndian.PutUint32(data, event.Events)
	binary.NativeEndian.PutUint32(data[epollDataOffset:], uint32(event.Fd))
	binary.NativeEndian.PutUint32(data[epollDataOffset+4:], uint32(event.Pad))
}

func EpollCtl(epfd, op, fd int, event *EpollEvent) error {
	var data []byte
	if event != nil {
		data = make([]byte, epollEventSize)
		putEpollEvent(data, *event)
	}
	if code := dart.SyscallEpollCtl(epfd, op, fd, data); code != 0 {
		return Errno(code)
	}
	return nil
}

func EpollWait(epfd int, events []EpollEvent, msec int) (int, error) {
	data := make([]byte, len(events)*epollEventSize)
	for i, event := range events {
		putEpollEvent(data[i*epollEventSize:], event)
	}
	n, code := dart.SyscallEpollWait(epfd, data, len(events), msec)
	for i := range events {
		entry := data[i*epollEventSize:]
		events[i].Events = binary.NativeEndian.Uint32(entry)
		events[i].Fd = int32(binary.NativeEndian.Uint32(entry[epollDataOffset:]))
		events[i].Pad = int32(binary.NativeEndian.Uint32(entry[epollDataOffset+4:]))
	}
	if code != 0 {
		return n, Errno(code)
	}
	return n, nil
}
