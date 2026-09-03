//go:build dart

// Package x509 reads an X.509 certificate.
//
// The overlay holds the parse path alone. It reads a DER certificate or a PEM
// certificate, it gives each field of the certificate that the parse
// understands, and it holds a pool of certificates.
//
// The overlay holds no chain builder, no trust store, and no signature check.
// The secure socket of Dart owns that work, thus Certificate.Verify,
// CheckSignature, and SystemCertPool stay out. A name that stays out gives a
// compile error, and no name of this package gives an answer that it cannot
// prove.
//
// The overlay also holds no PublicKey field, because crypto/rsa, crypto/ecdsa,
// and crypto/ed25519 stay outside the surface. PublicKeyAlgorithm names the
// algorithm, and RawSubjectPublicKeyInfo holds the DER of the key.
package x509

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"crypto/x509/pkix"
	"encoding/asn1"
)

// The ASN.1 shape of a certificate. RFC 5280, 4.1.

type certificate struct {
	Raw                asn1.RawContent
	TBSCertificate     tbsCertificate
	SignatureAlgorithm rawAlgorithmIdentifier
	SignatureValue     asn1.BitString
}

type tbsCertificate struct {
	Raw                asn1.RawContent
	Version            int `asn1:"optional,explicit,default:0,tag:0"`
	SerialNumber       *big.Int
	SignatureAlgorithm pkix.AlgorithmIdentifier
	Issuer             asn1.RawValue
	Validity           validity
	Subject            asn1.RawValue
	PublicKey          publicKeyInfo
	UniqueId           asn1.BitString   `asn1:"optional,tag:1"`
	SubjectUniqueId    asn1.BitString   `asn1:"optional,tag:2"`
	Extensions         []pkix.Extension `asn1:"omitempty,optional,explicit,tag:3"`
}

// rawAlgorithmIdentifier holds the fields of pkix.AlgorithmIdentifier and the
// DER of the value. Certificate.RawSignatureAlgorithm reads that DER, and
// pkix holds no such field.
type rawAlgorithmIdentifier struct {
	Raw        asn1.RawContent
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

// ident gives the pkix form of the identifier. A conversion between 2 struct
// types stays outside the subset, thus the copy names each field.
func (a rawAlgorithmIdentifier) ident() pkix.AlgorithmIdentifier {
	return pkix.AlgorithmIdentifier{Algorithm: a.Algorithm, Parameters: a.Parameters}
}

type validity struct {
	NotBefore, NotAfter time.Time
}

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

// authKeyId holds the shape of the authority key identifier. RFC 5280, 4.2.1.1.
type authKeyId struct {
	Id []byte `asn1:"optional,tag:0"`
}

// basicConstraints holds the shape of the basic constraints. RFC 5280,
// 4.2.1.9.
type basicConstraints struct {
	IsCA       bool `asn1:"optional"`
	MaxPathLen int  `asn1:"optional,default:-1"`
}

// distributionPoint holds the shape of a CRL distribution point. RFC 5280,
// 4.2.1.13.
type distributionPoint struct {
	DistributionPoint distributionPointName `asn1:"optional,tag:0"`
	Reason            asn1.BitString        `asn1:"optional,tag:1"`
	CRLIssuer         asn1.RawValue         `asn1:"optional,tag:2"`
}

type distributionPointName struct {
	FullName     asn1.RawValue    `asn1:"optional,tag:0"`
	RelativeName pkix.RDNSequence `asn1:"optional,tag:1"`
}

// authorityInfoAccess holds 1 access description. RFC 5280, 4.2.2.1.
type authorityInfoAccess struct {
	Method   asn1.ObjectIdentifier
	Location asn1.RawValue
}

// pssParameters holds the parameters of an RSA PSS signature. RFC 3447,
// Appendix A.2.3.
type pssParameters struct {
	Hash         pkix.AlgorithmIdentifier `asn1:"explicit,tag:0"`
	MGF          pkix.AlgorithmIdentifier `asn1:"explicit,tag:1"`
	SaltLength   int                      `asn1:"explicit,tag:2"`
	TrailerField int                      `asn1:"optional,explicit,tag:3,default:1"`
}

// A SignatureAlgorithm names the algorithm of the signature of a certificate.
type SignatureAlgorithm int

const (
	UnknownSignatureAlgorithm SignatureAlgorithm = iota

	MD2WithRSA  // Unsupported.
	MD5WithRSA  // Only supported for signing, not verification.
	SHA1WithRSA // Only supported for signing, and verification of CRLs, CSRs, and OCSP responses.
	SHA256WithRSA
	SHA384WithRSA
	SHA512WithRSA
	DSAWithSHA1   // Unsupported.
	DSAWithSHA256 // Unsupported.
	ECDSAWithSHA1 // Only supported for signing, and verification of CRLs, CSRs, and OCSP responses.
	ECDSAWithSHA256
	ECDSAWithSHA384
	ECDSAWithSHA512
	SHA256WithRSAPSS
	SHA384WithRSAPSS
	SHA512WithRSAPSS
	PureEd25519
	MLDSA44
	MLDSA65
	MLDSA87
)

// A PublicKeyAlgorithm names the algorithm of the public key of a certificate.
type PublicKeyAlgorithm int

const (
	UnknownPublicKeyAlgorithm PublicKeyAlgorithm = iota
	RSA
	DSA // Only supported for parsing.
	ECDSA
	Ed25519
	MLDSA
)

var publicKeyAlgoName = [...]string{
	RSA:     "RSA",
	DSA:     "DSA",
	ECDSA:   "ECDSA",
	Ed25519: "Ed25519",
	MLDSA:   "ML-DSA",
}

// String names the algorithm.
func (algo PublicKeyAlgorithm) String() string {
	if 0 < algo && int(algo) < len(publicKeyAlgoName) {
		return publicKeyAlgoName[algo]
	}
	return strconv.Itoa(int(algo))
}

var (
	oidSignatureMD2WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 2}
	oidSignatureMD5WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 4}
	oidSignatureSHA1WithRSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5}
	oidSignatureSHA256WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidSignatureSHA384WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidSignatureSHA512WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
	oidSignatureRSAPSS          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
	oidSignatureDSAWithSHA1     = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 3}
	oidSignatureDSAWithSHA256   = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 2}
	oidSignatureECDSAWithSHA1   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 1}
	oidSignatureECDSAWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidSignatureECDSAWithSHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	oidSignatureECDSAWithSHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
	oidSignatureEd25519         = asn1.ObjectIdentifier{1, 3, 101, 112}

	oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidSHA384 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	oidSHA512 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}

	oidMGF1 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 8}

	// oidISOSignatureSHA1WithRSA names the same algorithm as
	// oidSignatureSHA1WithRSA. Some certificates hold it.
	oidISOSignatureSHA1WithRSA = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 29}

	oidPublicKeyRSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidPublicKeyDSA     = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 1}
	oidPublicKeyECDSA   = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	oidPublicKeyEd25519 = asn1.ObjectIdentifier{1, 3, 101, 112}

	oidPublicKeyMLDSA44 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 17}
	oidPublicKeyMLDSA65 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 18}
	oidPublicKeyMLDSA87 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 19}
)

