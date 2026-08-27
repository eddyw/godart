//go:build dart

package main

import "syscall/dart"

// arguments gives the arguments of the program, without the name of the
// program.
func arguments() []string { return dart.Args() }
