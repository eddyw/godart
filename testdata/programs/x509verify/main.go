// Command x509verify builds a chain of 3 certificates and checks it. The
// program signs each certificate itself, thus each answer below comes from a
// real signature and not from a fixed byte set.
//
// A forged certificate must fail the check. The program builds 1, and it
// reports the error that Verify gives.
package main

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

func must(err error) {
	if err != nil {
		fmt.Println("unexpected error:", err)
		panic("stop")
	}
}

var start = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
var end = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
var now = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func template(cn string, serial int64, isCA bool) *x509.Certificate {
	t := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: cn, Organization: []string{"gart"}},
		NotBefore:             start,
		NotAfter:              end,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	if isCA {
		t.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	} else {
		t.KeyUsage = x509.KeyUsageDigitalSignature
		t.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		t.DNSNames = []string{"example.com", "*.example.com"}
		t.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1)}
	}
	return t
}

func issue(t, parent *x509.Certificate, pub any, signer crypto.Signer) *x509.Certificate {
	der, err := x509.CreateCertificate(rand.Reader, t, parent, pub, signer)
	must(err)
	c, err := x509.ParseCertificate(der)
	must(err)
	return c
}

func main() {
	// The root of the chain, with a key of P-256.
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(err)
	rootTmpl := template("gart root", 1, true)
	root := issue(rootTmpl, rootTmpl, rootKey.Public(), rootKey)
	fmt.Println(root.Subject.CommonName, root.IsCA, root.Version)
	fmt.Println(root.CheckSignatureFrom(root))

	// The intermediate, with a key of RSA 2048.
	interKey, err := rsa.GenerateKey(rand.Reader, 2048)
	must(err)
	inter := issue(template("gart intermediate", 2, true), root, interKey.Public(), rootKey)
	fmt.Println(inter.Subject.CommonName, inter.IsCA)
	fmt.Println(inter.CheckSignatureFrom(root))
	fmt.Println(inter.CheckSignatureFrom(inter) != nil)

	// The leaf, with a key of Ed25519.
	leafPub, leafPriv, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	leaf := issue(template("example.com", 3, false), inter, leafPub, interKey)
	fmt.Println(leaf.Subject.CommonName, leaf.IsCA, leaf.DNSNames)
	fmt.Println(leaf.CheckSignatureFrom(inter))
	fmt.Println(leaf.PublicKeyAlgorithm, leaf.SignatureAlgorithm)
	_ = leafPriv

	// The chain builds from the leaf to the root.
	roots := x509.NewCertPool()
	roots.AddCert(root)
	inters := x509.NewCertPool()
	inters.AddCert(inter)
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: inters,
		CurrentTime:   now,
		DNSName:       "example.com",
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	chains, err := leaf.Verify(opts)
	must(err)
	fmt.Println(len(chains), len(chains[0]))
	for _, c := range chains[0] {
		fmt.Println(" ", c.Subject.CommonName)
	}

	// A wildcard name of the certificate passes.
	opts.DNSName = "www.example.com"
	_, err = leaf.Verify(opts)
	fmt.Println("wildcard:", err)

	// A name that the certificate does not hold fails.
	opts.DNSName = "other.test"
	_, err = leaf.Verify(opts)
	fmt.Println("bad name:", err)

	// A time outside the validity fails.
	opts.DNSName = "example.com"
	opts.CurrentTime = time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = leaf.Verify(opts)
	fmt.Println("expired:", err)

	// A missing intermediate fails.
	opts.CurrentTime = now
	opts.Intermediates = x509.NewCertPool()
	_, err = leaf.Verify(opts)
	fmt.Println("no intermediate:", err)

	// A key usage that the chain does not hold fails.
	opts.Intermediates = inters
	opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}
	_, err = leaf.Verify(opts)
	fmt.Println("bad usage:", err)

	// A forged certificate must fail. The program builds a leaf with the same
	// subject and the same key, and it signs the leaf with another key that
	// carries the name of the intermediate.
	fakeKey, err := rsa.GenerateKey(rand.Reader, 2048)
	must(err)
	fakeInterTmpl := template("gart intermediate", 2, true)
	fakeInter := issue(fakeInterTmpl, fakeInterTmpl, fakeKey.Public(), fakeKey)
	forged := issue(template("example.com", 3, false), fakeInter, leafPub, fakeKey)
	fmt.Println("forged parses:", forged.Subject.CommonName)
	fmt.Println("forged signature of the real intermediate:",
		forged.CheckSignatureFrom(inter) != nil)
	opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	_, err = forged.Verify(opts)
	fmt.Println("forged:", err != nil)

	// A chain that swaps the signature of the real leaf onto the forged bytes
	// must fail too.
	swapped := *forged
	swapped.Signature = leaf.Signature
	fmt.Println("swapped:", checkRaw(&swapped, inter) != nil)

	// The key family: the write and the read of each form.
	pkix1, err := x509.MarshalPKIXPublicKey(rootKey.Public())
	must(err)
	back, err := x509.ParsePKIXPublicKey(pkix1)
	must(err)
	fmt.Println("pkix ecdsa:", back.(*ecdsa.PublicKey).Equal(rootKey.Public()))

	pkix2, err := x509.MarshalPKIXPublicKey(leafPub)
	must(err)
	back2, err := x509.ParsePKIXPublicKey(pkix2)
	must(err)
	fmt.Println("pkix ed25519:", back2.(ed25519.PublicKey).Equal(leafPub))

	p1 := x509.MarshalPKCS1PrivateKey(interKey)
	k1, err := x509.ParsePKCS1PrivateKey(p1)
	must(err)
	fmt.Println("pkcs1 private:", k1.Equal(interKey))
	p1pub := x509.MarshalPKCS1PublicKey(&interKey.PublicKey)
	k1pub, err := x509.ParsePKCS1PublicKey(p1pub)
	must(err)
	fmt.Println("pkcs1 public:", k1pub.Equal(&interKey.PublicKey))

	p8, err := x509.MarshalPKCS8PrivateKey(rootKey)
	must(err)
	k8, err := x509.ParsePKCS8PrivateKey(p8)
	must(err)
	fmt.Println("pkcs8 ecdsa:", k8.(*ecdsa.PrivateKey).Equal(rootKey))

	p8b, err := x509.MarshalPKCS8PrivateKey(leafPriv)
	must(err)
	k8b, err := x509.ParsePKCS8PrivateKey(p8b)
	must(err)
	fmt.Println("pkcs8 ed25519:", k8b.(ed25519.PrivateKey).Equal(leafPriv))

	ec, err := x509.MarshalECPrivateKey(rootKey)
	must(err)
	kec, err := x509.ParseECPrivateKey(ec)
	must(err)
	fmt.Println("sec1:", kec.Equal(rootKey))

	// The certificate request.
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "csr.example.com"},
		DNSNames: []string{"csr.example.com"},
	}, rootKey)
	must(err)
	csr, err := x509.ParseCertificateRequest(csrDER)
	must(err)
	fmt.Println("csr:", csr.Subject.CommonName, csr.DNSNames, csr.CheckSignature())

	// The revocation list.
	crlDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:     big.NewInt(7),
		ThisUpdate: start,
		NextUpdate: end,
		RevokedCertificateEntries: []x509.RevocationListEntry{
			{SerialNumber: big.NewInt(3), RevocationTime: start, ReasonCode: 1},
		},
	}, root, rootKey)
	must(err)
	crl, err := x509.ParseRevocationList(crlDER)
	must(err)
	fmt.Println("crl:", crl.Number, len(crl.RevokedCertificateEntries),
		crl.RevokedCertificateEntries[0].SerialNumber, crl.CheckSignatureFrom(root))

	// The PEM encryption of RFC 1423.
	block, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", p1,
		[]byte("secret"), x509.PEMCipherAES256)
	must(err)
	fmt.Println("pem encrypted:", x509.IsEncryptedPEMBlock(block), block.Type)
	plain, err := x509.DecryptPEMBlock(block, []byte("secret"))
	must(err)
	fmt.Println("pem round trip:", len(plain) == len(p1))
	// Legacy PEM can accept a wrong password when random data has valid padding.
	badPasswordBlock, err := x509.EncryptPEMBlock(bytes.NewReader([]byte("0123456789abcdef")),
		"X", []byte("fixed PEM payload"), []byte("secret"), x509.PEMCipherAES256)
	must(err)
	_, err = x509.DecryptPEMBlock(badPasswordBlock, []byte("wrong"))
	fmt.Println("pem bad password:", err)
	fmt.Println(pem.EncodeToMemory(&pem.Block{Type: "X", Bytes: []byte{1, 2, 3}}) != nil)

	// SetFallbackRoots names the roots that a check reads when the caller names
	// none. It runs 1 time, and a second call panics.
	//
	// The behavior after the call differs between Go and gart, thus the
	// program checks the panic alone. Go on macOS reads the verifier of the
	// platform, and the Dart target reaches no such verifier. Read
	// docs/parity.md.
	x509.SetFallbackRoots(roots)
	fmt.Println("second call panics:", panics(func() { x509.SetFallbackRoots(roots) }))
	fmt.Println("nil pool panics:", panics(func() { x509.SetFallbackRoots(nil) }))

	// A signature of the leaf over a message, and the check of it.
	msg := []byte("the message")
	sum := sha256.Sum256(msg)
	sig := ed25519.Sign(leafPriv, msg)
	fmt.Println("leaf signature:", leaf.CheckSignature(x509.PureEd25519, msg, sig) == nil)
	fmt.Println("leaf wrong digest:", leaf.CheckSignature(x509.PureEd25519, sum[:], sig) != nil)
}

// checkRaw checks the signature of a certificate that the program changed. The
// parse holds the raw bytes, thus the check reads the new signature.
func checkRaw(c *x509.Certificate, parent *x509.Certificate) error {
	return parent.CheckSignature(c.SignatureAlgorithm, c.RawTBSCertificate, c.Signature)
}

// panics reports whether f stops the program.
func panics(f func()) (out bool) {
	defer func() {
		if recover() != nil {
			out = true
		}
	}()
	f()
	return false
}
