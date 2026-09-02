// Command x509use parses 2 certificates that it holds as a PEM constant, and
// it writes each field that the surface of crypto/x509 gives.
package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// caPEM holds a self signed CA certificate with an ECDSA P-256 key.
const caPEM = `-----BEGIN CERTIFICATE-----
MIIDRjCCAuygAwIBAgION1XaNodVpPsxY+AUPMEwCgYIKoZIzj0EAwIwdTELMAkG
A1UEBhMCREUxDzANBgNVBAgTBkJlcmxpbjEPMA0GA1UEBxMGQmVybGluMRUwEwYD
VQQKEwxHYXJ0IFByb2plY3QxETAPBgNVBAsTCENvbXBpbGVyMRowGAYDVQQDExFH
YXJ0IFRlc3QgUm9vdCBDQTAeFw0yNDAzMDExMDMwMDBaFw0zNDAzMDExMDMwMDBa
MHUxCzAJBgNVBAYTAkRFMQ8wDQYDVQQIEwZCZXJsaW4xDzANBgNVBAcTBkJlcmxp
bjEVMBMGA1UEChMMR2FydCBQcm9qZWN0MREwDwYDVQQLEwhDb21waWxlcjEaMBgG
A1UEAxMRR2FydCBUZXN0IFJvb3QgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNC
AARvRfKOUfGfUZiOU8ECn8K/qQFAsG+M/IJgFk9DwFFQyaUE/ovc5z7LH+VN45DW
UTz7xQUW2kli/wlZx4Ryi8aDo4IBXjCCAVowDgYDVR0PAQH/BAQDAgGGMB0GA1Ud
JQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjASBgNVHRMBAf8ECDAGAQH/AgECMBEG
A1UdDgQKBAgBAgMEBQYHCDBdBggrBgEFBQcBAQRRME8wIwYIKwYBBQUHMAGGF2h0
dHA6Ly9vY3NwLmV4YW1wbGUuY29tMCgGCCsGAQUFBzAChhxodHRwOi8vY2EuZXhh
bXBsZS5jb20vY2EuY3J0MHEGA1UdEQRqMGiCDmNhLmV4YW1wbGUuY29tghAqLmNh
LmV4YW1wbGUuY29tgQ9wa2lAZXhhbXBsZS5jb22HBMAAAgqHECABDbgAAAAAAAAA
AAAAAAGGG2h0dHBzOi8vY2EuZXhhbXBsZS5jb20vcm9vdDAwBgNVHR8EKTAnMCWg
I6Ahhh9odHRwOi8vY3JsLmV4YW1wbGUuY29tL3Jvb3QuY3JsMAoGCCqGSM49BAMC
A0gAMEUCIQCKLbMYnsnnHbUKE4Po9wPzSKhyy0hDDAJruNMFJfhWSgIgTCRwNaVM
tWgrpwHnyPOOYjtjJzUzVU7ondeM3jnvYAI=
-----END CERTIFICATE-----
`

// leafPEM holds a leaf certificate with an RSA key, signed by the CA above.
const leafPEM = `-----BEGIN CERTIFICATE-----
MIIC5DCCAoqgAwIBAgICEAAwCgYIKoZIzj0EAwIwdTELMAkGA1UEBhMCREUxDzAN
BgNVBAgTBkJlcmxpbjEPMA0GA1UEBxMGQmVybGluMRUwEwYDVQQKEwxHYXJ0IFBy
b2plY3QxETAPBgNVBAsTCENvbXBpbGVyMRowGAYDVQQDExFHYXJ0IFRlc3QgUm9v
dCBDQTAeFw0yNDA2MTUwODAwMDBaFw0yNjA2MTUwODAwMDBaMDIxFTATBgNVBAoT
DEdhcnQgUHJvamVjdDEZMBcGA1UEAxMQbGVhZi5leGFtcGxlLmNvbTCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBAKhI0d3nBebrmM4MlBj3SWMJ3L5w5Fsa
n9o3sTjv7uCfW7nJ4ivwQ6Q0zQHz0EYWit2GDP+GMlJayLMLR/37kJa6FH0pFSJo
9yABaSDp5CrjvCYQA7J6EkEGHwffXDtIDPyWv0Qv2FtU9YkDuE2yfPdBiV3EBJg5
0SG5Pkxx74blaytzzwehopLrSaE5eMUrXbA+V3d56xQrWVyL49SChiMRExauBJge
bV77WswAblBvAkIZwO/o02N39dv10HT7AgwGlXPs1Ms05H15Tt8UaqW3kDNAQYVS
hKshqfmfywF32H7YK9nPd4Ca1ciDL4Cu8xu49jW4m0BYMs2tA5uqezECAwEAAaOB
gTB/MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMB
Af8EAjAAMBMGA1UdIwQMMAqACAECAwQFBgcIMDUGA1UdEQQuMCyCEGxlYWYuZXhh
bXBsZS5jb22CEioubGVhZi5leGFtcGxlLmNvbYcEfwAAATAKBggqhkjOPQQDAgNI
ADBFAiAkNlcYBgVVRZtV6LgGJxElC0xRKuT8NQdFBRPD8A9KogIhAKzFOp5QuiXF
kiBwV7j6DeEi6FChFukoCqpbxTvb45GU
-----END CERTIFICATE-----
`

