// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func Sysinfo(info *Sysinfo_t) error {
	var data []byte
	if info != nil {
		data = make([]byte, 112)
		binary.NativeEndian.PutUint64(data, uint64(info.Uptime))
		for i, value := range info.Loads {
			binary.NativeEndian.PutUint64(data[8+i*8:], value)
		}
		binary.NativeEndian.PutUint64(data[32:], info.Totalram)
		binary.NativeEndian.PutUint64(data[40:], info.Freeram)
		binary.NativeEndian.PutUint64(data[48:], info.Sharedram)
		binary.NativeEndian.PutUint64(data[56:], info.Bufferram)
		binary.NativeEndian.PutUint64(data[64:], info.Totalswap)
		binary.NativeEndian.PutUint64(data[72:], info.Freeswap)
		binary.NativeEndian.PutUint64(data[88:], info.Totalhigh)
		binary.NativeEndian.PutUint64(data[96:], info.Freehigh)
		binary.NativeEndian.PutUint16(data[80:], info.Procs)
		binary.NativeEndian.PutUint16(data[82:], info.Pad)
		copy(data[84:88], info.Pad_cgo_0[:])
		binary.NativeEndian.PutUint32(data[104:], info.Unit)
		copy(data[108:112], info.Pad_cgo_1[:])
	}
	code := dart.SyscallSysinfo(data)
	if info != nil {
		info.Uptime = int64(binary.NativeEndian.Uint64(data))
		for i := range info.Loads {
			info.Loads[i] = binary.NativeEndian.Uint64(data[8+i*8:])
		}
		info.Totalram = binary.NativeEndian.Uint64(data[32:])
		info.Freeram = binary.NativeEndian.Uint64(data[40:])
		info.Sharedram = binary.NativeEndian.Uint64(data[48:])
		info.Bufferram = binary.NativeEndian.Uint64(data[56:])
		info.Totalswap = binary.NativeEndian.Uint64(data[64:])
		info.Freeswap = binary.NativeEndian.Uint64(data[72:])
		info.Totalhigh = binary.NativeEndian.Uint64(data[88:])
		info.Freehigh = binary.NativeEndian.Uint64(data[96:])
		info.Procs = binary.NativeEndian.Uint16(data[80:])
		info.Pad = binary.NativeEndian.Uint16(data[82:])
		copy(info.Pad_cgo_0[:], data[84:88])
		info.Unit = binary.NativeEndian.Uint32(data[104:])
		copy(info.Pad_cgo_1[:], data[108:112])
	}
	if code != 0 {
		return Errno(code)
	}
	return nil
}
