// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func putKevent(data []byte, event Kevent_t) {
	binary.NativeEndian.PutUint64(data, event.Ident)
	binary.NativeEndian.PutUint16(data[8:], uint16(event.Filter))
	binary.NativeEndian.PutUint16(data[10:], event.Flags)
	binary.NativeEndian.PutUint32(data[12:], event.Fflags)
	binary.NativeEndian.PutUint64(data[16:], uint64(event.Data))
}

// A receipt can overwrite a later change when the caller slices overlap.
func keventBuffers(changes, events []Kevent_t) ([]byte, []byte) {
	changeStart, eventStart := -1, -1
	if len(changes) > 0 && len(events) > 0 {
		for i := range changes {
			if &changes[i] == &events[0] {
				changeStart, eventStart = 0, i
				break
			}
		}
		if changeStart < 0 {
			for i := range events {
				if &events[i] == &changes[0] {
					changeStart, eventStart = i, 0
					break
				}
			}
		}
	}
	if changeStart < 0 {
		return make([]byte, len(changes)*32), make([]byte, len(events)*32)
	}
	end := changeStart + len(changes)
	if eventStart+len(events) > end {
		end = eventStart + len(events)
	}
	data := make([]byte, end*32)
	return data[changeStart*32 : (changeStart+len(changes))*32], data[eventStart*32 : (eventStart+len(events))*32]
}

// Kevent requires nil Udata values and a queue whose registered events have nil Udata.
// Native calls can block the Dart scheduler.
func Kevent(kq int, changes, events []Kevent_t, timeout *Timespec) (int, error) {
	for _, event := range changes {
		if event.Udata != nil {
			return -1, ENOSYS
		}
	}
	for _, event := range events {
		if event.Udata != nil {
			return -1, ENOSYS
		}
	}
	change, event := keventBuffers(changes, events)
	for i, value := range changes {
		putKevent(change[i*32:], value)
	}
	for i, value := range events {
		putKevent(event[i*32:], value)
	}
	var limit []byte
	if timeout != nil {
		limit = make([]byte, 16)
		binary.NativeEndian.PutUint64(limit, uint64(timeout.Sec))
		binary.NativeEndian.PutUint64(limit[8:], uint64(timeout.Nsec))
	}
	n, code := dart.SyscallKevent(kq, change, event, limit)
	for i := range events {
		entry := event[i*32:]
		events[i].Ident = binary.NativeEndian.Uint64(entry)
		events[i].Filter = int16(binary.NativeEndian.Uint16(entry[8:]))
		events[i].Flags = binary.NativeEndian.Uint16(entry[10:])
		events[i].Fflags = binary.NativeEndian.Uint32(entry[12:])
		events[i].Data = int64(binary.NativeEndian.Uint64(entry[16:]))
	}
	if code != 0 {
		return n, syscallErrno(code)
	}
	return n, nil
}
