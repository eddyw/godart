//go:build dart

// This file patches the file rand_fips140v1.26.go of the package
// crypto/internal/rand.
//
// The body of fips140SetTestingReader is the body of Go. The init gives
// SetTestingReader to testing/cryptotest, which reaches that name with a
// go:linkname directive. The import rule of Go rejects a direct import of this
// package from testing/cryptotest.
package rand

import (
	"crypto/internal/fips140/drbg"
	"internal/linkhook"
	"io"
)

func init() {
	linkhook.SetCryptoReader = SetTestingReader
}

func fips140SetTestingReader(r io.Reader) {
	drbg.SetTestingReader(r)
}
