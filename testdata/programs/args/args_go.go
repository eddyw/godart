//go:build !dart

package main

import "os"

// arguments gives the arguments of the program, without the name of the
// program.
func arguments() []string { return os.Args[1:] }
