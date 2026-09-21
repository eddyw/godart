// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

package x509

import (
	"errors"
	"fmt"
	"runtime"
	"syscall/dart"
)

var certFiles, certDirectories []string

func init() {
	if runtime.GOOS == "linux" || runtime.GOOS == "android" {
		certFiles = []string{
			"/etc/ssl/certs/ca-certificates.crt",
			"/etc/pki/tls/certs/ca-bundle.crt",
			"/etc/ssl/ca-bundle.pem",
			"/etc/pki/tls/cacert.pem",
			"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
			"/etc/ssl/cert.pem",
		}
		certDirectories = []string{"/etc/ssl/certs", "/etc/pki/tls/certs"}
		if runtime.GOOS == "android" {
			certDirectories = append(certDirectories,
				"/system/etc/security/cacerts", "/data/misc/keychain/certs-added")
		}
	}
}

func (c *Certificate) systemVerify(opts *VerifyOptions) ([][]*Certificate, error) {
	certs := []string{string(c.Raw)}
	if opts.Intermediates != nil {
		for _, lc := range opts.Intermediates.lazyCerts {
			cert, err := lc.getCert()
			if err != nil {
				return nil, err
			}
			certs = append(certs, string(cert.Raw))
		}
	}
	date := float64(opts.CurrentTime.Unix()) - 978307200 + float64(opts.CurrentTime.Nanosecond())/1e9
	der, code, message := dart.VerifySystemCertificates(certs, opts.DNSName, date, !opts.CurrentTime.IsZero())
	if message != "" {
		switch code {
		case -1:
			return nil, errors.New(message)
		case -67818:
			return nil, CertificateInvalidError{c, Expired, message}
		case -67602:
			return nil, HostnameError{c, opts.DNSName}
		case -67843:
			return nil, UnknownAuthorityError{Cert: c}
		default:
			return nil, fmt.Errorf("x509: %s", message)
		}
	}
	chain := make([]*Certificate, 0, len(der))
	for _, raw := range der {
		cert, err := ParseCertificate([]byte(raw))
		if err != nil {
			return nil, err
		}
		chain = append(chain, cert)
	}
	if len(chain) == 0 {
		return nil, errors.New("x509: platform verifier returned an empty chain")
	}
	if opts.DNSName != "" {
		if err := chain[0].VerifyHostname(opts.DNSName); err != nil {
			return nil, err
		}
	}
	keyUsages := opts.KeyUsages
	if len(keyUsages) == 0 {
		keyUsages = []ExtKeyUsage{ExtKeyUsageServerAuth}
	}
	for _, usage := range keyUsages {
		if usage == ExtKeyUsageAny {
			return [][]*Certificate{chain}, nil
		}
	}
	if !checkChainForKeyUsage(chain, keyUsages) {
		return nil, CertificateInvalidError{c, IncompatibleUsage, ""}
	}
	return [][]*Certificate{chain}, nil
}
