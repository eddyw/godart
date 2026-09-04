//go:build dart

// This file patches the file cpu.go of the package
// crypto/internal/fips140deps/cpu.
//
// The real file reads internal/cpu and internal/goarch. The Dart VM runs no
// instruction of the machine that a Go program names, thus each flag is false
// and every caller takes the portable path.
package cpu

const (
	BigEndian = false
	AMD64     = false
	ARM64     = false
	PPC64     = false
	PPC64le   = false
)

var (
	ARM64HasAES    = false
	ARM64HasPMULL  = false
	ARM64HasSHA2   = false
	ARM64HasSHA512 = false
	ARM64HasSHA3   = false

	LOONG64HasLSX  = false
	LOONG64HasLASX = false

	RISCV64HasV = false

	S390XHasAES    = false
	S390XHasAESCBC = false
	S390XHasAESCTR = false
	S390XHasAESGCM = false
	S390XHasECDSA  = false
	S390XHasGHASH  = false
	S390XHasSHA256 = false
	S390XHasSHA3   = false
	S390XHasSHA512 = false

	X86HasAES       = false
	X86HasADX       = false
	X86HasAVX       = false
	X86HasAVX2      = false
	X86HasBMI2      = false
	X86HasPCLMULQDQ = false
	X86HasSHA       = false
	X86HasSSE41     = false
	X86HasSSSE3     = false
)
