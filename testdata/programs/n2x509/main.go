// Command n2x509 proves the OID type, the policy extensions, the certificate
// request, and the revocation list that the sixteenth session added to the
// package crypto/x509.
package main

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

func main() {
	oids()
	extKeyUsageOID()
	policies()
	request()
	revocation()
	pemBlocks()
}

// oids proves each name of the OID type.
func oids() {
	o, err := x509.ParseOID("2.23.140.1.2.1")
	fmt.Println("parse:", o, err)
	_, err = x509.ParseOID("+1.2")
	fmt.Println("sign rejected:", err)
	_, err = x509.ParseOID("3.1")
	fmt.Println("bad first arc:", err)

	fromInts, err := x509.OIDFromInts([]uint64{2, 23, 140, 1, 2, 1})
	fmt.Println("from ints:", fromInts, err, fromInts.Equal(o))
	_, err = x509.OIDFromInts([]uint64{1})
	fmt.Println("short:", err)

	asn := asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 1}
	fromASN1, err := x509.OIDFromASN1OID(asn)
	fmt.Println("from asn1:", fromASN1, err, fromASN1.EqualASN1OID(asn))
	fmt.Println("other asn1:", fromASN1.EqualASN1OID(asn1.ObjectIdentifier{1, 2}))

	text, err := o.MarshalText()
	fmt.Println("text:", string(text), err)
	binary, err := o.MarshalBinary()
	fmt.Println("binary:", fmt.Sprintf("%x", binary), err)
	var back x509.OID
	fmt.Println("unmarshal text:", back.UnmarshalText(text), back.Equal(o))
	var raw x509.OID
	fmt.Println("unmarshal binary:", raw.UnmarshalBinary(binary), raw.Equal(o))
	fmt.Println("append text:", string(mustAppendText(o)))
	fmt.Println("append binary:", fmt.Sprintf("%x", mustAppendBinary(o)))

	// An arc above 64 bits keeps its value.
	big, err := x509.ParseOID("1.2.340282366920938463463374607431768211456")
	fmt.Println("big arc:", big, err)
}

func mustAppendText(o x509.OID) []byte {
	out, err := o.AppendText([]byte("oid="))
	if err != nil {
		panic(err)
	}
	return out
}

func mustAppendBinary(o x509.OID) []byte {
	out, err := o.AppendBinary(nil)
	if err != nil {
		panic(err)
	}
	return out
}

// extKeyUsageOID proves the OID of each extended key usage.
func extKeyUsageOID() {
	for _, eku := range []x509.ExtKeyUsage{
		x509.ExtKeyUsageAny, x509.ExtKeyUsageServerAuth,
		x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageOCSPSigning,
	} {
		fmt.Printf("eku %-20s %s\n", eku, eku.OID())
	}
}

// decode reads the first PEM block of a text.
func decode(text string) []byte {
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		panic("the text holds no PEM block")
	}
	return block.Bytes
}

// policies proves each policy field of a certificate.
func policies() {
	cert, err := x509.ParseCertificate(decode(leafPEM))
	if err != nil {
		fmt.Println("parse leaf:", err)
		return
	}
	fmt.Println("subject:", cert.Subject.CommonName, cert.DNSNames)
	var names []string
	for _, p := range cert.Policies {
		names = append(names, p.String())
	}
	fmt.Println("policies:", strings.Join(names, " "))
	var legacy []string
	for _, p := range cert.PolicyIdentifiers {
		legacy = append(legacy, p.String())
	}
	fmt.Println("policy identifiers:", strings.Join(legacy, " "))
	for _, m := range cert.PolicyMappings {
		fmt.Println("mapping:", m.IssuerDomainPolicy, "->", m.SubjectDomainPolicy)
	}
	fmt.Println("require explicit:", cert.RequireExplicitPolicy,
		cert.RequireExplicitPolicyZero)
	fmt.Println("inhibit mapping:", cert.InhibitPolicyMapping,
		cert.InhibitPolicyMappingZero)
	fmt.Println("inhibit any:", cert.InhibitAnyPolicy, cert.InhibitAnyPolicyZero)
	fmt.Println("extra extensions:", len(cert.ExtraExtensions))
	fmt.Println("unhandled critical:", len(cert.UnhandledCriticalExtensions))
}