// signatureAlgorithmDetails holds the name and the object identifier of each
// algorithm. The table of the real package also holds the hash function, and
// the surface holds no package crypto, thus this table drops that column.
var signatureAlgorithmDetails = []struct {
	algo     SignatureAlgorithm
	name     string
	oid      asn1.ObjectIdentifier
	isRSAPSS bool
}{
	{MD2WithRSA, "MD2-RSA", oidSignatureMD2WithRSA, false},
	{MD5WithRSA, "MD5-RSA", oidSignatureMD5WithRSA, false},
	{SHA1WithRSA, "SHA1-RSA", oidSignatureSHA1WithRSA, false},
	{SHA1WithRSA, "SHA1-RSA", oidISOSignatureSHA1WithRSA, false},
	{SHA256WithRSA, "SHA256-RSA", oidSignatureSHA256WithRSA, false},
	{SHA384WithRSA, "SHA384-RSA", oidSignatureSHA384WithRSA, false},
	{SHA512WithRSA, "SHA512-RSA", oidSignatureSHA512WithRSA, false},
	{SHA256WithRSAPSS, "SHA256-RSAPSS", oidSignatureRSAPSS, true},
	{SHA384WithRSAPSS, "SHA384-RSAPSS", oidSignatureRSAPSS, true},
	{SHA512WithRSAPSS, "SHA512-RSAPSS", oidSignatureRSAPSS, true},
	{DSAWithSHA1, "DSA-SHA1", oidSignatureDSAWithSHA1, false},
	{DSAWithSHA256, "DSA-SHA256", oidSignatureDSAWithSHA256, false},
	{ECDSAWithSHA1, "ECDSA-SHA1", oidSignatureECDSAWithSHA1, false},
	{ECDSAWithSHA256, "ECDSA-SHA256", oidSignatureECDSAWithSHA256, false},
	{ECDSAWithSHA384, "ECDSA-SHA384", oidSignatureECDSAWithSHA384, false},
	{ECDSAWithSHA512, "ECDSA-SHA512", oidSignatureECDSAWithSHA512, false},
	{PureEd25519, "Ed25519", oidSignatureEd25519, false},
	{MLDSA44, "ML-DSA-44", oidPublicKeyMLDSA44, false},
	{MLDSA65, "ML-DSA-65", oidPublicKeyMLDSA65, false},
	{MLDSA87, "ML-DSA-87", oidPublicKeyMLDSA87, false},
}

// String names the algorithm.
func (algo SignatureAlgorithm) String() string {
	for _, details := range signatureAlgorithmDetails {
		if details.algo == algo {
			return details.name
		}
	}
	return strconv.Itoa(int(algo))
}

// isRSAPSS reports whether the algorithm is an RSA PSS algorithm.
func (algo SignatureAlgorithm) isRSAPSS() bool {
	for _, details := range signatureAlgorithmDetails {
		if details.algo == algo {
			return details.isRSAPSS
		}
	}
	return false
}

func getSignatureAlgorithmFromAI(ai pkix.AlgorithmIdentifier) SignatureAlgorithm {
	if ai.Algorithm.Equal(oidSignatureEd25519) ||
		ai.Algorithm.Equal(oidPublicKeyMLDSA44) ||
		ai.Algorithm.Equal(oidPublicKeyMLDSA65) ||
		ai.Algorithm.Equal(oidPublicKeyMLDSA87) {
		// RFC 8410, Section 3, and RFC 9881, Section 2, each name a parameter
		// field that stays absent.
		if len(ai.Parameters.FullBytes) != 0 {
			return UnknownSignatureAlgorithm
		}
	}

	if !ai.Algorithm.Equal(oidSignatureRSAPSS) {
		for _, details := range signatureAlgorithmDetails {
			if ai.Algorithm.Equal(details.oid) {
				return details.algo
			}
		}
		return UnknownSignatureAlgorithm
	}

	// RSA PSS holds the hash and the salt length in the parameters.
	var params pssParameters
	if _, err := asn1.Unmarshal(ai.Parameters.FullBytes, &params); err != nil {
		return UnknownSignatureAlgorithm
	}
	var mgf1HashFunc pkix.AlgorithmIdentifier
	if _, err := asn1.Unmarshal(params.MGF.Parameters.FullBytes, &mgf1HashFunc); err != nil {
		return UnknownSignatureAlgorithm
	}

	// The 3 buckets below need the hash of MGF1 to match the hash of the
	// message, the salt length to match the length of the hash, and the
	// trailer field to hold its default.
	if (len(params.Hash.Parameters.FullBytes) != 0 && !bytes.Equal(params.Hash.Parameters.FullBytes, asn1.NullBytes)) ||
		!params.MGF.Algorithm.Equal(oidMGF1) ||
		!mgf1HashFunc.Algorithm.Equal(params.Hash.Algorithm) ||
		(len(mgf1HashFunc.Parameters.FullBytes) != 0 && !bytes.Equal(mgf1HashFunc.Parameters.FullBytes, asn1.NullBytes)) ||
		params.TrailerField != 1 {
		return UnknownSignatureAlgorithm
	}

	switch {
	case params.Hash.Algorithm.Equal(oidSHA256) && params.SaltLength == 32:
		return SHA256WithRSAPSS
	case params.Hash.Algorithm.Equal(oidSHA384) && params.SaltLength == 48:
		return SHA384WithRSAPSS
	case params.Hash.Algorithm.Equal(oidSHA512) && params.SaltLength == 64:
		return SHA512WithRSAPSS
	}
	return UnknownSignatureAlgorithm
}

