//go:build dart

// This file holds the trust store of the package. Each line of the fallback
// comes from the file root.go of crypto/x509 of Go.
//
// The secure socket of Dart owns the trust decision of a handshake, and it
// exports no root certificate. Thus loadSystemRoots always reports a failure of
// the load, exactly as Go does on a platform that holds no store.
// SetFallbackRoots then gives the pool that Verify reads.
package x509

import (
	"internal/godebug"
	"sync"
)

var (
	once             sync.Once
	systemRootsMu    sync.RWMutex
	systemRoots      *CertPool
	systemRootsErr   error
	fallbacksSet     bool
	useFallbackRoots bool
)

func systemRootsPool() *CertPool {
	once.Do(initSystemRoots)
	systemRootsMu.RLock()
	defer systemRootsMu.RUnlock()
	return systemRoots
}

func initSystemRoots() {
	systemRootsMu.Lock()
	defer systemRootsMu.Unlock()

	fallbackRoots := systemRoots
	systemRoots, systemRootsErr = loadSystemRoots()
	if systemRootsErr != nil {
		systemRoots = nil
	}

	if fallbackRoots == nil {
		return // no fallbacks to try
	}

	systemCertsAvail := systemRoots != nil && (systemRoots.len() > 0 || systemRoots.systemPool)

	if !useFallbackRoots && systemCertsAvail {
		return
	}

	if useFallbackRoots && systemCertsAvail {
		x509usefallbackroots.IncNonDefault() // overriding system certs with fallback certs.
	}

	systemRoots, systemRootsErr = fallbackRoots, nil
}

var x509usefallbackroots = godebug.New("x509usefallbackroots")

// SetFallbackRoots names the roots that the check of a certificate reads, when
// the caller names no root of its own and the platform holds no store. It
// panics for a nil pool.
//
// The Dart target holds no store of the platform, thus this call always names
// the roots that Verify reads.
//
// SetFallbackRoots runs 1 time. A second call panics.
func SetFallbackRoots(roots *CertPool) {
	if roots == nil {
		panic("roots must be non-nil")
	}

	systemRootsMu.Lock()
	defer systemRootsMu.Unlock()

	if fallbacksSet {
		panic("SetFallbackRoots has already been called")
	}
	fallbacksSet = true

	// initSystemRoots may not have run yet. The call handles that case here,
	// because a program calls SetFallbackRoots at its start most of the time.
	if systemRoots == nil && systemRootsErr == nil {
		systemRoots = roots
		useFallbackRoots = x509usefallbackroots.Value() == "1"
		return
	}

	once.Do(func() { panic("unreachable") }) // the roots of the system stand already.

	forceFallbackRoots := x509usefallbackroots.Value() == "1"
	systemCertsAvail := systemRoots != nil && (systemRoots.len() > 0 || systemRoots.systemPool)

	if !forceFallbackRoots && systemCertsAvail {
		return
	}

	if forceFallbackRoots && systemCertsAvail {
		x509usefallbackroots.IncNonDefault() // overriding system certs with fallback certs.
	}

	systemRoots, systemRootsErr = roots, nil
}

// errNoTrustStore names the cause that Dart gives.
var errNoTrustStore = errorText("the secure socket of Dart owns the trust " +
	"store, and it exports no root certificate")

// errorText builds an error of a text.
type errorText string

func (e errorText) Error() string { return string(e) }

// loadSystemRoots always reports the failure of the load. Dart holds the store
// inside its secure socket, and it exports no certificate of it.
func loadSystemRoots() (*CertPool, error) {
	return nil, SystemRootsError{Err: errNoTrustStore}
}
