// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart && (linux || darwin) && (amd64 || arm64)

package syscall

import "syscall/dart"

func Sendmsg(fd int, data, oob []byte, to Sockaddr, flags int) error {
	_, err := SendmsgN(fd, data, oob, to, flags)
	return err
}

func SendmsgN(fd int, data, oob []byte, to Sockaddr, flags int) (int, error) {
	var address []byte
	if to != nil {
		var err error
		address, err = to.sockaddrBytes()
		if err != nil {
			return 0, err
		}
	}
	payload, err := messagePayload(fd, data, len(oob))
	if err != nil {
		return 0, err
	}
	n, code := dart.SyscallSendmsgN(fd, payload, oob, flags, address)
	if code != 0 {
		return 0, syscallErrno(code)
	}
	if len(oob) > 0 && len(data) == 0 {
		return 0, nil
	}
	return n, nil
}

func Recvmsg(fd int, data, oob []byte, flags int) (n, oobn, recvflags int, from Sockaddr, err error) {
	payload, err := messagePayload(fd, data, len(oob))
	if err != nil {
		return
	}
	address := make([]byte, SizeofSockaddrAny)
	n, oobn, recvflags, code := dart.SyscallRecvmsg(fd, payload, oob, flags, address)
	if code != 0 {
		return n, oobn, recvflags, nil, syscallErrno(code)
	}
	from, err = messageSource(address)
	return
}