func getPublicKeyAlgorithmFromOID(oid asn1.ObjectIdentifier) PublicKeyAlgorithm {
	switch {
	case oid.Equal(oidPublicKeyRSA):
		return RSA
	case oid.Equal(oidPublicKeyDSA):
		return DSA
	case oid.Equal(oidPublicKeyECDSA):
		return ECDSA
	case oid.Equal(oidPublicKeyEd25519):
		return Ed25519
	case oid.Equal(oidPublicKeyMLDSA44), oid.Equal(oidPublicKeyMLDSA65),
		oid.Equal(oidPublicKeyMLDSA87):
		return MLDSA
	}
	return UnknownPublicKeyAlgorithm
}

// A KeyUsage holds the set of operations that the key of a certificate allows.
// It is a bit map of the KeyUsage constants.
type KeyUsage int

const (
	KeyUsageDigitalSignature KeyUsage = 1 << iota
	KeyUsageContentCommitment
	KeyUsageKeyEncipherment
	KeyUsageDataEncipherment
	KeyUsageKeyAgreement
	KeyUsageCertSign
	KeyUsageCRLSign
	KeyUsageEncipherOnly
	KeyUsageDecipherOnly
)

// String names 1 bit of the key usage. A value with 2 bits gives the numeral
// form, exactly as in Go.
func (i KeyUsage) String() string {
	switch i {
	case KeyUsageDigitalSignature:
		return "digitalSignature"
	case KeyUsageContentCommitment:
		return "contentCommitment"
	case KeyUsageKeyEncipherment:
		return "keyEncipherment"
	case KeyUsageDataEncipherment:
		return "dataEncipherment"
	case KeyUsageKeyAgreement:
		return "keyAgreement"
	case KeyUsageCertSign:
		return "keyCertSign"
	case KeyUsageCRLSign:
		return "cRLSign"
	case KeyUsageEncipherOnly:
		return "encipherOnly"
	case KeyUsageDecipherOnly:
		return "decipherOnly"
	}
	return "KeyUsage(" + strconv.FormatInt(int64(i), 10) + ")"
}

// An ExtKeyUsage names 1 operation of the extended key usage extension.
type ExtKeyUsage int

const (
	ExtKeyUsageAny ExtKeyUsage = iota
	ExtKeyUsageServerAuth
	ExtKeyUsageClientAuth
	ExtKeyUsageCodeSigning
	ExtKeyUsageEmailProtection
	ExtKeyUsageIPSECEndSystem
	ExtKeyUsageIPSECTunnel
	ExtKeyUsageIPSECUser
	ExtKeyUsageTimeStamping
	ExtKeyUsageOCSPSigning
	ExtKeyUsageMicrosoftServerGatedCrypto
	ExtKeyUsageNetscapeServerGatedCrypto
	ExtKeyUsageMicrosoftCommercialCodeSigning
	ExtKeyUsageMicrosoftKernelCodeSigning
)

const extKeyUsageName = "anyExtendedKeyUsageserverAuthclientAuthcodeSigningemailProtectionipsecEndSystemipsecTunnelipsecUsertimeStampingOCSPSigningmsSGCnsSGCmsCodeCommsKernelCode"

var extKeyUsageIndex = [...]uint8{0, 19, 29, 39, 50, 65, 79, 90, 99, 111, 122, 127, 132, 141, 153}

// String names the extended key usage.
func (i ExtKeyUsage) String() string {
	idx := int(i)
	if i < 0 || idx >= len(extKeyUsageIndex)-1 {
		return "ExtKeyUsage(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return extKeyUsageName[extKeyUsageIndex[idx]:extKeyUsageIndex[idx+1]]
}

var (
	oidExtKeyUsageAny                            = asn1.ObjectIdentifier{2, 5, 29, 37, 0}
	oidExtKeyUsageServerAuth                     = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 1}
	oidExtKeyUsageClientAuth                     = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 2}
	oidExtKeyUsageCodeSigning                    = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 3}
	oidExtKeyUsageEmailProtection                = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 4}
	oidExtKeyUsageIPSECEndSystem                 = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 5}
	oidExtKeyUsageIPSECTunnel                    = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 6}
	oidExtKeyUsageIPSECUser                      = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 7}
	oidExtKeyUsageTimeStamping                   = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 8}
	oidExtKeyUsageOCSPSigning                    = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 9}
	oidExtKeyUsageMicrosoftServerGatedCrypto     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 10, 3, 3}
	oidExtKeyUsageNetscapeServerGatedCrypto      = asn1.ObjectIdentifier{2, 16, 840, 1, 113730, 4, 1}
	oidExtKeyUsageMicrosoftCommercialCodeSigning = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 1, 22}
	oidExtKeyUsageMicrosoftKernelCodeSigning     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 61, 1, 1}
)

var extKeyUsageOIDs = []struct {
	extKeyUsage ExtKeyUsage
	oid         asn1.ObjectIdentifier
}{
	{ExtKeyUsageAny, oidExtKeyUsageAny},
	{ExtKeyUsageServerAuth, oidExtKeyUsageServerAuth},
	{ExtKeyUsageClientAuth, oidExtKeyUsageClientAuth},
	{ExtKeyUsageCodeSigning, oidExtKeyUsageCodeSigning},
	{ExtKeyUsageEmailProtection, oidExtKeyUsageEmailProtection},
	{ExtKeyUsageIPSECEndSystem, oidExtKeyUsageIPSECEndSystem},
	{ExtKeyUsageIPSECTunnel, oidExtKeyUsageIPSECTunnel},
	{ExtKeyUsageIPSECUser, oidExtKeyUsageIPSECUser},
	{ExtKeyUsageTimeStamping, oidExtKeyUsageTimeStamping},
	{ExtKeyUsageOCSPSigning, oidExtKeyUsageOCSPSigning},
	{ExtKeyUsageMicrosoftServerGatedCrypto, oidExtKeyUsageMicrosoftServerGatedCrypto},
	{ExtKeyUsageNetscapeServerGatedCrypto, oidExtKeyUsageNetscapeServerGatedCrypto},
	{ExtKeyUsageMicrosoftCommercialCodeSigning, oidExtKeyUsageMicrosoftCommercialCodeSigning},
	{ExtKeyUsageMicrosoftKernelCodeSigning, oidExtKeyUsageMicrosoftKernelCodeSigning},
}

