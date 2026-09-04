//go:build dart

// Package abi holds the constants of the layout that the compiler of Go and
// the runtime of Go share.
//
// This overlay holds the 5 names of the PC/line table alone. debug/gosym reads
// them to name the version of a table that it parses, and each other name of
// the real package reads the package unsafe or a value of the runtime of Go.
package abi

// PCLnTabMagic is the version at the start of the PC/line table.
type PCLnTabMagic uint32

const (
	// Go12PCLnTabMagic is the value of Go 1.2 through Go 1.15.
	Go12PCLnTabMagic PCLnTabMagic = 0xfffffffb
	// Go116PCLnTabMagic is the value of Go 1.16 and Go 1.17.
	Go116PCLnTabMagic PCLnTabMagic = 0xfffffffa
	// Go118PCLnTabMagic is the value of Go 1.18 and Go 1.19.
	Go118PCLnTabMagic PCLnTabMagic = 0xfffffff0
	// Go120PCLnTabMagic is the value of Go 1.20 and each version after it.
	Go120PCLnTabMagic PCLnTabMagic = 0xfffffff1

	// CurrentPCLnTabMagic is the value that the current toolchain writes.
	CurrentPCLnTabMagic = Go120PCLnTabMagic
)
