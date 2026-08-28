//go:build dart

package main

import "syscall/dart"

// dialFailed reports whether the dial of the port failed. The dial gives up
// control, thus this function and main each become a state machine.
func dialFailed(port int) bool {
	_, cause := dart.DialTCP("127.0.0.1", port)
	return cause != ""
}
