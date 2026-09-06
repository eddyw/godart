//go:build dart

// Package asan gives the flag of the address sanitizer of Go. A build of Go
// sets the flag with -asan, and the Dart output holds no sanitizer, thus the
// flag stays off.
package asan

// Enabled reports whether the address sanitizer runs.
const Enabled = false