// caKeyPEM holds the private key of the CA certificate above. The key signs
// nothing here, and tls.X509KeyPair needs it.
const caKeyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgTQOtksGTtsKGyEZv
Cau2q/sQAvTutqqIrpwPwJNldpKhRANCAARvRfKOUfGfUZiOU8ECn8K/qQFAsG+M
/IJgFk9DwFFQyaUE/ovc5z7LH+VN45DWUTz7xQUW2kli/wlZx4Ryi8aD
-----END PRIVATE KEY-----
`

func main() {
	caDER := decode(caPEM)
	leafDER := decode(leafPEM)

	ca, err := x509.ParseCertificate(caDER)
	fmt.Println("parse ca err:", err)
	leaf, err := x509.ParseCertificate(leafDER)
	fmt.Println("parse leaf err:", err)

	report("ca", ca)
	report("leaf", leaf)

	fmt.Println()
	fmt.Println("== the chain ==")
	both := append(append([]byte{}, caDER...), leafDER...)
	chain, err := x509.ParseCertificates(both)
	fmt.Println("count:", len(chain), "err:", err)
	for _, c := range chain {
		fmt.Println("  ", c.Subject.CommonName)
	}

	fmt.Println()
	fmt.Println("== the round trip ==")
	back := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	fmt.Println("pem equal:", string(back) == caPEM)
	again, err := x509.ParseCertificate(ca.Raw)
	fmt.Println("reparse err:", err, "equal:", ca.Equal(again))
	fmt.Println("ca equal leaf:", ca.Equal(leaf))

	fmt.Println()
	fmt.Println("== the pool ==")
	pool := x509.NewCertPool()
	fmt.Println("append pem:", pool.AppendCertsFromPEM([]byte(caPEM+leafPEM)))
	fmt.Println("subjects:", len(pool.Subjects()))
	pool.AddCert(ca)
	fmt.Println("after a second add:", len(pool.Subjects()))
	other := x509.NewCertPool()
	other.AddCert(ca)
	other.AddCert(leaf)
	fmt.Println("equal:", pool.Equal(other))
	clone := pool.Clone()
	fmt.Println("clone equal:", pool.Equal(clone))
	empty := x509.NewCertPool()
	fmt.Println("empty equal:", pool.Equal(empty), empty.Equal(x509.NewCertPool()))
	fmt.Println("append junk:", empty.AppendCertsFromPEM([]byte("not a pem block")))

	fmt.Println()
	fmt.Println("== the host check ==")
	for _, host := range []string{
		"leaf.example.com", "LEAF.EXAMPLE.COM", "a.leaf.example.com",
		"a.b.leaf.example.com", "other.example.com", "127.0.0.1",
		"[127.0.0.1]", "192.0.2.10",
	} {
		err := leaf.VerifyHostname(host)
		if err == nil {
			fmt.Printf("  %-22s ok\n", host)
			continue
		}
		fmt.Printf("  %-22s %v\n", host, err)
	}
	fmt.Println("  ca 192.0.2.10:", ca.VerifyHostname("192.0.2.10"))
	fmt.Println("  ca 2001:db8::1:", ca.VerifyHostname("[2001:db8::1]"))

	fmt.Println()
	fmt.Println("== the tls key pair ==")
	pair, err := tls.X509KeyPair([]byte(caPEM), []byte(caKeyPEM))
	fmt.Println("x509 key pair err:", err)
	fmt.Println("chain length:", len(pair.Certificate))
	fmt.Println("leaf der equal:", bytes.Equal(pair.Certificate[0], ca.Raw))
	fmt.Println("leaf subject:", pair.Leaf.Subject.CommonName)
	fmt.Println("leaf serial:", pair.Leaf.SerialNumber)
	fmt.Println("leaf equal ca:", pair.Leaf.Equal(ca))
	_, err = tls.X509KeyPair([]byte("no pem here"), []byte(caKeyPEM))
	fmt.Println("no certificate:", err)
	_, err = tls.X509KeyPair([]byte(caPEM), []byte("no pem here"))
	fmt.Println("no key:", err)

	// The text of a parse error differs from the text of Go, because Go reads
	// the DER with cryptobyte and the overlay reads it with encoding/asn1.
	// The lines below thus report the presence of the error alone.
	fmt.Println()
	fmt.Println("== a broken input ==")
	_, err = x509.ParseCertificate([]byte{0x30, 0x03, 0x02, 0x01, 0x05})
	fmt.Println("short der rejected:", err != nil)
	_, err = x509.ParseCertificate(append(append([]byte{}, caDER...), 0x00))
	fmt.Println("trailing rejected:", err != nil)
	_, err = x509.ParseCertificates([]byte{0x01, 0x02})
	fmt.Println("junk rejected:", err != nil)
}

// decode gives the DER of the first PEM block of the text.
func decode(text string) []byte {
	block, rest := pem.Decode([]byte(text))
	if block == nil {
		panic("no PEM block")
	}
	if len(rest) != 0 {
		panic("more than 1 PEM block")
	}
	if block.Type != "CERTIFICATE" {
		panic("the block is no certificate")
	}
	return block.Bytes
}

// report writes each field of a certificate that the surface gives.
func report(name string, c *x509.Certificate) {
	fmt.Println()
	fmt.Println("==", name, "==")
	fmt.Println("version:", c.Version)
	fmt.Println("serial:", c.SerialNumber)
	fmt.Println("subject:", c.Subject)
	fmt.Println("subject cn:", c.Subject.CommonName)
	fmt.Println("subject o:", c.Subject.Organization)
	fmt.Println("subject ou:", c.Subject.OrganizationalUnit)
	fmt.Println("subject c:", c.Subject.Country)
	fmt.Println("subject st:", c.Subject.Province)
	fmt.Println("subject l:", c.Subject.Locality)
	fmt.Println("issuer:", c.Issuer)
	fmt.Println("issuer cn:", c.Issuer.CommonName)
	fmt.Println("not before:", c.NotBefore.UTC().Format(time.RFC3339))
	fmt.Println("not after:", c.NotAfter.UTC().Format(time.RFC3339))
	fmt.Println("signature algorithm:", c.SignatureAlgorithm)
	fmt.Println("public key algorithm:", c.PublicKeyAlgorithm)
	fmt.Println("key usage:", keyUsage(c.KeyUsage))
	fmt.Println("ext key usage:", extKeyUsage(c.ExtKeyUsage))
	fmt.Println("unknown ext key usage:", len(c.UnknownExtKeyUsage))
	fmt.Println("basic constraints valid:", c.BasicConstraintsValid)
	fmt.Println("is ca:", c.IsCA)
	fmt.Println("max path len:", c.MaxPathLen, "zero:", c.MaxPathLenZero)
	fmt.Println("subject key id:", hex.EncodeToString(c.SubjectKeyId))
	fmt.Println("authority key id:", hex.EncodeToString(c.AuthorityKeyId))
	fmt.Println("dns names:", c.DNSNames)
	fmt.Println("email addresses:", c.EmailAddresses)
	fmt.Println("ip addresses:", ips(c))
	fmt.Println("uris:", uris(c))
	fmt.Println("ocsp server:", c.OCSPServer)
	fmt.Println("issuing certificate url:", c.IssuingCertificateURL)
	fmt.Println("crl distribution points:", c.CRLDistributionPoints)
	fmt.Println("raw len:", len(c.Raw))
	fmt.Println("raw tbs len:", len(c.RawTBSCertificate))
	fmt.Println("raw subject len:", len(c.RawSubject))
	fmt.Println("raw issuer len:", len(c.RawIssuer))
	fmt.Println("raw spki len:", len(c.RawSubjectPublicKeyInfo))
	fmt.Println("raw signature algorithm len:", len(c.RawSignatureAlgorithm))
	fmt.Println("signature len:", len(c.Signature))
	sum := sha256.Sum256(c.Raw)
	fmt.Println("raw sha256:", hex.EncodeToString(sum[:]))
	sum = sha256.Sum256(c.RawSubjectPublicKeyInfo)
	fmt.Println("spki sha256:", hex.EncodeToString(sum[:]))
	fmt.Println("extensions:")
	for _, e := range c.Extensions {
		fmt.Printf("  %-24s critical=%-5v len=%d\n", e.Id, e.Critical, len(e.Value))
	}
	fmt.Println("unhandled critical:", len(c.UnhandledCriticalExtensions))
}

// keyUsage names each bit that the key usage holds.
func keyUsage(u x509.KeyUsage) string {
	all := []x509.KeyUsage{
		x509.KeyUsageDigitalSignature,
		x509.KeyUsageContentCommitment,
		x509.KeyUsageKeyEncipherment,
		x509.KeyUsageDataEncipherment,
		x509.KeyUsageKeyAgreement,
		x509.KeyUsageCertSign,
		x509.KeyUsageCRLSign,
		x509.KeyUsageEncipherOnly,
		x509.KeyUsageDecipherOnly,
	}
	var out []string
	for _, bit := range all {
		if u&bit != 0 {
			out = append(out, bit.String())
		}
	}
	return strings.Join(out, "|")
}

// extKeyUsage names each extended key usage.
func extKeyUsage(list []x509.ExtKeyUsage) string {
	var out []string
	for _, u := range list {
		out = append(out, u.String())
	}
	return strings.Join(out, "|")
}

// ips names each IP address of the certificate.
func ips(c *x509.Certificate) []string {
	var out []string
	for _, ip := range c.IPAddresses {
		out = append(out, ip.String())
	}
	return out
}

// uris names each URI of the certificate.
func uris(c *x509.Certificate) []string {
	var out []string
	for _, u := range c.URIs {
		out = append(out, u.String())
	}
	return out
}