// request proves ParseCertificateRequest.
func request() {
	csr, err := x509.ParseCertificateRequest(decode(csrPEM))
	if err != nil {
		fmt.Println("parse csr:", err)
		return
	}
	fmt.Println("csr version:", csr.Version)
	fmt.Println("csr subject:", csr.Subject.CommonName, csr.Subject.Organization)
	fmt.Println("csr algorithm:", csr.SignatureAlgorithm, csr.PublicKeyAlgorithm)
	fmt.Println("csr dns:", csr.DNSNames)
	fmt.Println("csr email:", csr.EmailAddresses)
	fmt.Println("csr ip:", csr.IPAddresses)
	for _, u := range csr.URIs {
		fmt.Println("csr uri:", u.String())
	}
	fmt.Println("csr raw sizes:", len(csr.Raw) > 0, len(csr.RawTBSCertificateRequest) > 0,
		len(csr.RawSubject) > 0, len(csr.RawSubjectPublicKeyInfo) > 0,
		len(csr.RawSignatureAlgorithm) > 0, len(csr.Signature) > 0)
	fmt.Println("csr extensions:", len(csr.Extensions), len(csr.ExtraExtensions),
		len(csr.Attributes))
	_, err = x509.ParseCertificateRequest([]byte{1, 2, 3})
	fmt.Println("bad csr rejected:", err != nil)
}

// revocation proves ParseRevocationList, ParseCRL, and ParseDERCRL.
func revocation() {
	der := decode(crlPEM)
	list, err := x509.ParseRevocationList(der)
	if err != nil {
		fmt.Println("parse crl:", err)
		return
	}
	fmt.Println("crl issuer:", list.Issuer.CommonName)
	fmt.Println("crl number:", list.Number)
	fmt.Println("crl times:", list.ThisUpdate.UTC().Format(time.RFC3339),
		list.NextUpdate.UTC().Format(time.RFC3339))
	fmt.Println("crl algorithm:", list.SignatureAlgorithm)
	for _, e := range list.RevokedCertificateEntries {
		fmt.Println("revoked:", e.SerialNumber, e.ReasonCode,
			e.RevocationTime.UTC().Format(time.RFC3339), len(e.Extensions),
			len(e.ExtraExtensions), len(e.Raw) > 0)
	}
	fmt.Println("deprecated list:", len(list.RevokedCertificates))
	fmt.Println("crl raw sizes:", len(list.Raw) > 0, len(list.RawTBSRevocationList) > 0,
		len(list.RawIssuer) > 0, len(list.RawSignatureAlgorithm) > 0,
		len(list.Signature) > 0, len(list.AuthorityKeyId) > 0)
	fmt.Println("crl extensions:", len(list.Extensions), len(list.ExtraExtensions))

	old, err := x509.ParseCRL([]byte(crlPEM))
	fmt.Println("parse crl pem:", err, len(old.TBSCertList.RevokedCertificates))
	old, err = x509.ParseDERCRL(der)
	fmt.Println("parse crl der:", err, old.TBSCertList.Version)
}

// pemBlocks proves IsEncryptedPEMBlock.
func pemBlocks() {
	plain, _ := pem.Decode([]byte(csrPEM))
	fmt.Println("plain encrypted:", x509.IsEncryptedPEMBlock(plain))
	locked := &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-128-CBC,00"},
		Bytes:   []byte{1, 2, 3},
	}
	fmt.Println("locked encrypted:", x509.IsEncryptedPEMBlock(locked))
}

const caPEM = `-----BEGIN CERTIFICATE-----
MIIBbzCCARWgAwIBAgIBATAKBggqhkjOPQQDAjAfMQswCQYDVQQKEwJOMjEQMA4G
A1UEAxMHbjIgcm9vdDAeFw0yNDAxMDIwMzA0MDVaFw00NDAxMDIwMzA0MDVaMB8x
CzAJBgNVBAoTAk4yMRAwDgYDVQQDEwduMiByb290MFkwEwYHKoZIzj0CAQYIKoZI
zj0DAQcDQgAE+BY4ff7ZzKO+ZBpKLwCxXCyON2XhBbLZqzU8XSqzalj/fiJTbXxx
G+ibsf8lQFb+9xNj9Q5rUHcuCu4rGkOB66NCMEAwDgYDVR0PAQH/BAQDAgEGMA8G
A1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFEtz6SVzEObd3+ulOyDx6aO/0v4rMAoG
CCqGSM49BAMCA0gAMEUCIQC3ns3naMy0Vg9UxSO+g3pNLphpaa6TNUu2KujLM+p9
WgIgZnMVYiCU0gHEynuFEsq4o2KyKiMwXqTXjgAOSGkaKr8=
-----END CERTIFICATE-----
`