// OID gives the object identifier of the extended key usage. It stops the
// program for a value that the list above does not hold.
func (eku ExtKeyUsage) OID() OID {
	for _, pair := range extKeyUsageOIDs {
		if pair.extKeyUsage == eku {
			oid, err := OIDFromASN1OID(pair.oid)
			if err != nil {
				panic("x509: internal error: known ExtKeyUsage has invalid OID")
			}
			return oid
		}
	}
	panic("x509: internal error: known ExtKeyUsage has no OID")
}

func extKeyUsageFromOID(oid asn1.ObjectIdentifier) (eku ExtKeyUsage, ok bool) {
	for _, pair := range extKeyUsageOIDs {
		if oid.Equal(pair.oid) {
			return pair.extKeyUsage, true
		}
	}
	return
}

// The object identifier of each extension that the parse understands.
var (
	oidExtensionSubjectKeyId          = asn1.ObjectIdentifier{2, 5, 29, 14}
	oidExtensionKeyUsage              = asn1.ObjectIdentifier{2, 5, 29, 15}
	oidExtensionSubjectAltName        = asn1.ObjectIdentifier{2, 5, 29, 17}
	oidExtensionBasicConstraints      = asn1.ObjectIdentifier{2, 5, 29, 19}
	oidExtensionCRLDistributionPoints = asn1.ObjectIdentifier{2, 5, 29, 31}
	oidExtensionAuthorityKeyId        = asn1.ObjectIdentifier{2, 5, 29, 35}
	oidExtensionExtendedKeyUsage      = asn1.ObjectIdentifier{2, 5, 29, 37}
	oidExtensionNameConstraints       = asn1.ObjectIdentifier{2, 5, 29, 30}
	oidExtensionAuthorityInfoAccess   = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 1, 1}
	oidExtensionCertificatePolicies   = asn1.ObjectIdentifier{2, 5, 29, 32}
	oidExtensionPolicyMappings        = asn1.ObjectIdentifier{2, 5, 29, 33}
	oidExtensionPolicyConstraints     = asn1.ObjectIdentifier{2, 5, 29, 36}
	oidExtensionInhibitAnyPolicy      = asn1.ObjectIdentifier{2, 5, 29, 54}

	oidAuthorityInfoAccessOcsp    = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 48, 1}
	oidAuthorityInfoAccessIssuers = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 48, 2}
)

// A Certificate holds 1 X.509 certificate.
//
// The overlay holds no PublicKey field. The package doc names the cause.
type Certificate struct {
	// Raw holds the whole DER of the certificate.
	Raw []byte
	// RawTBSCertificate holds the DER of the part that the signature covers.
	RawTBSCertificate []byte
	// RawSubjectPublicKeyInfo holds the DER of the SubjectPublicKeyInfo.
	RawSubjectPublicKeyInfo []byte
	// RawSubject holds the DER of the subject.
	RawSubject []byte
	// RawIssuer holds the DER of the issuer.
	RawIssuer []byte
	// RawSignatureAlgorithm holds the DER of the AlgorithmIdentifier.
	RawSignatureAlgorithm []byte

	Signature          []byte
	SignatureAlgorithm SignatureAlgorithm

	PublicKeyAlgorithm PublicKeyAlgorithm

	Version             int
	SerialNumber        *big.Int
	Issuer              pkix.Name
	Subject             pkix.Name
	NotBefore, NotAfter time.Time
	KeyUsage            KeyUsage

	// Extensions holds each extension of the certificate in its raw form.
	Extensions []pkix.Extension

	// ExtraExtensions holds each extension that CreateCertificate copies raw
	// into the certificate that it makes. Go leaves this field empty after a
	// parse, thus the field stays empty here too, and that matches Go
	// exactly. Read Extensions for each extension of a parsed certificate.
	ExtraExtensions []pkix.Extension

	// UnhandledCriticalExtensions names each critical extension that the
	// parse did not read.
	UnhandledCriticalExtensions []asn1.ObjectIdentifier

	ExtKeyUsage        []ExtKeyUsage
	UnknownExtKeyUsage []asn1.ObjectIdentifier

	// BasicConstraintsValid reports whether IsCA, MaxPathLen, and
	// MaxPathLenZero hold a value.
	BasicConstraintsValid bool
	IsCA                  bool

	// MaxPathLen and MaxPathLenZero report the presence and the value of the
	// pathLenConstraint of the basic constraints. A value of -1 names an
	// absent field.
	MaxPathLen     int
	MaxPathLenZero bool

	SubjectKeyId   []byte
	AuthorityKeyId []byte

	// The access description of the authority. RFC 5280, 4.2.2.1.
	OCSPServer            []string
	IssuingCertificateURL []string

	// The subject alternative names.
	DNSNames       []string
	EmailAddresses []string
	IPAddresses    []net.IP
	URIs           []*url.URL

	// CRLDistributionPoints holds each URI of the CRL distribution points.
	CRLDistributionPoints []string

	// The name constraints. RFC 5280, 4.2.1.10. A constraint holds the names
	// that a certificate under this authority may carry, and the names that
	// it may not carry.
	//
	// PermittedDNSDomainsCritical reports whether the extension carries the
	// critical mark. The overlay reads each field, and it runs no check of a
	// chain, thus no constraint below ever rejects a certificate here.
	PermittedDNSDomainsCritical bool
	PermittedDNSDomains         []string
	ExcludedDNSDomains          []string
	PermittedIPRanges           []*net.IPNet
	ExcludedIPRanges            []*net.IPNet
	PermittedEmailAddresses     []string
	ExcludedEmailAddresses      []string
	PermittedURIDomains         []string
	ExcludedURIDomains          []string

	// PolicyIdentifiers holds each policy identifier of the
	// certificatePolicies extension in the asn1.ObjectIdentifier form. That
	// form holds a component of 31 bits at most. An identifier that needs
	// more bits stands in Policies alone.
	PolicyIdentifiers []asn1.ObjectIdentifier

	// Policies holds each policy identifier of the certificatePolicies
	// extension. RFC 5280, 4.2.1.4.
	Policies []OID

	// InhibitAnyPolicy and InhibitAnyPolicyZero report the presence and the
	// value of the inhibitAnyPolicy extension. RFC 5280, 4.2.1.14.
	//
	// The value counts the certificates after this one in the path. Each such
	// certificate may use the anyPolicy identifier to match any other policy.
	//
	// After a parse, a value above 0 reports a field that the extension held.
	// A value of 0 with InhibitAnyPolicyZero false reports an absent field.
	InhibitAnyPolicy int
	// InhibitAnyPolicyZero reports a value of 0 that the extension held.
	InhibitAnyPolicyZero bool

	// InhibitPolicyMapping and InhibitPolicyMappingZero report the presence
	// and the value of the inhibitPolicyMapping field of the policyConstraints
	// extension. RFC 5280, 4.2.1.11.
	//
	// The value counts the certificates after this one in the path. Each such
	// certificate may use the policyMappings extension.
	//
	// After a parse, a value above 0 reports a field that the extension held.
	// A value of 0 with InhibitPolicyMappingZero false reports an absent
	// field.
	InhibitPolicyMapping int
	// InhibitPolicyMappingZero reports a value of 0 that the extension held.
	InhibitPolicyMappingZero bool

	// RequireExplicitPolicy and RequireExplicitPolicyZero report the presence
	// and the value of the requireExplicitPolicy field of the
	// policyConstraints extension. RFC 5280, 4.2.1.11.
	//
	// The value counts the certificates after this one in the path. After
	// that count, the rest of the path demands an explicit policy. Each such
	// certificate must then hold a required policy identifier. The
	// policyMappings extension may declare an identifier as the equal of one.
	//
	// After a parse, a value above 0 reports a field that the extension held.
	// A value of 0 with RequireExplicitPolicyZero false reports an absent
	// field.
	RequireExplicitPolicy int
	// RequireExplicitPolicyZero reports a value of 0 that the extension held.
	RequireExplicitPolicyZero bool

	// PolicyMappings holds each entry of the policyMappings extension. RFC
	// 5280, 4.2.1.5.
	PolicyMappings []PolicyMapping
}

