//go:build dart

// The value surface of the package: each identifier of the wire, each
// enumeration with its name, and each pure data function. Dart holds the
// handshake, thus a program reads such a value to name a suite, a curve, or a
// signature scheme, and never to steer the connection.

package tls

import (
	"strconv"
	"strings"
)

// A list of cipher suite identifiers that Go carries. Dart chooses the suite
// of a connection on its own, thus a value here names a suite only.
const (
	TLS_RSA_WITH_RC4_128_SHA                      uint16 = 0x0005
	TLS_RSA_WITH_3DES_EDE_CBC_SHA                 uint16 = 0x000a
	TLS_RSA_WITH_AES_128_CBC_SHA                  uint16 = 0x002f
	TLS_RSA_WITH_AES_256_CBC_SHA                  uint16 = 0x0035
	TLS_RSA_WITH_AES_128_CBC_SHA256               uint16 = 0x003c
	TLS_RSA_WITH_AES_128_GCM_SHA256               uint16 = 0x009c
	TLS_RSA_WITH_AES_256_GCM_SHA384               uint16 = 0x009d
	TLS_ECDHE_ECDSA_WITH_RC4_128_SHA              uint16 = 0xc007
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA          uint16 = 0xc009
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA          uint16 = 0xc00a
	TLS_ECDHE_RSA_WITH_RC4_128_SHA                uint16 = 0xc011
	TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA           uint16 = 0xc012
	TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA            uint16 = 0xc013
	TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA            uint16 = 0xc014
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256       uint16 = 0xc023
	TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256         uint16 = 0xc027
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256         uint16 = 0xc02f
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256       uint16 = 0xc02b
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384         uint16 = 0xc030
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384       uint16 = 0xc02c
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256   uint16 = 0xcca8
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256 uint16 = 0xcca9

	TLS_AES_128_GCM_SHA256       uint16 = 0x1301
	TLS_AES_256_GCM_SHA384       uint16 = 0x1302
	TLS_CHACHA20_POLY1305_SHA256 uint16 = 0x1303

	// TLS_FALLBACK_SCSV is no suite. It marks a client that falls back to an
	// older version. Read RFC 7507.
	TLS_FALLBACK_SCSV uint16 = 0x5600

	// The 2 names below carry no _SHA256 suffix. Go keeps them for the code
	// that took them before the suffix arrived.
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305   = TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305 = TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
)

// VersionName gives the name of a version number, for example "TLS 1.3". An
// unknown number gives the hexadecimal form.
func VersionName(version uint16) string {
	switch version {
	case VersionSSL30:
		return "SSLv3"
	case VersionTLS10:
		return "TLS 1.0"
	case VersionTLS11:
		return "TLS 1.1"
	case VersionTLS12:
		return "TLS 1.2"
	case VersionTLS13:
		return "TLS 1.3"
	default:
		return hex4(version)
	}
}

// hex4 writes a 16 bit number as the verb %04X of Go writes it. This package
// holds no fmt, thus each text here joins strings.
func hex4(v uint16) string {
	s := strings.ToUpper(strconv.FormatUint(uint64(v), 16))
	for len(s) < 4 {
		s = "0" + s
	}
	return "0x" + s
}

// CurveID names a key exchange mechanism. The name says curve for a reason of
// history, and the set now holds a post quantum mechanism too.
type CurveID uint16

const (
	CurveP256          CurveID = 23
	CurveP384          CurveID = 24
	CurveP521          CurveID = 25
	X25519             CurveID = 29
	X25519MLKEM768     CurveID = 4588
	SecP256r1MLKEM768  CurveID = 4587
	SecP384r1MLKEM1024 CurveID = 4589
	MLKEM1024          CurveID = 514
)

func (i CurveID) String() string {
	switch i {
	case CurveP256:
		return "CurveP256"
	case CurveP384:
		return "CurveP384"
	case CurveP521:
		return "CurveP521"
	case X25519:
		return "X25519"
	case MLKEM1024:
		return "MLKEM1024"
	case SecP256r1MLKEM768:
		return "SecP256r1MLKEM768"
	case X25519MLKEM768:
		return "X25519MLKEM768"
	case SecP384r1MLKEM1024:
		return "SecP384r1MLKEM1024"
	}
	return "CurveID(" + strconv.FormatInt(int64(i), 10) + ")"
}

// SignatureScheme names a signature algorithm of TLS. Read RFC 8446, section
// 4.2.3.
type SignatureScheme uint16

const (
	PKCS1WithSHA256 SignatureScheme = 0x0401
	PKCS1WithSHA384 SignatureScheme = 0x0501
	PKCS1WithSHA512 SignatureScheme = 0x0601

	PSSWithSHA256 SignatureScheme = 0x0804
	PSSWithSHA384 SignatureScheme = 0x0805
	PSSWithSHA512 SignatureScheme = 0x0806

	ECDSAWithP256AndSHA256 SignatureScheme = 0x0403
	ECDSAWithP384AndSHA384 SignatureScheme = 0x0503
	ECDSAWithP521AndSHA512 SignatureScheme = 0x0603

	Ed25519 SignatureScheme = 0x0807

	MLDSA44 SignatureScheme = 0x0904
	MLDSA65 SignatureScheme = 0x0905
	MLDSA87 SignatureScheme = 0x0906

	// The 2 schemes below belong to TLS 1.2 only.
	PKCS1WithSHA1 SignatureScheme = 0x0201
	ECDSAWithSHA1 SignatureScheme = 0x0203
)