const leafPEM = `-----BEGIN CERTIFICATE-----
MIICHTCCAcKgAwIBAgIBAjAKBggqhkjOPQQDAjAfMQswCQYDVQQKEwJOMjEQMA4G
A1UEAxMHbjIgcm9vdDAeFw0yNDAxMDIwMzA0MDVaFw00NDAxMDIwMzA0MDVaMCQx
CzAJBgNVBAoTAk4yMRUwEwYDVQQDEwxsZWFmLmV4YW1wbGUwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAAT4Fjh9/tnMo75kGkovALFcLI43ZeEFstmrNTxdKrNqWP9+
IlNtfHEb6Jux/yVAVv73E2P1DmtQdy4K7isaQ4Hro4HpMIHmMA4GA1UdDwEB/wQE
AwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIw
ADAfBgNVHSMEGDAWgBRLc+klcxDm3d/rpTsg8emjv9L+KzAnBgNVHREEIDAeggxs
ZWFmLmV4YW1wbGWCDioubGVhZi5leGFtcGxlMCAGA1UdIAQZMBcwCAYGZ4EMAQIB
MAsGCSsGAQQBho0fATAeBgNVHSEEFzAVMBMGBmeBDAECAQYJKwYBBAGGjR8CMA8G
A1UdJAQIMAaAAQCBAQMwCgYDVR02BAMCAQUwCgYIKoZIzj0EAwIDSQAwRgIhALnc
5+9Bn15b8Yf03M+ikpJhAyItoJAd0rX41HniiQT6AiEAg6rAc+XVAsFL8vE4fiSf
J2iisx0zoDD1ndssg1j6uIs=
-----END CERTIFICATE-----
`

const csrPEM = `-----BEGIN CERTIFICATE REQUEST-----
MIIBQTCB5wIBADAjMQswCQYDVQQKEwJOMjEUMBIGA1UEAxMLY3NyLmV4YW1wbGUw
WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT4Fjh9/tnMo75kGkovALFcLI43ZeEF
stmrNTxdKrNqWP9+IlNtfHEb6Jux/yVAVv73E2P1DmtQdy4K7isaQ4HroGIwYAYJ
KoZIhvcNAQkOMVMwUTBPBgNVHREESDBGggtjc3IuZXhhbXBsZYILYWx0LmV4YW1w
bGWBDWFAZXhhbXBsZS5jb22HBMAAAgeGFWh0dHBzOi8vY3NyLmV4YW1wbGUvcDAK
BggqhkjOPQQDAgNJADBGAiEAj0/ZAYuz7HPXQ/0TZ+4xmZtUN6Dv0j95zGNbnF2t
HvECIQDTfMipFcq6hyI6pmVJjJcMwJRwSJBBCDT3G6e9b4nxNg==
-----END CERTIFICATE REQUEST-----
`

const crlPEM = `-----BEGIN X509 CRL-----
MIIBDzCBtwIBATAKBggqhkjOPQQDAjAfMQswCQYDVQQKEwJOMjEQMA4GA1UEAxMH
bjIgcm9vdBcNMjQwMTAyMDMwNDA1WhcNNDQwMTAyMDMwNDA1WjA2MCACAQIXDTI0
MDEwMjAzMDQwNVowDDAKBgNVHRUEAwoBATASAgEDFw0yNDAxMDIwMzA0MDVaoC8w
LTAfBgNVHSMEGDAWgBRLc+klcxDm3d/rpTsg8emjv9L+KzAKBgNVHRQEAwIBBzAK
BggqhkjOPQQDAgNHADBEAiAw4ORR3yrvFoMvyIdmfbbTU0xK2RlaAmGRxBllJ15Q
JgIgazhF4nHzNGJSUsAfEZMluTPi8bC9s93oMPfBgIqeBVI=
-----END X509 CRL-----
`
