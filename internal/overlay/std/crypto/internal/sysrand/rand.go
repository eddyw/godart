//go:build dart

// Package sysrand gives random bytes of the operating system.
//
// The real package reads getrandom of Linux or arc4random of macOS through the
// package unsafe. This overlay reads Random.secure of Dart, which reads the
// same generator of the operating system. The web build reads window.crypto.
package sysrand

import "syscall/dart"

// Read fills b with random bytes of the operating system. It always fills b,
// and it never gives an error.
func Read(b []byte) {
	copy(b, dart.RandomBytes(len(b)))
}
