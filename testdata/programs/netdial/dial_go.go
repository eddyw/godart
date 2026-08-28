//go:build !dart

package main

import (
	"net"
	"strconv"
)

func dialFailed(port int) bool {
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return true
	}
	conn.Close()
	return false
}
