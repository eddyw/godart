// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && darwin && (amd64 || arm64)

package syscall

func messagePayload(fd int, data []byte, controlLength int) ([]byte, error) {
	if len(data) == 0 && controlLength > 0 {
		return make([]byte, 1), nil
	}
	return data, nil
}

func messageSource(address []byte) (Sockaddr, error) {
	if address[1] == AF_UNSPEC {
		return nil, nil
	}
	return sockaddrFromBytes(address)
}
