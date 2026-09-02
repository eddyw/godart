//go:build dart

package x509

import (
	"bytes"
	"encoding/pem"
)

// A CertPool holds a set of certificates.
//
// The real package builds a chain with a pool. This overlay holds no chain
// builder, thus a pool is a set alone. A program passes it to another package
// that owns the trust decision.
type CertPool struct {
	certs []*Certificate
	// haveSum names the DER of each certificate that the pool holds, thus
	// AddCert takes 1 certificate 1 time.
	haveSum map[string]bool
}

// NewCertPool gives an empty pool.
func NewCertPool() *CertPool {
	return &CertPool{haveSum: map[string]bool{}}
}

// len gives the count of the certificates of the pool.
func (s *CertPool) len() int {
	if s == nil {
		return 0
	}
	return len(s.certs)
}

// AddCert puts a certificate into the pool. It panics with a nil certificate.
func (s *CertPool) AddCert(cert *Certificate) {
	if cert == nil {
		panic("adding nil Certificate to CertPool")
	}
	if s.haveSum == nil {
		s.haveSum = map[string]bool{}
	}
	key := string(cert.Raw)
	if s.haveSum[key] {
		return
	}
	s.haveSum[key] = true
	s.certs = append(s.certs, cert)
}

// AppendCertsFromPEM reads each PEM block of the type CERTIFICATE and puts the
// certificate into the pool. It reports whether it added at least 1
// certificate.
func (s *CertPool) AppendCertsFromPEM(pemCerts []byte) (ok bool) {
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}
		cert, err := ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		s.AddCert(cert)
		ok = true
	}
	return ok
}

// Subjects gives the DER of the subject of each certificate of the pool.
//
// Deprecated: the pool of the real package can hold a certificate of the
// system, and that pool gives no subject. This pool always holds each one.
func (s *CertPool) Subjects() [][]byte {
	res := make([][]byte, s.len())
	for i, cert := range s.certs {
		res[i] = cert.RawSubject
	}
	return res
}

// Equal reports whether the 2 pools hold the same certificates in the same
// order.
func (s *CertPool) Equal(other *CertPool) bool {
	if s == nil || other == nil {
		return s == other
	}
	if s.len() != other.len() {
		return false
	}
	for i := range s.certs {
		if !bytes.Equal(s.certs[i].Raw, other.certs[i].Raw) {
			return false
		}
	}
	return true
}

// Clone gives a copy of the pool.
func (s *CertPool) Clone() *CertPool {
	if s == nil {
		return nil
	}
	out := NewCertPool()
	out.certs = make([]*Certificate, len(s.certs))
	copy(out.certs, s.certs)
	for k, v := range s.haveSum {
		out.haveSum[k] = v
	}
	return out
}
