package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func certificate(text string) *x509.Certificate {
	block, _ := pem.Decode([]byte(text))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	return cert
}

func must(ok bool) {
	if !ok {
		panic("certificate trust differs")
	}
}

func panics(f func()) (result bool) {
	defer func() { result = recover() != nil }()
	f()
	return
}

func main() {
	if mode := os.Getenv("GART_TRUST_CASE"); mode != "" {
		environment(mode)
		return
	}

	pool, err := x509.SystemCertPool()
	must(pool != nil && err == nil)
	fmt.Println("system roots", pool != nil, err == nil)
	leaf := certificate(gtsRoot)
	intermediates := x509.NewCertPool()
	must(intermediates.AppendCertsFromPEM([]byte(gtsIntermediate)))
	opts := x509.VerifyOptions{Intermediates: intermediates, CurrentTime: time.Unix(1677615892, 0)}
	customGoogle := x509.NewCertPool()
	customGoogle.AddCert(leaf)
	opts.Roots = customGoogle
	chains, err := leaf.Verify(opts)
	must(err == nil && len(chains) > 0)
	fmt.Println("explicit chain true")
	opts.DNSName = "invalid.example"
	_, err = leaf.Verify(opts)
	var hostname x509.HostnameError
	must(errors.As(err, &hostname))
	opts.DNSName = ""
	opts.CurrentTime = leaf.NotAfter.Add(time.Hour)
	_, err = leaf.Verify(opts)
	var invalid x509.CertificateInvalidError
	must(errors.As(err, &invalid) && invalid.Reason == x509.Expired)
	fmt.Println("hostname and expiry true")

	untrusted := certificate(selfSigned)
	_, err = untrusted.Verify(x509.VerifyOptions{CurrentTime: untrusted.NotBefore.Add(time.Minute)})
	must(err != nil)
	custom := pool.Clone()
	custom.AddCert(untrusted)
	_, err = untrusted.Verify(x509.VerifyOptions{Roots: custom, CurrentTime: untrusted.NotBefore.Add(time.Minute), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
	must(err == nil)
	fmt.Println("custom roots true")

	must(panics(func() { x509.SetFallbackRoots(nil) }))
	must(os.Setenv("GODEBUG", "x509usefallbackroots=1") == nil)
	fallback := x509.NewCertPool()
	fallback.AddCert(untrusted)
	x509.SetFallbackRoots(fallback)
	_, err = untrusted.Verify(x509.VerifyOptions{CurrentTime: untrusted.NotBefore.Add(time.Minute), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
	must(err == nil)
	_, err = leaf.Verify(x509.VerifyOptions{Intermediates: intermediates, CurrentTime: time.Unix(1677615892, 0)})
	must(err != nil)
	must(panics(func() { x509.SetFallbackRoots(fallback) }))
	fmt.Println("fallback roots true")
}

func environment(mode string) {
	dir, err := os.MkdirTemp("", "gart-system-roots-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "root.pem")
	must(os.WriteFile(file, []byte(selfSigned), 0600) == nil)
	must(os.Setenv("SSL_CERT_FILE", filepath.Join(dir, "missing")) == nil)
	must(os.Setenv("SSL_CERT_DIR", filepath.Join(dir, "missing-dir")) == nil)
	switch mode {
	case "file", "platform":
		must(os.Setenv("SSL_CERT_FILE", file) == nil)
	case "directories":
		other := filepath.Join(dir, "other")
		must(os.Mkdir(other, 0700) == nil)
		must(os.Symlink("root.pem", filepath.Join(dir, "root-link")) == nil)
		must(os.Setenv("SSL_CERT_DIR", other+string(os.PathListSeparator)+dir) == nil)
	case "missing", "early-fallback":
	default:
		panic(mode)
	}
	if mode == "platform" {
		must(os.Setenv("GODEBUG", "x509sslcertoverrideplatform=0") == nil)
	}
	if mode == "early-fallback" {
		must(os.Setenv("GODEBUG", "x509usefallbackroots=1") == nil)
		roots := x509.NewCertPool()
		roots.AddCert(certificate(selfSigned))
		x509.SetFallbackRoots(roots)
	}
	roots, err := x509.SystemCertPool()
	must(err == nil && roots != nil)
	cert := certificate(selfSigned)
	_, err = cert.Verify(x509.VerifyOptions{CurrentTime: cert.NotBefore.Add(time.Minute), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
	trusted := mode != "missing" && !(mode == "platform" && (runtime.GOOS == "darwin" || runtime.GOOS == "windows"))
	must((err == nil) == trusted)
	fmt.Println(mode, "true")
}
