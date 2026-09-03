//go:build dart

package x509

import "encoding/pem"

// This file holds the 2 names of the trust store and 1 test of a PEM block.
//
// The secure socket of Dart owns the trust decision of a handshake, and it
// exports no root certificate. Thus SystemCertPool always reports a failure
// of the load, exactly as Go does on a platform that holds no store.
//
// `SetFallbackRoots` stays out. Go reads the fallback pool inside `Verify`
// alone, and the overlay holds no `Verify`. A call that writes a pool that no
// code reads would lie.

// A SystemRootsError reports a failure of the load of the root certificates
// of the platform.
type SystemRootsError struct {
	Err error
}

func (se SystemRootsError) Error() string {
	msg := "x509: failed to load system roots and no roots provided"
	if se.Err != nil {
		return msg + "; " + se.Err.Error()
	}
	return msg
}

// Unwrap gives the cause of the failure.
func (se SystemRootsError) Unwrap() error { return se.Err }

// errNoTrustStore names the cause that Dart gives.
var errNoTrustStore = errorText("the secure socket of Dart owns the trust " +
	"store, and it exports no root certificate")

// errorText builds an error of a text. The package imports no errors package
// at this point, thus the type stands here.
type errorText string

func (e errorText) Error() string { return string(e) }

// SystemCertPool always gives a SystemRootsError.
//
// Go reads the root certificates of the platform. Dart holds the store inside
// its secure socket, and it exports no certificate of it. A pool with no root
// would report an empty trust store, thus the overlay reports the failure of
// the load.
func SystemCertPool() (*CertPool, error) {
	return nil, SystemRootsError{Err: errNoTrustStore}
}

// IsEncryptedPEMBlock reports a PEM block that RFC 1423 encrypted.
//
// The overlay holds no DecryptPEMBlock and no EncryptPEMBlock, because
// `crypto/aes` and `crypto/des` each stand outside the surface. A program
// reads this test and reports the block that it cannot open.
//
// Deprecated: The PEM encryption of RFC 1423 is insecure by design. It
// authenticates no ciphertext, thus an attacker can read the plain text with
// a padding oracle attack.
func IsEncryptedPEMBlock(b *pem.Block) bool {
	_, ok := b.Headers["DEK-Info"]
	return ok
}
