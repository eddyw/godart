// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "encoding/binary"

func UnixCredentials(ucred *Ucred) []byte {
	data := make([]byte, CmsgSpace(SizeofUcred))
	binary.NativeEndian.PutUint64(data, uint64(CmsgLen(SizeofUcred)))
	binary.NativeEndian.PutUint32(data[8:], uint32(SOL_SOCKET))
	binary.NativeEndian.PutUint32(data[12:], uint32(SCM_CREDENTIALS))
	payload := data[CmsgLen(0):]
	binary.NativeEndian.PutUint32(payload, uint32(ucred.Pid))
	binary.NativeEndian.PutUint32(payload[4:], ucred.Uid)
	binary.NativeEndian.PutUint32(payload[8:], ucred.Gid)
	return data
}

func ParseUnixCredentials(message *SocketControlMessage) (*Ucred, error) {
	if message.Header.Level != SOL_SOCKET {
		return nil, EINVAL
	}
	if message.Header.Type != SCM_CREDENTIALS {
		return nil, EINVAL
	}
	if len(message.Data) < SizeofUcred {
		return nil, EINVAL
	}
	return &Ucred{
		Pid: int32(binary.NativeEndian.Uint32(message.Data)),
		Uid: binary.NativeEndian.Uint32(message.Data[4:]),
		Gid: binary.NativeEndian.Uint32(message.Data[8:]),
	}, nil
}
