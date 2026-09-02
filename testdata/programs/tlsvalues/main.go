// Command tlsvalues prints the value surface of crypto/tls. The output of the
// Go toolchain and the output of the Dart build must stay equal, thus a wrong
// constant of the overlay shows up as a difference.
package main

import (
	"crypto/tls"
	"fmt"
)

type suite struct {
	name string
	id   uint16
}

var suites = []suite{
	{"TLS_RSA_WITH_RC4_128_SHA", tls.TLS_RSA_WITH_RC4_128_SHA},
	{"TLS_RSA_WITH_3DES_EDE_CBC_SHA", tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA},
	{"TLS_RSA_WITH_AES_128_CBC_SHA", tls.TLS_RSA_WITH_AES_128_CBC_SHA},
	{"TLS_RSA_WITH_AES_256_CBC_SHA", tls.TLS_RSA_WITH_AES_256_CBC_SHA},
	{"TLS_RSA_WITH_AES_128_CBC_SHA256", tls.TLS_RSA_WITH_AES_128_CBC_SHA256},
	{"TLS_RSA_WITH_AES_128_GCM_SHA256", tls.TLS_RSA_WITH_AES_128_GCM_SHA256},
	{"TLS_RSA_WITH_AES_256_GCM_SHA384", tls.TLS_RSA_WITH_AES_256_GCM_SHA384},
	{"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA", tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA},
	{"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA", tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA},
	{"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA},
	{"TLS_ECDHE_RSA_WITH_RC4_128_SHA", tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA},
	{"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA", tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA},
	{"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA", tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA},
	{"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA", tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA},
	{"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256", tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256},
	{"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256", tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256},
	{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	{"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	{"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
	{"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
	{"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256", tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256},
	{"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256", tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256},
	{"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305", tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305},
	{"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305", tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305},
	{"TLS_AES_128_GCM_SHA256", tls.TLS_AES_128_GCM_SHA256},
	{"TLS_AES_256_GCM_SHA384", tls.TLS_AES_256_GCM_SHA384},
	{"TLS_CHACHA20_POLY1305_SHA256", tls.TLS_CHACHA20_POLY1305_SHA256},
	{"TLS_FALLBACK_SCSV", tls.TLS_FALLBACK_SCSV},
}

var versions = []uint16{
	tls.VersionSSL30,
	tls.VersionTLS10,
	tls.VersionTLS11,
	tls.VersionTLS12,
	tls.VersionTLS13,
	0x0000,
	0x0305,
	0xabcd,
	0xffff,
}

var curves = []tls.CurveID{
	tls.CurveP256,
	tls.CurveP384,
	tls.CurveP521,
	tls.X25519,
	tls.MLKEM1024,
	tls.SecP256r1MLKEM768,
	tls.X25519MLKEM768,
	tls.SecP384r1MLKEM1024,
	tls.CurveID(0),
	tls.CurveID(26),
	tls.CurveID(4590),
	tls.CurveID(65535),
}

var schemes = []tls.SignatureScheme{
	tls.PKCS1WithSHA256,
	tls.PKCS1WithSHA384,
	tls.PKCS1WithSHA512,
	tls.PSSWithSHA256,
	tls.PSSWithSHA384,
	tls.PSSWithSHA512,
	tls.ECDSAWithP256AndSHA256,
	tls.ECDSAWithP384AndSHA384,
	tls.ECDSAWithP521AndSHA512,
	tls.Ed25519,
	tls.MLDSA44,
	tls.MLDSA65,
	tls.MLDSA87,
	tls.PKCS1WithSHA1,
	tls.ECDSAWithSHA1,
	tls.SignatureScheme(0),
	tls.SignatureScheme(2051),
	tls.SignatureScheme(2056),
	tls.SignatureScheme(2311),
	tls.SignatureScheme(65535),
}

var auths = []tls.ClientAuthType{
	tls.NoClientCert,
	tls.RequestClientCert,
	tls.RequireAnyClientCert,
	tls.VerifyClientCertIfGiven,
	tls.RequireAndVerifyClientCert,
	tls.ClientAuthType(5),
	tls.ClientAuthType(-1),
}

var levels = []tls.QUICEncryptionLevel{
	tls.QUICEncryptionLevelInitial,
	tls.QUICEncryptionLevelEarly,
	tls.QUICEncryptionLevelHandshake,
	tls.QUICEncryptionLevelApplication,
	tls.QUICEncryptionLevel(4),
	tls.QUICEncryptionLevel(-1),
}

func main() {
	fmt.Println("-- cipher suite identifiers --")
	for _, s := range suites {
		fmt.Printf("%s %d %s\n", s.name, s.id, tls.CipherSuiteName(s.id))
	}
	fmt.Printf("unknown %d %s\n", 0x0001, tls.CipherSuiteName(0x0001))
	fmt.Printf("unknown %d %s\n", 0xdead, tls.CipherSuiteName(0xdead))

	fmt.Println("-- versions --")
	for _, v := range versions {
		fmt.Printf("%d %s\n", v, tls.VersionName(v))
	}

	fmt.Println("-- curves --")
	for _, c := range curves {
		fmt.Printf("%d %s\n", uint16(c), c.String())
	}

	fmt.Println("-- signature schemes --")
	for _, s := range schemes {
		fmt.Printf("%d %s\n", uint16(s), s.String())
	}

	fmt.Println("-- client auth types --")
	for _, a := range auths {
		fmt.Printf("%d %s\n", int(a), a.String())
	}

	fmt.Println("-- renegotiation support --")
	fmt.Printf("RenegotiateNever %d\n", int(tls.RenegotiateNever))
	fmt.Printf("RenegotiateOnceAsClient %d\n", int(tls.RenegotiateOnceAsClient))
	fmt.Printf("RenegotiateFreelyAsClient %d\n", int(tls.RenegotiateFreelyAsClient))

	fmt.Println("-- QUIC encryption levels --")
	for _, l := range levels {
		fmt.Printf("%d %s\n", int(l), l.String())
	}

	fmt.Println("-- CipherSuites --")
	report(tls.CipherSuites())
	fmt.Println("-- InsecureCipherSuites --")
	report(tls.InsecureCipherSuites())
}

func report(list []*tls.CipherSuite) {
	fmt.Printf("count %d\n", len(list))
	for _, c := range list {
		fmt.Printf("%d %s insecure=%t versions=", c.ID, c.Name, c.Insecure)
		for i, v := range c.SupportedVersions {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf("%s", tls.VersionName(v))
		}
		fmt.Println()
	}
}