// A PolicyMapping holds 1 entry of the policyMappings extension. RFC 5280,
// 4.2.1.5.
type PolicyMapping struct {
	// IssuerDomainPolicy holds a policy identifier that the issuer treats as
	// the equal of SubjectDomainPolicy in the subject certificate.
	IssuerDomainPolicy OID
	// SubjectDomainPolicy holds a policy identifier that the issuer treats as
	// the equal of IssuerDomainPolicy in the issuer certificate.
	SubjectDomainPolicy OID
}

// Equal reports whether the 2 certificates hold the same DER.
func (c *Certificate) Equal(other *Certificate) bool {
	if c == nil || other == nil {
		return c == other
	}
	return bytes.Equal(c.Raw, other.Raw)
}

func (c *Certificate) hasSANExtension() bool {
	return oidInExtensions(oidExtensionSubjectAltName, c.Extensions)
}

func oidInExtensions(oid asn1.ObjectIdentifier, extensions []pkix.Extension) bool {
	for _, e := range extensions {
		if e.Id.Equal(oid) {
			return true
		}
	}
	return false
}

// ParseCertificate reads 1 certificate of the DER data.
func ParseCertificate(der []byte) (*Certificate, error) {
	cert := new(certificate)
	rest, err := asn1.Unmarshal(der, cert)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, asn1.SyntaxError{Msg: "trailing data"}
	}
	return buildCertificate(cert)
}

// ParseCertificates reads each certificate that the DER data holds. The
// certificates must stand one after the other, with no separator.
func ParseCertificates(der []byte) ([]*Certificate, error) {
	var out []*Certificate
	for len(der) > 0 {
		cert := new(certificate)
		rest, err := asn1.Unmarshal(der, cert)
		if err != nil {
			return nil, err
		}
		built, err := buildCertificate(cert)
		if err != nil {
			return nil, err
		}
		out = append(out, built)
		der = rest
	}
	return out, nil
}

func buildCertificate(in *certificate) (*Certificate, error) {
	out := new(Certificate)
	out.Raw = in.Raw
	out.RawTBSCertificate = in.TBSCertificate.Raw
	out.RawSubjectPublicKeyInfo = in.TBSCertificate.PublicKey.Raw
	out.RawSubject = in.TBSCertificate.Subject.FullBytes
	out.RawIssuer = in.TBSCertificate.Issuer.FullBytes
	out.RawSignatureAlgorithm = in.SignatureAlgorithm.Raw

	out.Signature = in.SignatureValue.RightAlign()
	out.SignatureAlgorithm = getSignatureAlgorithmFromAI(in.SignatureAlgorithm.ident())
	out.PublicKeyAlgorithm = getPublicKeyAlgorithmFromOID(in.TBSCertificate.PublicKey.Algorithm.Algorithm)

	out.Version = in.TBSCertificate.Version + 1
	out.SerialNumber = in.TBSCertificate.SerialNumber
	if out.SerialNumber.Sign() == -1 {
		// RFC 5280, 4.1.2.2 names a positive integer. Go rejects a negative
		// serial too, and the setting x509negativeserial of GODEBUG turns that
		// rejection off. Each setting of the surface holds its default, thus
		// the rejection here always stands.
		return nil, errors.New("x509: negative serial number")
	}
	out.NotBefore = in.TBSCertificate.Validity.NotBefore
	out.NotAfter = in.TBSCertificate.Validity.NotAfter

	var issuer, subject pkix.RDNSequence
	if rest, err := asn1.Unmarshal(in.TBSCertificate.Subject.FullBytes, &subject); err != nil {
		return nil, err
	} else if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after X.509 subject")
	}
	if rest, err := asn1.Unmarshal(in.TBSCertificate.Issuer.FullBytes, &issuer); err != nil {
		return nil, err
	} else if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after X.509 issuer")
	}
	out.Issuer.FillFromRDNSequence(&issuer)
	out.Subject.FillFromRDNSequence(&subject)

	out.Extensions = in.TBSCertificate.Extensions
	out.MaxPathLen = -1
	for _, e := range out.Extensions {
		handled, err := parseExtension(out, e)
		if err != nil {
			return nil, err
		}
		if !handled && e.Critical {
			out.UnhandledCriticalExtensions = append(out.UnhandledCriticalExtensions, e.Id)
		}
	}
	return out, nil
}

