//go:build dart

// This file patches the file stat_actime2.go of the package archive/tar.
//
// The real file reads syscall.Stat_t, and the patched stat_unix.go calls
// neither function of it.
package tar
