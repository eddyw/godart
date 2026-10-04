// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && linux && (amd64 || arm64)

package syscall

import "encoding/binary"

func messagePayload(fd int, data []byte, controlLength int) ([]byte, error) {
	if len(data) == 0 && controlLength > 0 {
		typ, err := GetsockoptInt(fd, SOL_SOCKET, SO_TYPE)
		if err != nil {
			return nil, err
		}
		if typ != SOCK_DGRAM {
			return make([]byte, 1), nil
		}
	}
	return data, nil
}

func messageSource(address []byte) (Sockaddr, error) {
	if binary.NativeEndian.Uint16(address) == AF_UNSPEC {
		return nil, nil
	}
	return sockaddrFromBytes(address)
}