// parseExtension reads 1 extension into the certificate. It reports whether
// the parse understood the extension.
func parseExtension(out *Certificate, e pkix.Extension) (bool, error) {
	switch {
	case e.Id.Equal(oidExtensionSubjectKeyId):
		return true, parseSubjectKeyId(out, e.Value)
	case e.Id.Equal(oidExtensionKeyUsage):
		return true, parseKeyUsage(out, e.Value)
	case e.Id.Equal(oidExtensionSubjectAltName):
		return true, parseSANExtension(out, e.Value)
	case e.Id.Equal(oidExtensionBasicConstraints):
		return true, parseBasicConstraints(out, e.Value)
	case e.Id.Equal(oidExtensionCRLDistributionPoints):
		return true, parseCRLDistributionPoints(out, e.Value)
	case e.Id.Equal(oidExtensionAuthorityKeyId):
		return true, parseAuthorityKeyId(out, e.Value)
	case e.Id.Equal(oidExtensionExtendedKeyUsage):
		return true, parseExtKeyUsage(out, e.Value)
	case e.Id.Equal(oidExtensionAuthorityInfoAccess):
		return true, parseAuthorityInfoAccess(out, e.Value)
	case e.Id.Equal(oidExtensionCertificatePolicies):
		return true, parseCertificatePolicies(out, e.Value)
	case e.Id.Equal(oidExtensionPolicyMappings):
		return true, parsePolicyMappings(out, e.Value)
	case e.Id.Equal(oidExtensionPolicyConstraints):
		return true, parsePolicyConstraints(out, e.Value)
	case e.Id.Equal(oidExtensionInhibitAnyPolicy):
		return true, parseInhibitAnyPolicy(out, e.Value)
	case e.Id.Equal(oidExtensionNameConstraints):
		unhandled, err := parseNameConstraints(out, e)
		return !unhandled, err
	}
	return false, nil
}

func parseKeyUsage(out *Certificate, value []byte) error {
	var usageBits asn1.BitString
	rest, err := asn1.Unmarshal(value, &usageBits)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 KeyUsage")
	}
	var usage int
	for i := 0; i < 9; i++ {
		if usageBits.At(i) != 0 {
			usage |= 1 << uint(i)
		}
	}
	out.KeyUsage = KeyUsage(usage)
	return nil
}

func parseBasicConstraints(out *Certificate, value []byte) error {
	var constraints basicConstraints
	rest, err := asn1.Unmarshal(value, &constraints)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 BasicConstraints")
	}
	out.BasicConstraintsValid = true
	out.IsCA = constraints.IsCA
	out.MaxPathLen = constraints.MaxPathLen
	out.MaxPathLenZero = out.MaxPathLen == 0
	return nil
}

func parseSubjectKeyId(out *Certificate, value []byte) error {
	var keyid []byte
	rest, err := asn1.Unmarshal(value, &keyid)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 key-id")
	}
	out.SubjectKeyId = keyid
	return nil
}

func parseAuthorityKeyId(out *Certificate, value []byte) error {
	var a authKeyId
	rest, err := asn1.Unmarshal(value, &a)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 authority key-id")
	}
	out.AuthorityKeyId = a.Id
	return nil
}

func parseExtKeyUsage(out *Certificate, value []byte) error {
	var keyUsage []asn1.ObjectIdentifier
	rest, err := asn1.Unmarshal(value, &keyUsage)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 ExtendedKeyUsage")
	}
	for _, u := range keyUsage {
		if extKeyUsage, ok := extKeyUsageFromOID(u); ok {
			out.ExtKeyUsage = append(out.ExtKeyUsage, extKeyUsage)
			continue
		}
		out.UnknownExtKeyUsage = append(out.UnknownExtKeyUsage, u)
	}
	return nil
}

func parseCRLDistributionPoints(out *Certificate, value []byte) error {
	var cdp []distributionPoint
	rest, err := asn1.Unmarshal(value, &cdp)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 CRL distribution point")
	}
	for _, dp := range cdp {
		// RFC 5280, 4.2.1.13 allows an empty distributionPoint.
		if len(dp.DistributionPoint.FullName.Bytes) == 0 {
			continue
		}
		dpName := dp.DistributionPoint.FullName.Bytes
		for len(dpName) > 0 {
			var n asn1.RawValue
			dpName, err = asn1.Unmarshal(dpName, &n)
			if err != nil {
				return err
			}
			if n.Tag == 6 {
				out.CRLDistributionPoints = append(out.CRLDistributionPoints, string(n.Bytes))
			}
		}
	}
	return nil
}

func parseAuthorityInfoAccess(out *Certificate, value []byte) error {
	var aia []authorityInfoAccess
	rest, err := asn1.Unmarshal(value, &aia)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 authority information")
	}
	for _, v := range aia {
		// RFC 5280, 4.2.2.1 names a GeneralName, and the parse reads the
		// uniformResourceIdentifier form alone.
		if v.Location.Tag != 6 {
			continue
		}
		if v.Method.Equal(oidAuthorityInfoAccessOcsp) {
			out.OCSPServer = append(out.OCSPServer, string(v.Location.Bytes))
		} else if v.Method.Equal(oidAuthorityInfoAccessIssuers) {
			out.IssuingCertificateURL = append(out.IssuingCertificateURL, string(v.Location.Bytes))
		}
	}
	return nil
}

// parseCertificatePolicies reads the certificatePolicies extension. RFC 5280,
// 4.2.1.4. The parse reads the policyIdentifier of each entry, and it steps
// over each policy qualifier.
func parseCertificatePolicies(out *Certificate, value []byte) error {
	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(value, &seq)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 certificate policies")
	}
	if !seq.IsCompound || seq.Tag != 16 || seq.Class != 0 {
		return errors.New("x509: invalid certificate policies")
	}
	seenOIDs := map[string]bool{}
	var policies []OID
	body := seq.Bytes
	for len(body) > 0 {
		var info asn1.RawValue
		body, err = asn1.Unmarshal(body, &info)
		if err != nil {
			return err
		}
		if !info.IsCompound || info.Tag != 16 || info.Class != 0 {
			return errors.New("x509: invalid certificate policies")
		}
		var ident asn1.RawValue
		if _, err = asn1.Unmarshal(info.Bytes, &ident); err != nil {
			return err
		}
		if ident.IsCompound || ident.Tag != 6 || ident.Class != 0 {
			return errors.New("x509: invalid certificate policies")
		}
		// RFC 5280, 4.2.1.4 allows 1 entry per policy identifier alone.
		if seenOIDs[string(ident.Bytes)] {
			return errors.New("x509: invalid certificate policies")
		}
		seenOIDs[string(ident.Bytes)] = true
		oid, ok := newOIDFromDER(ident.Bytes)
		if !ok {
			return errors.New("x509: invalid certificate policies")
		}
		policies = append(policies, oid)
	}
	out.Policies = policies
	// PolicyIdentifiers holds each identifier that fits the
	// asn1.ObjectIdentifier form. An identifier of a wider component drops
	// out, and Policies still holds it.
	out.PolicyIdentifiers = make([]asn1.ObjectIdentifier, 0, len(policies))
	for _, oid := range policies {
		if asn1OID, ok := oid.toASN1OID(); ok {
			out.PolicyIdentifiers = append(out.PolicyIdentifiers, asn1OID)
		}
	}
	return nil
}

