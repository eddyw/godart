//go:build dart

// This file patches the file stat_unix.go of the package archive/tar.
//
// The real file reads syscall.Stat_t and os/user, and the surface holds
// neither 1. It leaves sysStat nil, thus FileInfoHeader skips the owner name
// and the group name of a header.
package tar