func (i SignatureScheme) String() string {
	switch i {
	case PKCS1WithSHA1:
		return "PKCS1WithSHA1"
	case ECDSAWithSHA1:
		return "ECDSAWithSHA1"
	case PKCS1WithSHA256:
		return "PKCS1WithSHA256"
	case ECDSAWithP256AndSHA256:
		return "ECDSAWithP256AndSHA256"
	case PKCS1WithSHA384:
		return "PKCS1WithSHA384"
	case ECDSAWithP384AndSHA384:
		return "ECDSAWithP384AndSHA384"
	case PKCS1WithSHA512:
		return "PKCS1WithSHA512"
	case ECDSAWithP521AndSHA512:
		return "ECDSAWithP521AndSHA512"
	case PSSWithSHA256:
		return "PSSWithSHA256"
	case PSSWithSHA384:
		return "PSSWithSHA384"
	case PSSWithSHA512:
		return "PSSWithSHA512"
	case Ed25519:
		return "Ed25519"
	case MLDSA44:
		return "MLDSA44"
	case MLDSA65:
		return "MLDSA65"
	case MLDSA87:
		return "MLDSA87"
	}
	return "SignatureScheme(" + strconv.FormatInt(int64(i), 10) + ")"
}

// ClientAuthType names the policy of a server for the authentication of a
// client. Dart holds the handshake, thus no Config field takes such a value.
type ClientAuthType int

const (
	NoClientCert ClientAuthType = iota
	RequestClientCert
	RequireAnyClientCert
	VerifyClientCertIfGiven
	RequireAndVerifyClientCert
)

func (i ClientAuthType) String() string {
	switch i {
	case NoClientCert:
		return "NoClientCert"
	case RequestClientCert:
		return "RequestClientCert"
	case RequireAnyClientCert:
		return "RequireAnyClientCert"
	case VerifyClientCertIfGiven:
		return "VerifyClientCertIfGiven"
	case RequireAndVerifyClientCert:
		return "RequireAndVerifyClientCert"
	}
	return "ClientAuthType(" + strconv.FormatInt(int64(i), 10) + ")"
}

// RenegotiationSupport names which renegotiation a client accepts. Dart holds
// the handshake, thus no Config field takes such a value.
type RenegotiationSupport int

const (
	RenegotiateNever RenegotiationSupport = iota
	RenegotiateOnceAsClient
	RenegotiateFreelyAsClient
)

// CipherSuite describes 1 cipher suite of TLS.
type CipherSuite struct {
	ID   uint16
	Name string

	// SupportedVersions lists each version of the protocol that can take the
	// suite.
	SupportedVersions []uint16

	// Insecure marks a suite with a known weakness.
	Insecure bool
}

var (
	supportedUpToTLS12 = []uint16{VersionTLS10, VersionTLS11, VersionTLS12}
	supportedOnlyTLS12 = []uint16{VersionTLS12}
	supportedOnlyTLS13 = []uint16{VersionTLS13}
)

// CipherSuites lists each suite of Go that carries no known weakness, in the
// order of the identifier. InsecureCipherSuites lists the rest.
func CipherSuites() []*CipherSuite {
	return []*CipherSuite{
		{TLS_AES_128_GCM_SHA256, "TLS_AES_128_GCM_SHA256", supportedOnlyTLS13, false},
		{TLS_AES_256_GCM_SHA384, "TLS_AES_256_GCM_SHA384", supportedOnlyTLS13, false},
		{TLS_CHACHA20_POLY1305_SHA256, "TLS_CHACHA20_POLY1305_SHA256", supportedOnlyTLS13, false},

		{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},
		{TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256", supportedOnlyTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256", supportedOnlyTLS12, false},
	}
}

// InsecureCipherSuites lists each suite of Go that carries a known weakness.
// Most programs take the list of CipherSuites instead.
func InsecureCipherSuites() []*CipherSuite {
	return []*CipherSuite{
		{TLS_RSA_WITH_RC4_128_SHA, "TLS_RSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_AES_128_CBC_SHA, "TLS_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_AES_256_CBC_SHA, "TLS_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_AES_128_CBC_SHA256, "TLS_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_AES_128_GCM_SHA256, "TLS_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_AES_256_GCM_SHA384, "TLS_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_RSA_WITH_RC4_128_SHA, "TLS_ECDHE_RSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
	}
}

// CipherSuiteName gives the standard name of a suite identifier. An unknown
// identifier gives the hexadecimal form.
func CipherSuiteName(id uint16) string {
	for _, c := range CipherSuites() {
		if c.ID == id {
			return c.Name
		}
	}
	for _, c := range InsecureCipherSuites() {
		if c.ID == id {
			return c.Name
		}
	}
	return hex4(id)
}

// QUICEncryptionLevel names the level of encryption of a handshake message of
// QUIC. The surface holds the value only, because the host gives no QUIC
// connection.
type QUICEncryptionLevel int

const (
	QUICEncryptionLevelInitial = QUICEncryptionLevel(iota)
	QUICEncryptionLevelEarly
	QUICEncryptionLevelHandshake
	QUICEncryptionLevelApplication
)

func (l QUICEncryptionLevel) String() string {
	switch l {
	case QUICEncryptionLevelInitial:
		return "Initial"
	case QUICEncryptionLevelEarly:
		return "Early"
	case QUICEncryptionLevelHandshake:
		return "Handshake"
	case QUICEncryptionLevelApplication:
		return "Application"
	}
	return "QUICEncryptionLevel(" + strconv.Itoa(int(l)) + ")"
}