// parsePolicyMappings reads the policyMappings extension. RFC 5280, 4.2.1.5.
func parsePolicyMappings(out *Certificate, value []byte) error {
	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(value, &seq)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 policy mappings")
	}
	if !seq.IsCompound || seq.Tag != 16 || seq.Class != 0 {
		return errors.New("x509: invalid policy mappings extension")
	}
	body := seq.Bytes
	for len(body) > 0 {
		var pair asn1.RawValue
		body, err = asn1.Unmarshal(body, &pair)
		if err != nil {
			return err
		}
		if !pair.IsCompound || pair.Tag != 16 || pair.Class != 0 {
			return errors.New("x509: invalid policy mappings extension")
		}
		var issuer, subject asn1.RawValue
		var pairBody []byte
		pairBody, err = asn1.Unmarshal(pair.Bytes, &issuer)
		if err != nil {
			return err
		}
		if _, err = asn1.Unmarshal(pairBody, &subject); err != nil {
			return err
		}
		if issuer.IsCompound || issuer.Tag != 6 || issuer.Class != 0 {
			return errors.New("x509: invalid policy mappings extension")
		}
		if subject.IsCompound || subject.Tag != 6 || subject.Class != 0 {
			return errors.New("x509: invalid policy mappings extension")
		}
		// Go holds the DER of each identifier here, and it runs no check of
		// the base 128 form. The parse of the overlay does the same.
		out.PolicyMappings = append(out.PolicyMappings, PolicyMapping{
			IssuerDomainPolicy:  OID{der: issuer.Bytes},
			SubjectDomainPolicy: OID{der: subject.Bytes},
		})
	}
	return nil
}

// parsePolicyConstraints reads the policyConstraints extension. RFC 5280,
// 4.2.1.11.
func parsePolicyConstraints(out *Certificate, value []byte) error {
	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(value, &seq)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 policy constraints")
	}
	if !seq.IsCompound || seq.Tag != 16 || seq.Class != 0 {
		return errors.New("x509: invalid policy constraints extension")
	}
	body := seq.Bytes
	// The 2 fields are each optional. requireExplicitPolicy carries the
	// context tag 0, thus the first octet of it holds the value 0x80.
	// inhibitPolicyMapping carries the context tag 1, thus the first octet of
	// it holds the value 0x81.
	if len(body) > 0 && body[0] == 0x80 {
		var v int64
		body, err = asn1.UnmarshalWithParams(body, &v, "tag:0")
		if err != nil {
			return errors.New("x509: invalid policy constraints extension")
		}
		out.RequireExplicitPolicy = int(v)
		if int64(out.RequireExplicitPolicy) != v {
			return errors.New("x509: policy constraints requireExplicitPolicy field overflows int")
		}
		out.RequireExplicitPolicyZero = out.RequireExplicitPolicy == 0
	}
	if len(body) > 0 && body[0] == 0x81 {
		var v int64
		body, err = asn1.UnmarshalWithParams(body, &v, "tag:1")
		if err != nil {
			return errors.New("x509: invalid policy constraints extension")
		}
		out.InhibitPolicyMapping = int(v)
		if int64(out.InhibitPolicyMapping) != v {
			return errors.New("x509: policy constraints inhibitPolicyMapping field overflows int")
		}
		out.InhibitPolicyMappingZero = out.InhibitPolicyMapping == 0
	}
	return nil
}

// parseInhibitAnyPolicy reads the inhibitAnyPolicy extension. RFC 5280,
// 4.2.1.14.
func parseInhibitAnyPolicy(out *Certificate, value []byte) error {
	var v int64
	rest, err := asn1.Unmarshal(value, &v)
	if err != nil {
		return errors.New("x509: invalid inhibit any policy extension")
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 inhibit any policy")
	}
	out.InhibitAnyPolicy = int(v)
	if int64(out.InhibitAnyPolicy) != v {
		return errors.New("x509: invalid inhibit any policy extension")
	}
	out.InhibitAnyPolicyZero = out.InhibitAnyPolicy == 0
	return nil
}

func parseSANExtension(out *Certificate, value []byte) error {
	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(value, &seq)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 extension")
	}
	if !seq.IsCompound || seq.Tag != 16 || seq.Class != 0 {
		return asn1.StructuralError{Msg: "bad SAN sequence"}
	}
	rest = seq.Bytes
	for len(rest) > 0 {
		var v asn1.RawValue
		rest, err = asn1.Unmarshal(rest, &v)
		if err != nil {
			return err
		}
		switch v.Tag {
		case 1:
			out.EmailAddresses = append(out.EmailAddresses, string(v.Bytes))
		case 2:
			out.DNSNames = append(out.DNSNames, string(v.Bytes))
		case 6:
			uri, err := url.Parse(string(v.Bytes))
			if err != nil {
				return fmt.Errorf("x509: cannot parse URI %q: %s", string(v.Bytes), err)
			}
			if len(uri.Host) > 0 {
				if _, ok := domainToReverseLabels(uri.Host); !ok {
					return fmt.Errorf("x509: cannot parse URI %q: invalid domain", string(v.Bytes))
				}
			}
			out.URIs = append(out.URIs, uri)
		case 7:
			switch len(v.Bytes) {
			case net.IPv4len, net.IPv6len:
				out.IPAddresses = append(out.IPAddresses, v.Bytes)
			default:
				return errors.New("x509: cannot parse IP address of length " + strconv.Itoa(len(v.Bytes)))
			}
		}
	}
	return nil
}

