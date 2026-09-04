//go:build dart

// This file patches the file sig.go of the package crypto/internal/boring/sig.
//
// The real file declares each function with no body, because an assembly file
// holds a code signature for the linker. Each function performs no operation,
// thus an empty body holds the same behavior.
package sig

// BoringCrypto marks the presence of the BoringCrypto module.
func BoringCrypto() {}

// FIPSOnly marks the presence of the package crypto/tls/fipsonly.
func FIPSOnly() {}

// StandardCrypto marks the presence of the standard crypto of Go.
func StandardCrypto() {}
