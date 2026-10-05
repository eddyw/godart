// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

import (
	"encoding/binary"
	"syscall/dart"
)

func bpfScalar(fd, request, value int) (int, error) {
	data := make([]byte, 8)
	binary.NativeEndian.PutUint64(data, uint64(value))
	if code := dart.SyscallBpfIoctl(fd, request, data, nil); code != 0 {
		return 0, syscallErrno(code)
	}
	return int(binary.NativeEndian.Uint64(data)), nil
}

func BpfBuflen(fd int) (int, error)         { return bpfScalar(fd, BIOCGBLEN, 0) }
func SetBpfBuflen(fd, l int) (int, error)   { return bpfScalar(fd, BIOCSBLEN, l) }
func BpfDatalink(fd int) (int, error)       { return bpfScalar(fd, BIOCGDLT, 0) }
func SetBpfDatalink(fd, t int) (int, error) { return bpfScalar(fd, BIOCSDLT, t) }
func SetBpfPromisc(fd, m int) error         { _, err := bpfScalar(fd, BIOCPROMISC, m); return err }
func FlushBpf(fd int) error {
	if code := dart.SyscallBpfIoctl(fd, BIOCFLUSH, nil, nil); code != 0 {
		return syscallErrno(code)
	}
	return nil
}

func BpfInterface(fd int, name string) (string, error) {
	// The ioctl uses a 32-byte ifreq. The original ivalue occupies 18 bytes.
	data := make([]byte, 32)
	if code := dart.SyscallBpfIoctl(fd, BIOCGETIF, data, nil); code != 0 {
		return "", syscallErrno(code)
	}
	return name, nil
}
func SetBpfInterface(fd int, name string) error {
	data := make([]byte, 32)
	copy(data[:IFNAMSIZ], []byte(name))
	if code := dart.SyscallBpfIoctl(fd, BIOCSETIF, data, nil); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
func BpfTimeout(fd int) (*Timeval, error) {
	data := make([]byte, 16)
	if code := dart.SyscallBpfIoctl(fd, BIOCGRTIMEOUT, data, nil); code != 0 {
		return nil, syscallErrno(code)
	}
	tv := &Timeval{Sec: int64(binary.NativeEndian.Uint64(data)), Usec: int32(binary.NativeEndian.Uint32(data[8:]))}
	copy(tv.Pad_cgo_0[:], data[12:])
	return tv, nil
}
func SetBpfTimeout(fd int, tv *Timeval) error {
	var data []byte
	if tv != nil {
		data = make([]byte, 16)
		binary.NativeEndian.PutUint64(data, uint64(tv.Sec))
		binary.NativeEndian.PutUint32(data[8:], uint32(tv.Usec))
		copy(data[12:], tv.Pad_cgo_0[:])
	}
	code := dart.SyscallBpfIoctl(fd, BIOCSRTIMEOUT, data, nil)
	if tv != nil {
		tv.Sec = int64(binary.NativeEndian.Uint64(data))
		tv.Usec = int32(binary.NativeEndian.Uint32(data[8:]))
		copy(tv.Pad_cgo_0[:], data[12:])
	}
	if code != 0 {
		return syscallErrno(code)
	}
	return nil
}
func BpfStats(fd int) (*BpfStat, error) {
	data := make([]byte, 8)
	if code := dart.SyscallBpfIoctl(fd, BIOCGSTATS, data, nil); code != 0 {
		return nil, syscallErrno(code)
	}
	return &BpfStat{Recv: binary.NativeEndian.Uint32(data), Drop: binary.NativeEndian.Uint32(data[4:])}, nil
}
func SetBpfImmediate(fd, m int) error { _, err := bpfScalar(fd, BIOCIMMEDIATE, m); return err }
func putBpfInsn(data []byte, value BpfInsn) {
	binary.NativeEndian.PutUint16(data, value.Code)
	data[2], data[3] = value.Jt, value.Jf
	binary.NativeEndian.PutUint32(data[4:], value.K)
}
func SetBpf(fd int, i []BpfInsn) error {
	first := i[0]
	data := make([]byte, 16)
	binary.NativeEndian.PutUint32(data, uint32(len(i)))
	insns := make([]byte, len(i)*8)
	putBpfInsn(insns, first)
	for n := 1; n < len(i); n++ {
		putBpfInsn(insns[n*8:], i[n])
	}
	if code := dart.SyscallBpfIoctl(fd, BIOCSETF, data, insns); code != 0 {
		return syscallErrno(code)
	}
	return nil
}
func CheckBpfVersion(fd int) error {
	data := make([]byte, 4)
	if code := dart.SyscallBpfIoctl(fd, BIOCVERSION, data, nil); code != 0 {
		return syscallErrno(code)
	}
	if binary.NativeEndian.Uint16(data) != BPF_MAJOR_VERSION || binary.NativeEndian.Uint16(data[2:]) != BPF_MINOR_VERSION {
		return EINVAL
	}
	return nil
}
func BpfHeadercmpl(fd int) (int, error) { return bpfScalar(fd, BIOCGHDRCMPLT, 0) }
func SetBpfHeadercmpl(fd, f int) error  { _, err := bpfScalar(fd, BIOCSHDRCMPLT, f); return err }