// domainToReverseLabels gives the labels of a domain in reverse order, and it
// reports whether the domain holds a valid form.
func domainToReverseLabels(domain string) (reverseLabels []string, ok bool) {
	for len(domain) > 0 {
		if i := strings.LastIndexByte(domain, '.'); i == -1 {
			reverseLabels = append(reverseLabels, domain)
			domain = ""
		} else {
			reverseLabels = append(reverseLabels, domain[i+1:])
			domain = domain[:i]
		}
	}
	if len(reverseLabels) > 0 && len(reverseLabels[0]) == 0 {
		// An empty label at the end names an absolute domain.
		return nil, false
	}
	for _, label := range reverseLabels {
		if len(label) == 0 {
			return nil, false
		}
		for _, c := range label {
			if c < 33 || c > 126 {
				return nil, false
			}
		}
	}
	return reverseLabels, true
}

// A HostnameError reports that no name of a certificate matches a host.
type HostnameError struct {
	Certificate *Certificate
	Host        string
}

func (h HostnameError) Error() string {
	c := h.Certificate
	maxNamesIncluded := 100

	if !c.hasSANExtension() && matchHostnames(c.Subject.CommonName, splitHostname(h.Host)) {
		return "x509: certificate relies on legacy Common Name field, use SANs instead"
	}

	var valid strings.Builder
	if ip := net.ParseIP(h.Host); ip != nil {
		if len(c.IPAddresses) == 0 {
			return "x509: cannot validate certificate for " + h.Host + " because it doesn't contain any IP SANs"
		}
		if len(c.IPAddresses) >= maxNamesIncluded {
			return fmt.Sprintf("x509: certificate is valid for %d IP SANs, but none matched %s", len(c.IPAddresses), h.Host)
		}
		for _, san := range c.IPAddresses {
			if valid.Len() > 0 {
				valid.WriteString(", ")
			}
			valid.WriteString(san.String())
		}
	} else {
		if len(c.DNSNames) >= maxNamesIncluded {
			return fmt.Sprintf("x509: certificate is valid for %d names, but none matched %s", len(c.DNSNames), h.Host)
		}
		valid.WriteString(strings.Join(c.DNSNames, ", "))
	}

	if valid.Len() == 0 {
		return "x509: certificate is not valid for any names, but wanted to match " + h.Host
	}
	return "x509: certificate is valid for " + valid.String() + ", not " + h.Host
}

func validHostnamePattern(host string) bool { return validHostname(host, true) }
func validHostnameInput(host string) bool   { return validHostname(host, false) }

// validHostname reports whether a host holds a form that RFC 6125 2.2 allows,
// with the same leniency that Go holds for a legacy value.
func validHostname(host string, isPattern bool) bool {
	if !isPattern {
		host = strings.TrimSuffix(host, ".")
	}
	if len(host) == 0 {
		return false
	}
	if host == "*" {
		// A bare wildcard is no valid DNS name.
		return false
	}
	for i, part := range strings.Split(host, ".") {
		if part == "" {
			return false
		}
		if isPattern && i == 0 && part == "*" {
			// Only a whole leftmost wildcard matches.
			continue
		}
		for j, c := range part {
			if 'a' <= c && c <= 'z' {
				continue
			}
			if '0' <= c && c <= '9' {
				continue
			}
			if 'A' <= c && c <= 'Z' {
				continue
			}
			if c == '-' && j != 0 {
				continue
			}
			if c == '_' {
				// No valid character of a host name, and a deployment
				// outside the web PKI holds it often.
				continue
			}
			return false
		}
	}
	return true
}

func matchExactly(hostA, hostB string) bool {
	if hostA == "" || hostA == "." || hostB == "" || hostB == "." {
		return false
	}
	return toLowerCaseASCII(hostA) == toLowerCaseASCII(hostB)
}

func matchHostnames(pattern string, hostParts []string) bool {
	pattern = toLowerCaseASCII(pattern)
	if len(pattern) == 0 || len(hostParts) == 0 {
		return false
	}
	patternParts := strings.Split(pattern, ".")
	if len(patternParts) != len(hostParts) {
		return false
	}
	for i, patternPart := range patternParts {
		if i == 0 && patternPart == "*" {
			continue
		}
		if patternPart != hostParts[i] {
			return false
		}
	}
	return true
}

// toLowerCaseASCII gives the lower case form of a text. RFC 6125 6.4.1 names
// an operation over ASCII alone, thus the function reads no Unicode rule.
func toLowerCaseASCII(in string) string {
	isAlreadyLowerCase := true
	for _, c := range in {
		if c == utf8.RuneError {
			// A broken sequence can hold an upper case ASCII byte.
			isAlreadyLowerCase = false
			break
		}
		if 'A' <= c && c <= 'Z' {
			isAlreadyLowerCase = false
			break
		}
	}
	if isAlreadyLowerCase {
		return in
	}
	out := []byte(in)
	for i, c := range out {
		if 'A' <= c && c <= 'Z' {
			out[i] += 'a' - 'A'
		}
	}
	return string(out)
}

func splitHostname(host string) []string {
	host = strings.TrimSuffix(host, ".")
	return strings.Split(host, ".")
}

// VerifyHostname gives nil when a name of the certificate matches the host.
// It reads the subject alternative names alone, and it builds no chain and
// checks no signature.
//
// An IP address can stand inside square brackets, and it matches the IP names
// alone. Each other host matches the DNS names, with no case rule. A name of
// the certificate can hold a wildcard as the whole leftmost label.
//
// The legacy Common Name field reaches no match, exactly as in Go.
func (c *Certificate) VerifyHostname(h string) error {
	candidateIP := h
	if len(h) >= 3 && h[0] == '[' && h[len(h)-1] == ']' {
		candidateIP = h[1 : len(h)-1]
	}
	if addr, err := netip.ParseAddr(candidateIP); err == nil {
		// RFC 6125, Appendix B.2 names the IP names alone.
		ip := net.IP(addr.AsSlice())
		for _, candidate := range c.IPAddresses {
			if ip.Equal(candidate) {
				return nil
			}
		}
		return HostnameError{c, ip.String()}
	}

	candidateName := toLowerCaseASCII(h)
	validCandidateName := validHostnameInput(candidateName)
	hostParts := splitHostname(candidateName)

	for _, match := range c.DNSNames {
		// A perfect match always passes. The wildcard rule and the trailing
		// dot rule reach a valid host name alone.
		if validCandidateName && validHostnamePattern(match) {
			if matchHostnames(match, hostParts) {
				return nil
			}
		} else {
			if matchExactly(match, candidateName) {
				return nil
			}
		}
	}
	return HostnameError{c, h}
}
