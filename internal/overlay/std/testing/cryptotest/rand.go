//go:build dart

// This file patches the file rand.go of the package testing/cryptotest.
//
// The real file reaches crypto/internal/rand.SetTestingReader and
// testing.checkParallel with a go:linkname directive, and the subset holds no
// such directive. The package internal/linkhook carries the 2 names, because
// the import rule of Go keeps each 1 apart from this package.
package cryptotest

import (
	cryptorand "crypto/rand"
	"internal/byteorder"
	"internal/linkhook"
	mathrand "math/rand/v2"
	"sync"
	"testing"
)

// SetGlobalRandom sets a global, deterministic cryptographic randomness source
// for the duration of test t. It affects crypto/rand, and all implicit sources
// of cryptographic randomness in the crypto/... packages.
//
// SetGlobalRandom may be called multiple times in the same test to reset the
// random stream or change the seed.
//
// Because SetGlobalRandom affects the whole process, it cannot be used in
// parallel tests or tests with parallel ancestors.
//
// Note that the way cryptographic algorithms use randomness is generally not
// specified and may change over time. Thus, if a test expects a specific output
// from a cryptographic function, it may fail in the future even if it uses
// SetGlobalRandom.
func SetGlobalRandom(t *testing.T, seed uint64) {
	if t == nil {
		panic("cryptotest: SetGlobalRandom called with a nil *testing.T")
	}
	if !testing.Testing() {
		panic("cryptotest: SetGlobalRandom used in a non-test binary")
	}
	linkhook.CheckParallel(t)

	var s [32]byte
	byteorder.LEPutUint64(s[:8], seed)
	r := &lockedReader{r: mathrand.NewChaCha8(s)}

	// A program that reads no crypto package beyond crypto/rand holds no
	// crypto/internal/rand, thus the field stays nil.
	if linkhook.SetCryptoReader != nil {
		linkhook.SetCryptoReader(r)
	}
	previous := cryptorand.Reader
	cryptorand.Reader = r
	t.Cleanup(func() {
		cryptorand.Reader = previous
		if linkhook.SetCryptoReader != nil {
			linkhook.SetCryptoReader(nil)
		}
	})
}

// lockedReader holds a lock, because 2 goroutines can read the global source
// at the same time.
type lockedReader struct {
	mu sync.Mutex
	r  *mathrand.ChaCha8
}

func (lr *lockedReader) Read(b []byte) (n int, err error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.r.Read(b)
}
