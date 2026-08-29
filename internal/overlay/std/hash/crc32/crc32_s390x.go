//go:build dart

// This file patches the file crc32_s390x.go of the package hash/crc32.
//
// The real file reads the package internal/cpu and declares each update
// function with no body, because assembly holds the code. This file reports
// that no hardware form is available, thus each caller takes the table of the
// portable path.
package crc32

func archAvailableIEEE() bool                    { return false }
func archInitIEEE()                              { panic("not available") }
func archUpdateIEEE(crc uint32, p []byte) uint32 { panic("not available") }

func archAvailableCastagnoli() bool                    { return false }
func archInitCastagnoli()                              { panic("not available") }
func archUpdateCastagnoli(crc uint32, p []byte) uint32 { panic("not available") }
