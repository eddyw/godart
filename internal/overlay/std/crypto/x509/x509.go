//go:build dart

// Package x509 reads an X.509 certificate.
//
// The overlay holds the parse path alone. It reads a DER certificate or a PEM
// certificate, it gives each field of the certificate that the parse
// understands, and it holds a pool of certificates.
//
// The overlay holds no trust store of the platform. The secure socket of Dart
// owns that store, thus SystemCertPool always reports a failure of the load.
//
// A key of ML-DSA stays out, because crypto/mldsa stands outside the surface.
// A certificate that carries such a key gives an error of the parse.
package x509

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"time"

	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"internal/godebug"
	"io"

	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
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
	FullName     []asn1.RawValue  `asn1:"optional,tag:0"`
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

// signatureAlgorithmDetails holds the name, the object identifier, the
// parameters, the public key algorithm, and the hash of each algorithm. The
// table comes from the real package.
var signatureAlgorithmDetails = []struct {
	algo       SignatureAlgorithm
	name       string
	oid        asn1.ObjectIdentifier
	params     asn1.RawValue
	pubKeyAlgo PublicKeyAlgorithm
	hash       crypto.Hash
	isRSAPSS   bool
}{
	{MD5WithRSA, "MD5-RSA", oidSignatureMD5WithRSA, asn1.NullRawValue, RSA, crypto.MD5, false},
	{SHA1WithRSA, "SHA1-RSA", oidSignatureSHA1WithRSA, asn1.NullRawValue, RSA, crypto.SHA1, false},
	{SHA1WithRSA, "SHA1-RSA", oidISOSignatureSHA1WithRSA, asn1.NullRawValue, RSA, crypto.SHA1, false},
	{SHA256WithRSA, "SHA256-RSA", oidSignatureSHA256WithRSA, asn1.NullRawValue, RSA, crypto.SHA256, false},
	{SHA384WithRSA, "SHA384-RSA", oidSignatureSHA384WithRSA, asn1.NullRawValue, RSA, crypto.SHA384, false},
	{SHA512WithRSA, "SHA512-RSA", oidSignatureSHA512WithRSA, asn1.NullRawValue, RSA, crypto.SHA512, false},
	{SHA256WithRSAPSS, "SHA256-RSAPSS", oidSignatureRSAPSS, pssParametersSHA256, RSA, crypto.SHA256, true},
	{SHA384WithRSAPSS, "SHA384-RSAPSS", oidSignatureRSAPSS, pssParametersSHA384, RSA, crypto.SHA384, true},
	{SHA512WithRSAPSS, "SHA512-RSAPSS", oidSignatureRSAPSS, pssParametersSHA512, RSA, crypto.SHA512, true},
	{DSAWithSHA1, "DSA-SHA1", oidSignatureDSAWithSHA1, emptyRawValue, DSA, crypto.SHA1, false},
	{DSAWithSHA256, "DSA-SHA256", oidSignatureDSAWithSHA256, emptyRawValue, DSA, crypto.SHA256, false},
	{ECDSAWithSHA1, "ECDSA-SHA1", oidSignatureECDSAWithSHA1, emptyRawValue, ECDSA, crypto.SHA1, false},
	{ECDSAWithSHA256, "ECDSA-SHA256", oidSignatureECDSAWithSHA256, emptyRawValue, ECDSA, crypto.SHA256, false},
	{ECDSAWithSHA384, "ECDSA-SHA384", oidSignatureECDSAWithSHA384, emptyRawValue, ECDSA, crypto.SHA384, false},
	{ECDSAWithSHA512, "ECDSA-SHA512", oidSignatureECDSAWithSHA512, emptyRawValue, ECDSA, crypto.SHA512, false},
	{PureEd25519, "Ed25519", oidSignatureEd25519, emptyRawValue, Ed25519, crypto.Hash(0) /* no pre-hashing */, false},
	{MLDSA44, "ML-DSA-44", oidPublicKeyMLDSA44, emptyRawValue, MLDSA, crypto.Hash(0) /* no pre-hashing */, false},
	{MLDSA65, "ML-DSA-65", oidPublicKeyMLDSA65, emptyRawValue, MLDSA, crypto.Hash(0) /* no pre-hashing */, false},
	{MLDSA87, "ML-DSA-87", oidPublicKeyMLDSA87, emptyRawValue, MLDSA, crypto.Hash(0) /* no pre-hashing */, false},
}

var emptyRawValue = asn1.RawValue{}

// The DER of the parameters of an RSA PSS signature with SHA256, with SHA384,
// and with SHA512. RFC 3447, appendix A.2.3. The hash of MGF1 matches the hash
// of the message, the salt length matches the length of the hash, and the
// trailer field holds its default.
var (
	pssParametersSHA256 = asn1.RawValue{FullBytes: []byte{48, 52, 160, 15, 48, 13, 6, 9, 96, 134, 72, 1, 101, 3, 4, 2, 1, 5, 0, 161, 28, 48, 26, 6, 9, 42, 134, 72, 134, 247, 13, 1, 1, 8, 48, 13, 6, 9, 96, 134, 72, 1, 101, 3, 4, 2, 1, 5, 0, 162, 3, 2, 1, 32}}
	pssParametersSHA384 = asn1.RawValue{FullBytes: []byte{48, 52, 160, 15, 48, 13, 6, 9, 96, 134, 72, 1, 101, 3, 4, 2, 2, 5, 0, 161, 28, 48, 26, 6, 9, 42, 134, 72, 134, 247, 13, 1, 1, 8, 48, 13, 6, 9, 96, 134, 72, 1, 101, 3, 4, 2, 2, 5, 0, 162, 3, 2, 1, 48}}
	pssParametersSHA512 = asn1.RawValue{FullBytes: []byte{48, 52, 160, 15, 48, 13, 6, 9, 96, 134, 72, 1, 101, 3, 4, 2, 3, 5, 0, 161, 28, 48, 26, 6, 9, 42, 134, 72, 134, 247, 13, 1, 1, 8, 48, 13, 6, 9, 96, 134, 72, 1, 101, 3, 4, 2, 3, 5, 0, 162, 3, 2, 1, 64}}
)

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
	// PublicKey holds the key of the subject. It is a *rsa.PublicKey, a
	// *dsa.PublicKey, a *ecdsa.PublicKey, an ed25519.PublicKey, or a
	// *ecdh.PublicKey for X25519.
	PublicKey any

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

// ErrUnsupportedAlgorithm names an operation over an algorithm that the
// package does not hold.
var ErrUnsupportedAlgorithm = errors.New("x509: cannot verify signature: algorithm unimplemented")

// An InsecureAlgorithmError names a signature algorithm that is not secure.
// The check rejects such a signature.
type InsecureAlgorithmError SignatureAlgorithm

func (e InsecureAlgorithmError) Error() string {
	return fmt.Sprintf("x509: cannot verify signature: insecure algorithm %v", SignatureAlgorithm(e))
}

// A ConstraintViolationError names a use that the certificate does not permit.
// A check of a signature with a key that signs no certificate gives one.
type ConstraintViolationError struct{}

func (ConstraintViolationError) Error() string {
	return "x509: invalid signature: parent certificate cannot sign this kind of certificate"
}

// An UnhandledCriticalExtension names a critical extension that the check did
// not read.
type UnhandledCriticalExtension struct{}

func (h UnhandledCriticalExtension) Error() string {
	return "x509: unhandled critical extension"
}

// CheckSignatureFrom reports whether the signature on c is a valid signature
// of parent.
//
// The call runs few checks, and it builds no chain. Read Certificate.Verify.
func (c *Certificate) CheckSignatureFrom(parent *Certificate) error {
	// RFC 5280, 4.2.1.9: a version 3 certificate with no basic constraints,
	// or with the cA boolean absent, signs no certificate.
	if parent.Version == 3 && !parent.BasicConstraintsValid ||
		parent.BasicConstraintsValid && !parent.IsCA {
		return ConstraintViolationError{}
	}

	if parent.KeyUsage != 0 && parent.KeyUsage&KeyUsageCertSign == 0 {
		return ConstraintViolationError{}
	}

	if parent.PublicKeyAlgorithm == UnknownPublicKeyAlgorithm {
		return ErrUnsupportedAlgorithm
	}

	return checkSignature(c.SignatureAlgorithm, c.RawTBSCertificate, c.Signature, parent.PublicKey, false)
}

// CheckSignature reports whether signature is a valid signature over signed of
// the public key of c.
//
// The call runs no check of the validity of the certificate.
//
// It rejects a signature of MD5WithRSA, and it accepts a signature of
// SHA1WithRSA and of ECDSAWithSHA1.
func (c *Certificate) CheckSignature(algo SignatureAlgorithm, signed, signature []byte) error {
	return checkSignature(algo, signed, signature, c.PublicKey, true)
}

// CheckCRLSignature reports whether the signature of crl comes from c.
//
// Deprecated: Use RevocationList.CheckSignatureFrom instead.
func (c *Certificate) CheckCRLSignature(crl *pkix.CertificateList) error {
	algo := getSignatureAlgorithmFromAI(crl.SignatureAlgorithm)
	return c.CheckSignature(algo, crl.TBSCertList.Raw, crl.SignatureValue.RightAlign())
}

func signaturePublicKeyAlgoMismatchError(expectedPubKeyAlgo PublicKeyAlgorithm, pubKey any) error {
	return fmt.Errorf("x509: signature algorithm specifies an %s public key, but have public key of type %T", expectedPubKeyAlgo.String(), pubKey)
}

// checkSignature reports whether signature is a valid signature over signed of
// publicKey.
func checkSignature(algo SignatureAlgorithm, signed, signature []byte, publicKey crypto.PublicKey, allowSHA1 bool) (err error) {
	var hashType crypto.Hash
	var pubKeyAlgo PublicKeyAlgorithm

	for _, details := range signatureAlgorithmDetails {
		if details.algo == algo {
			hashType = details.hash
			pubKeyAlgo = details.pubKeyAlgo
			break
		}
	}

	switch hashType {
	case crypto.Hash(0):
		if pubKeyAlgo != Ed25519 && pubKeyAlgo != MLDSA {
			return ErrUnsupportedAlgorithm
		}
	case crypto.MD5:
		return InsecureAlgorithmError(algo)
	case crypto.SHA1:
		// A signature of SHA-1 stands for a CRL and for a CSR alone.
		if !allowSHA1 {
			return InsecureAlgorithmError(algo)
		}
		fallthrough
	default:
		if !hashType.Available() {
			return ErrUnsupportedAlgorithm
		}
		h := hashType.New()
		h.Write(signed)
		signed = h.Sum(nil)
	}

	switch pub := publicKey.(type) {
	case *rsa.PublicKey:
		if pubKeyAlgo != RSA {
			return signaturePublicKeyAlgoMismatchError(pubKeyAlgo, pub)
		}
		if algo.isRSAPSS() {
			return rsa.VerifyPSS(pub, hashType, signed, signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
		} else {
			return rsa.VerifyPKCS1v15(pub, hashType, signed, signature)
		}
	case *ecdsa.PublicKey:
		if pubKeyAlgo != ECDSA {
			return signaturePublicKeyAlgoMismatchError(pubKeyAlgo, pub)
		}
		if !ecdsa.VerifyASN1(pub, signed, signature) {
			return errors.New("x509: ECDSA verification failure")
		}
		return
	case ed25519.PublicKey:
		if pubKeyAlgo != Ed25519 {
			return signaturePublicKeyAlgoMismatchError(pubKeyAlgo, pub)
		}
		if !ed25519.Verify(pub, signed, signature) {
			return errors.New("x509: Ed25519 verification failure")
		}
		return
	}
	return ErrUnsupportedAlgorithm
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
	if out.PublicKeyAlgorithm != UnknownPublicKeyAlgorithm {
		key, err := parsePublicKey(&in.TBSCertificate.PublicKey)
		if err != nil {
			return nil, err
		}
		out.PublicKey = key
	}

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
		if len(dp.DistributionPoint.FullName) == 0 {
			continue
		}
		for _, fullName := range dp.DistributionPoint.FullName {
			if fullName.Tag == 6 {
				out.CRLDistributionPoints = append(out.CRLDistributionPoints, string(fullName.Bytes))
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

func (c *Certificate) hasNameConstraints() bool {
	return oidInExtensions(oidExtensionNameConstraints, c.Extensions)
}

func (c *Certificate) getSANExtension() []byte {
	for _, e := range c.Extensions {
		if e.Id.Equal(oidExtensionSubjectAltName) {
			return e.Value
		}
	}
	return nil
}

// The tag of each name of a GeneralName. RFC 5280, 4.2.1.6.
const (
	nameTypeEmail = 1
	nameTypeDNS   = 2
	nameTypeURI   = 6
	nameTypeIP    = 7
)

// oidFromExtKeyUsage gives the object identifier of an extended key usage.
func oidFromExtKeyUsage(eku ExtKeyUsage) (oid asn1.ObjectIdentifier, ok bool) {
	for _, pair := range extKeyUsageOIDs {
		if eku == pair.extKeyUsage {
			return pair.oid, true
		}
	}
	return
}

// reverseBitsInAByte writes the 8 bits of a byte in the reverse order.
func reverseBitsInAByte(in byte) byte {
	b1 := in>>4 | in<<4
	b2 := b1>>2&0x33 | b1<<2&0xcc
	b3 := b2>>1&0x55 | b2<<1&0xaa
	return b3
}

// asn1BitLength gives the count of the bits of a bit string. The first bit of
// a byte is the most significant 1, which is the rule of ASN.1.
func asn1BitLength(bitString []byte) int {
	bitLen := len(bitString) * 8

	for i := range bitString {
		b := bitString[len(bitString)-i-1]

		for bit := uint(0); bit < 8; bit++ {
			if (b>>bit)&1 == 1 {
				return bitLen
			}
			bitLen--
		}
	}

	return 0
}

// The lines below come from the file x509.go of crypto/x509 of Go. They build
// a certificate, a certificate request, and a revocation list.

// marshalSANs marshals a list of addresses into a the contents of an X.509
// SubjectAlternativeName extension.
func marshalSANs(dnsNames, emailAddresses []string, ipAddresses []net.IP, uris []*url.URL) (derBytes []byte, err error) {
	var rawValues []asn1.RawValue
	for _, name := range dnsNames {
		if err := isIA5String(name); err != nil {
			return nil, err
		}
		rawValues = append(rawValues, asn1.RawValue{Tag: nameTypeDNS, Class: 2, Bytes: []byte(name)})
	}
	for _, email := range emailAddresses {
		if err := isIA5String(email); err != nil {
			return nil, err
		}
		rawValues = append(rawValues, asn1.RawValue{Tag: nameTypeEmail, Class: 2, Bytes: []byte(email)})
	}
	for _, rawIP := range ipAddresses {
		// If possible, we always want to encode IPv4 addresses in 4 bytes.
		ip := rawIP.To4()
		if ip == nil {
			ip = rawIP
		}
		rawValues = append(rawValues, asn1.RawValue{Tag: nameTypeIP, Class: 2, Bytes: ip})
	}
	for _, uri := range uris {
		uriStr := uri.String()
		if err := isIA5String(uriStr); err != nil {
			return nil, err
		}
		rawValues = append(rawValues, asn1.RawValue{Tag: nameTypeURI, Class: 2, Bytes: []byte(uriStr)})
	}
	return asn1.Marshal(rawValues)
}

var x509usepolicies = godebug.New("x509usepolicies")

func buildCertExtensions(template *Certificate, subjectIsEmpty bool, authorityKeyId []byte, subjectKeyId []byte) (ret []pkix.Extension, err error) {
	ret = make([]pkix.Extension, 10 /* maximum number of elements. */)
	n := 0

	if template.KeyUsage != 0 &&
		!oidInExtensions(oidExtensionKeyUsage, template.ExtraExtensions) {
		ret[n], err = marshalKeyUsage(template.KeyUsage)
		if err != nil {
			return nil, err
		}
		n++
	}

	if (len(template.ExtKeyUsage) > 0 || len(template.UnknownExtKeyUsage) > 0) &&
		!oidInExtensions(oidExtensionExtendedKeyUsage, template.ExtraExtensions) {
		ret[n], err = marshalExtKeyUsage(template.ExtKeyUsage, template.UnknownExtKeyUsage)
		if err != nil {
			return nil, err
		}
		n++
	}

	if template.BasicConstraintsValid && !oidInExtensions(oidExtensionBasicConstraints, template.ExtraExtensions) {
		ret[n], err = marshalBasicConstraints(template.IsCA, template.MaxPathLen, template.MaxPathLenZero)
		if err != nil {
			return nil, err
		}
		n++
	}

	if len(subjectKeyId) > 0 && !oidInExtensions(oidExtensionSubjectKeyId, template.ExtraExtensions) {
		ret[n].Id = oidExtensionSubjectKeyId
		ret[n].Value, err = asn1.Marshal(subjectKeyId)
		if err != nil {
			return
		}
		n++
	}

	if len(authorityKeyId) > 0 && !oidInExtensions(oidExtensionAuthorityKeyId, template.ExtraExtensions) {
		ret[n].Id = oidExtensionAuthorityKeyId
		ret[n].Value, err = asn1.Marshal(authKeyId{authorityKeyId})
		if err != nil {
			return
		}
		n++
	}

	if (len(template.OCSPServer) > 0 || len(template.IssuingCertificateURL) > 0) &&
		!oidInExtensions(oidExtensionAuthorityInfoAccess, template.ExtraExtensions) {
		ret[n].Id = oidExtensionAuthorityInfoAccess
		var aiaValues []authorityInfoAccess
		for _, name := range template.OCSPServer {
			aiaValues = append(aiaValues, authorityInfoAccess{
				Method:   oidAuthorityInfoAccessOcsp,
				Location: asn1.RawValue{Tag: 6, Class: 2, Bytes: []byte(name)},
			})
		}
		for _, name := range template.IssuingCertificateURL {
			aiaValues = append(aiaValues, authorityInfoAccess{
				Method:   oidAuthorityInfoAccessIssuers,
				Location: asn1.RawValue{Tag: 6, Class: 2, Bytes: []byte(name)},
			})
		}
		ret[n].Value, err = asn1.Marshal(aiaValues)
		if err != nil {
			return
		}
		n++
	}

	if (len(template.DNSNames) > 0 || len(template.EmailAddresses) > 0 || len(template.IPAddresses) > 0 || len(template.URIs) > 0) &&
		!oidInExtensions(oidExtensionSubjectAltName, template.ExtraExtensions) {
		ret[n].Id = oidExtensionSubjectAltName
		// From RFC 5280, Section 4.2.1.6:
		// “If the subject field contains an empty sequence ... then
		// subjectAltName extension ... is marked as critical”
		ret[n].Critical = subjectIsEmpty
		ret[n].Value, err = marshalSANs(template.DNSNames, template.EmailAddresses, template.IPAddresses, template.URIs)
		if err != nil {
			return
		}
		n++
	}

	usePolicies := x509usepolicies.Value() != "0"
	if ((!usePolicies && len(template.PolicyIdentifiers) > 0) || (usePolicies && len(template.Policies) > 0)) &&
		!oidInExtensions(oidExtensionCertificatePolicies, template.ExtraExtensions) {
		ret[n], err = marshalCertificatePolicies(template.Policies, template.PolicyIdentifiers)
		if err != nil {
			return nil, err
		}
		n++
	}

	if (len(template.PermittedDNSDomains) > 0 || len(template.ExcludedDNSDomains) > 0 ||
		len(template.PermittedIPRanges) > 0 || len(template.ExcludedIPRanges) > 0 ||
		len(template.PermittedEmailAddresses) > 0 || len(template.ExcludedEmailAddresses) > 0 ||
		len(template.PermittedURIDomains) > 0 || len(template.ExcludedURIDomains) > 0) &&
		!oidInExtensions(oidExtensionNameConstraints, template.ExtraExtensions) {
		ret[n].Id = oidExtensionNameConstraints
		ret[n].Critical = template.PermittedDNSDomainsCritical

		ipAndMask := func(ipNet *net.IPNet) ([]byte, error) {
			maskedIP := ipNet.IP.Mask(ipNet.Mask)
			// This is extremely unlikely to actually happen, but lets save people from doing something they
			// probably shouldn't.
			if len(maskedIP) == net.IPv6len && maskedIP.To4() != nil {
				return nil, errors.New("x509: IP constraint contained IPv4-mapped IPv6 address with a IPv6 mask")
			}
			ipAndMask := make([]byte, 0, len(maskedIP)+len(ipNet.Mask))
			ipAndMask = append(ipAndMask, maskedIP...)
			ipAndMask = append(ipAndMask, ipNet.Mask...)
			return ipAndMask, nil
		}

		serialiseConstraints := func(dns []string, ips []*net.IPNet, emails []string, uriDomains []string) (der []byte, err error) {
			var b cryptobyte.Builder

			for _, name := range dns {
				if err = isIA5String(name); err != nil {
					return nil, err
				}

				b.AddASN1(cryptobyte_asn1.SEQUENCE, func(b *cryptobyte.Builder) {
					b.AddASN1(cryptobyte_asn1.Tag(2).ContextSpecific(), func(b *cryptobyte.Builder) {
						b.AddBytes([]byte(name))
					})
				})
			}

			for _, ipNet := range ips {
				encodedIPNet, err := ipAndMask(ipNet)
				if err != nil {
					return nil, err
				}
				b.AddASN1(cryptobyte_asn1.SEQUENCE, func(b *cryptobyte.Builder) {
					b.AddASN1(cryptobyte_asn1.Tag(7).ContextSpecific(), func(b *cryptobyte.Builder) {
						b.AddBytes(encodedIPNet)
					})
				})
			}

			for _, email := range emails {
				if err = isIA5String(email); err != nil {
					return nil, err
				}

				b.AddASN1(cryptobyte_asn1.SEQUENCE, func(b *cryptobyte.Builder) {
					b.AddASN1(cryptobyte_asn1.Tag(1).ContextSpecific(), func(b *cryptobyte.Builder) {
						b.AddBytes([]byte(email))
					})
				})
			}

			for _, uriDomain := range uriDomains {
				if err = isIA5String(uriDomain); err != nil {
					return nil, err
				}

				b.AddASN1(cryptobyte_asn1.SEQUENCE, func(b *cryptobyte.Builder) {
					b.AddASN1(cryptobyte_asn1.Tag(6).ContextSpecific(), func(b *cryptobyte.Builder) {
						b.AddBytes([]byte(uriDomain))
					})
				})
			}

			return b.Bytes()
		}

		permitted, err := serialiseConstraints(template.PermittedDNSDomains, template.PermittedIPRanges, template.PermittedEmailAddresses, template.PermittedURIDomains)
		if err != nil {
			return nil, err
		}

		excluded, err := serialiseConstraints(template.ExcludedDNSDomains, template.ExcludedIPRanges, template.ExcludedEmailAddresses, template.ExcludedURIDomains)
		if err != nil {
			return nil, err
		}

		var b cryptobyte.Builder
		b.AddASN1(cryptobyte_asn1.SEQUENCE, func(b *cryptobyte.Builder) {
			if len(permitted) > 0 {
				b.AddASN1(cryptobyte_asn1.Tag(0).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
					b.AddBytes(permitted)
				})
			}

			if len(excluded) > 0 {
				b.AddASN1(cryptobyte_asn1.Tag(1).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
					b.AddBytes(excluded)
				})
			}
		})

		ret[n].Value, err = b.Bytes()
		if err != nil {
			return nil, err
		}
		n++
	}

	if len(template.CRLDistributionPoints) > 0 &&
		!oidInExtensions(oidExtensionCRLDistributionPoints, template.ExtraExtensions) {
		ret[n].Id = oidExtensionCRLDistributionPoints

		var crlDp []distributionPoint
		for _, name := range template.CRLDistributionPoints {
			dp := distributionPoint{
				DistributionPoint: distributionPointName{
					FullName: []asn1.RawValue{
						{Tag: 6, Class: 2, Bytes: []byte(name)},
					},
				},
			}
			crlDp = append(crlDp, dp)
		}

		ret[n].Value, err = asn1.Marshal(crlDp)
		if err != nil {
			return
		}
		n++
	}

	// Adding another extension here? Remember to update the maximum number
	// of elements in the make() at the top of the function and the list of
	// template fields used in CreateCertificate documentation.

	return append(ret[:n], template.ExtraExtensions...), nil
}

func marshalKeyUsage(ku KeyUsage) (pkix.Extension, error) {
	ext := pkix.Extension{Id: oidExtensionKeyUsage, Critical: true}

	var a [2]byte
	a[0] = reverseBitsInAByte(byte(ku))
	a[1] = reverseBitsInAByte(byte(ku >> 8))

	l := 1
	if a[1] != 0 {
		l = 2
	}

	bitString := a[:l]
	var err error
	ext.Value, err = asn1.Marshal(asn1.BitString{Bytes: bitString, BitLength: asn1BitLength(bitString)})
	return ext, err
}

func marshalExtKeyUsage(extUsages []ExtKeyUsage, unknownUsages []asn1.ObjectIdentifier) (pkix.Extension, error) {
	ext := pkix.Extension{Id: oidExtensionExtendedKeyUsage}

	oids := make([]asn1.ObjectIdentifier, len(extUsages)+len(unknownUsages))
	for i, u := range extUsages {
		if oid, ok := oidFromExtKeyUsage(u); ok {
			oids[i] = oid
		} else {
			return ext, errors.New("x509: unknown extended key usage")
		}
	}

	copy(oids[len(extUsages):], unknownUsages)

	var err error
	ext.Value, err = asn1.Marshal(oids)
	return ext, err
}

func marshalBasicConstraints(isCA bool, maxPathLen int, maxPathLenZero bool) (pkix.Extension, error) {
	ext := pkix.Extension{Id: oidExtensionBasicConstraints, Critical: true}
	// Leaving MaxPathLen as zero indicates that no maximum path
	// length is desired, unless MaxPathLenZero is set. A value of
	// -1 causes encoding/asn1 to omit the value as desired.
	if maxPathLen == 0 && !maxPathLenZero {
		maxPathLen = -1
	}
	var err error
	ext.Value, err = asn1.Marshal(basicConstraints{isCA, maxPathLen})
	return ext, err
}

func marshalCertificatePolicies(policies []OID, policyIdentifiers []asn1.ObjectIdentifier) (pkix.Extension, error) {
	ext := pkix.Extension{Id: oidExtensionCertificatePolicies}

	b := cryptobyte.NewBuilder(make([]byte, 0, 128))
	b.AddASN1(cryptobyte_asn1.SEQUENCE, func(child *cryptobyte.Builder) {
		if x509usepolicies.Value() != "0" {
			x509usepolicies.IncNonDefault()
			for _, v := range policies {
				child.AddASN1(cryptobyte_asn1.SEQUENCE, func(child *cryptobyte.Builder) {
					child.AddASN1(cryptobyte_asn1.OBJECT_IDENTIFIER, func(child *cryptobyte.Builder) {
						if len(v.der) == 0 {
							child.SetError(errors.New("invalid policy object identifier"))
							return
						}
						child.AddBytes(v.der)
					})
				})
			}
		} else {
			for _, v := range policyIdentifiers {
				child.AddASN1(cryptobyte_asn1.SEQUENCE, func(child *cryptobyte.Builder) {
					child.AddASN1ObjectIdentifier(v)
				})
			}
		}
	})

	var err error
	ext.Value, err = b.Bytes()
	return ext, err
}

func buildCSRExtensions(template *CertificateRequest) ([]pkix.Extension, error) {
	var ret []pkix.Extension

	if (len(template.DNSNames) > 0 || len(template.EmailAddresses) > 0 || len(template.IPAddresses) > 0 || len(template.URIs) > 0) &&
		!oidInExtensions(oidExtensionSubjectAltName, template.ExtraExtensions) {
		sanBytes, err := marshalSANs(template.DNSNames, template.EmailAddresses, template.IPAddresses, template.URIs)
		if err != nil {
			return nil, err
		}

		ret = append(ret, pkix.Extension{
			Id:    oidExtensionSubjectAltName,
			Value: sanBytes,
		})
	}

	return append(ret, template.ExtraExtensions...), nil
}

func subjectBytes(cert *Certificate) ([]byte, error) {
	if len(cert.RawSubject) > 0 {
		return cert.RawSubject, nil
	}

	return asn1.Marshal(cert.Subject.ToRDNSequence())
}

// signingParamsForKey returns the signature algorithm and its Algorithm
// Identifier to use for signing, based on the key type. If sigAlgo is not zero
// then it overrides the default.
func signingParamsForKey(key crypto.Signer, sigAlgo SignatureAlgorithm) (SignatureAlgorithm, pkix.AlgorithmIdentifier, error) {
	var ai pkix.AlgorithmIdentifier
	var pubType PublicKeyAlgorithm
	var defaultAlgo SignatureAlgorithm

	switch pub := key.Public().(type) {
	case *rsa.PublicKey:
		pubType = RSA
		defaultAlgo = SHA256WithRSA

	case *ecdsa.PublicKey:
		pubType = ECDSA
		switch pub.Curve {
		case elliptic.P224(), elliptic.P256():
			defaultAlgo = ECDSAWithSHA256
		case elliptic.P384():
			defaultAlgo = ECDSAWithSHA384
		case elliptic.P521():
			defaultAlgo = ECDSAWithSHA512
		default:
			return 0, ai, errors.New("x509: unsupported elliptic curve")
		}

	case ed25519.PublicKey:
		pubType = Ed25519
		defaultAlgo = PureEd25519

	default:
		return 0, ai, errors.New("x509: only RSA, ECDSA and Ed25519 keys supported")
	}

	if sigAlgo == 0 {
		sigAlgo = defaultAlgo
	}

	for _, details := range signatureAlgorithmDetails {
		if details.algo == sigAlgo {
			if details.pubKeyAlgo != pubType {
				return 0, ai, errors.New("x509: requested SignatureAlgorithm does not match private key type")
			}
			if details.hash == crypto.MD5 {
				return 0, ai, errors.New("x509: signing with MD5 is not supported")
			}

			return sigAlgo, pkix.AlgorithmIdentifier{
				Algorithm:  details.oid,
				Parameters: details.params,
			}, nil
		}
	}

	return 0, ai, errors.New("x509: unknown SignatureAlgorithm")
}

func signTBS(tbs []byte, key crypto.Signer, sigAlg SignatureAlgorithm, rand io.Reader) ([]byte, error) {
	hashFunc := sigAlg.hashFunc()

	var signerOpts crypto.SignerOpts = hashFunc
	if sigAlg.isRSAPSS() {
		signerOpts = &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
			Hash:       hashFunc,
		}
	}

	signature, err := crypto.SignMessage(key, rand, tbs, signerOpts)
	if err != nil {
		return nil, err
	}

	// Check the signature to ensure the crypto.Signer behaved correctly.
	if err := checkSignature(sigAlg, tbs, signature, key.Public(), true); err != nil {
		return nil, fmt.Errorf("x509: signature returned by signer is invalid: %w", err)
	}

	return signature, nil
}

// emptyASN1Subject holds the DER of an empty subject, which is an empty
// SEQUENCE.
var emptyASN1Subject = []byte{0x30, 0}

// CreateCertificate creates a new X.509 v3 certificate based on a template.
// The following members of template are currently used:
//
//   - AuthorityKeyId
//   - BasicConstraintsValid
//   - CRLDistributionPoints
//   - DNSNames
//   - EmailAddresses
//   - ExcludedDNSDomains
//   - ExcludedEmailAddresses
//   - ExcludedIPRanges
//   - ExcludedURIDomains
//   - ExtKeyUsage
//   - ExtraExtensions
//   - IPAddresses
//   - IsCA
//   - IssuingCertificateURL
//   - KeyUsage
//   - MaxPathLen
//   - MaxPathLenZero
//   - NotAfter
//   - NotBefore
//   - OCSPServer
//   - PermittedDNSDomains
//   - PermittedDNSDomainsCritical
//   - PermittedEmailAddresses
//   - PermittedIPRanges
//   - PermittedURIDomains
//   - PolicyIdentifiers (see note below)
//   - Policies (see note below)
//   - SerialNumber
//   - SignatureAlgorithm
//   - Subject
//   - SubjectKeyId
//   - URIs
//   - UnknownExtKeyUsage
//
// The certificate is signed by parent. If parent is equal to template then the
// certificate is self-signed. The parameter pub is the public key of the
// certificate to be generated and priv is the private key of the signer.
//
// The returned slice is the certificate in DER encoding.
//
// The supported key types are *rsa.PublicKey, *ecdsa.PublicKey, and
// ed25519.PublicKey. pub must hold 1 of the 3, and priv must be a
// crypto.Signer or a crypto.MessageSigner with such a key.
//
// The AuthorityKeyId will be taken from the SubjectKeyId of parent, if any,
// unless the resulting certificate is self-signed. Otherwise the value from
// template will be used.
//
// If SubjectKeyId from template is empty and the template is a CA, SubjectKeyId
// will be generated from the hash of the public key.
//
// If template.SerialNumber is nil, a serial number will be generated which
// conforms to RFC 5280, Section 4.1.2.2 using entropy from rand.
//
// The PolicyIdentifier and Policies fields can both be used to marshal certificate
// policy OIDs. By default, only the Policies is marshaled, but if the
// GODEBUG setting "x509usepolicies" has the value "0", the PolicyIdentifiers field will
// be marshaled instead of the Policies field. This changed in Go 1.24. The Policies field can
// be used to marshal policy OIDs which have components that are larger than 31
// bits.
//
// IP addresses in IPAddresses which are in their IPv4-mapped IPv6 form will always be encoded
// in their IPv4 form.
func CreateCertificate(rand io.Reader, template, parent *Certificate, pub, priv any) ([]byte, error) {
	key, ok := priv.(crypto.Signer)
	if !ok {
		return nil, errors.New("x509: certificate private key does not implement crypto.Signer")
	}

	serialNumber := template.SerialNumber
	if serialNumber == nil {
		// Generate a serial number following RFC 5280, Section 4.1.2.2 if one
		// is not provided. The serial number must be positive and at most 20
		// octets *when encoded*.
		serialBytes := make([]byte, 20)
		if _, err := io.ReadFull(rand, serialBytes); err != nil {
			return nil, err
		}
		// If the top bit is set, the serial will be padded with a leading zero
		// byte during encoding, so that it's not interpreted as a negative
		// integer. This padding would make the serial 21 octets so we clear the
		// top bit to ensure the correct length in all cases.
		serialBytes[0] &= 0b0111_1111
		serialNumber = new(big.Int).SetBytes(serialBytes)
	}

	// RFC 5280 Section 4.1.2.2: serial number must be positive
	//
	// We _should_ also restrict serials to <= 20 octets, but it turns out a lot of people
	// get this wrong, in part because the encoding can itself alter the length of the
	// serial. For now we accept these non-conformant serials.
	if serialNumber.Sign() == -1 {
		return nil, errors.New("x509: serial number must be positive")
	}

	if template.BasicConstraintsValid && template.MaxPathLen < -1 {
		return nil, errors.New("x509: invalid MaxPathLen, must be greater or equal to -1")
	}

	if template.BasicConstraintsValid && !template.IsCA && template.MaxPathLen != -1 && (template.MaxPathLen != 0 || template.MaxPathLenZero) {
		return nil, errors.New("x509: only CAs are allowed to specify MaxPathLen")
	}

	signatureAlgorithm, algorithmIdentifier, err := signingParamsForKey(key, template.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	publicKeyBytes, publicKeyAlgorithm, err := marshalPublicKey(pub)
	if err != nil {
		return nil, err
	}
	if getPublicKeyAlgorithmFromOID(publicKeyAlgorithm.Algorithm) == UnknownPublicKeyAlgorithm {
		return nil, fmt.Errorf("x509: unsupported public key type: %T", pub)
	}

	asn1Issuer, err := subjectBytes(parent)
	if err != nil {
		return nil, err
	}

	asn1Subject, err := subjectBytes(template)
	if err != nil {
		return nil, err
	}

	authorityKeyId := template.AuthorityKeyId
	if !bytes.Equal(asn1Issuer, asn1Subject) && len(parent.SubjectKeyId) > 0 {
		authorityKeyId = parent.SubjectKeyId
	}

	subjectKeyId := template.SubjectKeyId
	if len(subjectKeyId) == 0 && template.IsCA {
		if x509sha256skid.Value() == "0" {
			x509sha256skid.IncNonDefault()
			// SubjectKeyId generated using method 1 in RFC 5280, Section 4.2.1.2:
			//   (1) The keyIdentifier is composed of the 160-bit SHA-1 hash of the
			//   value of the BIT STRING subjectPublicKey (excluding the tag,
			//   length, and number of unused bits).
			h := sha1.Sum(publicKeyBytes)
			subjectKeyId = h[:]
		} else {
			// SubjectKeyId generated using method 1 in RFC 7093, Section 2:
			//    1) The keyIdentifier is composed of the leftmost 160-bits of the
			//    SHA-256 hash of the value of the BIT STRING subjectPublicKey
			//    (excluding the tag, length, and number of unused bits).
			h := sha256.Sum256(publicKeyBytes)
			subjectKeyId = h[:20]
		}
	}

	// Check that the signer's public key matches the private key, if available.
	type privateKey interface {
		Equal(crypto.PublicKey) bool
	}
	if privPub, ok := key.Public().(privateKey); !ok {
		return nil, errors.New("x509: internal error: supported public key does not implement Equal")
	} else if parent.PublicKey != nil && !privPub.Equal(parent.PublicKey) {
		return nil, errors.New("x509: provided PrivateKey doesn't match parent's PublicKey")
	}

	extensions, err := buildCertExtensions(template, bytes.Equal(asn1Subject, emptyASN1Subject), authorityKeyId, subjectKeyId)
	if err != nil {
		return nil, err
	}

	encodedPublicKey := asn1.BitString{BitLength: len(publicKeyBytes) * 8, Bytes: publicKeyBytes}
	c := tbsCertificate{
		Version:            2,
		SerialNumber:       serialNumber,
		SignatureAlgorithm: algorithmIdentifier,
		Issuer:             asn1.RawValue{FullBytes: asn1Issuer},
		Validity:           validity{template.NotBefore.UTC(), template.NotAfter.UTC()},
		Subject:            asn1.RawValue{FullBytes: asn1Subject},
		PublicKey:          publicKeyInfo{nil, publicKeyAlgorithm, encodedPublicKey},
		Extensions:         extensions,
	}

	tbsCertContents, err := asn1.Marshal(c)
	if err != nil {
		return nil, err
	}
	// The write of the whole certificate reads the fields of c again. Go keeps
	// tbsCertContents in c.Raw to save that work, and the reflect library of
	// gart panics for a Bytes call over a defined slice type such as
	// asn1.RawContent. The 2 writes give the same DER, thus the signature
	// stands. Read docs/parity.md.

	signature, err := signTBS(tbsCertContents, key, signatureAlgorithm, rand)
	if err != nil {
		return nil, err
	}

	// certificate of the overlay holds the DER of the algorithm identifier in
	// its first field, thus the write names each field of it.
	return asn1.Marshal(certificate{
		TBSCertificate: c,
		SignatureAlgorithm: rawAlgorithmIdentifier{
			Algorithm:  algorithmIdentifier.Algorithm,
			Parameters: algorithmIdentifier.Parameters,
		},
		SignatureValue: asn1.BitString{Bytes: signature, BitLength: len(signature) * 8},
	})
}

var x509sha256skid = godebug.New("x509sha256skid")

// CreateCRL returns a DER encoded CRL, signed by this Certificate, that
// contains the given list of revoked certificates.
//
// Deprecated: this method does not generate an RFC 5280 conformant X.509 v2 CRL.
// To generate a standards compliant CRL, use [CreateRevocationList] instead.
func (c *Certificate) CreateCRL(rand io.Reader, priv any, revokedCerts []pkix.RevokedCertificate, now, expiry time.Time) (crlBytes []byte, err error) {
	key, ok := priv.(crypto.Signer)
	if !ok {
		return nil, errors.New("x509: certificate private key does not implement crypto.Signer")
	}

	signatureAlgorithm, algorithmIdentifier, err := signingParamsForKey(key, 0)
	if err != nil {
		return nil, err
	}

	// Force revocation times to UTC per RFC 5280.
	revokedCertsUTC := make([]pkix.RevokedCertificate, len(revokedCerts))
	for i, rc := range revokedCerts {
		rc.RevocationTime = rc.RevocationTime.UTC()
		revokedCertsUTC[i] = rc
	}

	tbsCertList := pkix.TBSCertificateList{
		Version:             1,
		Signature:           algorithmIdentifier,
		Issuer:              c.Subject.ToRDNSequence(),
		ThisUpdate:          now.UTC(),
		NextUpdate:          expiry.UTC(),
		RevokedCertificates: revokedCertsUTC,
	}

	// Authority Key Id
	if len(c.SubjectKeyId) > 0 {
		var aki pkix.Extension
		aki.Id = oidExtensionAuthorityKeyId
		aki.Value, err = asn1.Marshal(authKeyId{Id: c.SubjectKeyId})
		if err != nil {
			return nil, err
		}
		tbsCertList.Extensions = append(tbsCertList.Extensions, aki)
	}

	tbsCertListContents, err := asn1.Marshal(tbsCertList)
	if err != nil {
		return nil, err
	}
	// The write of the whole certificate reads the fields of c again. Go keeps
	// tbsCertContents in c.Raw to save that work, and the reflect library of
	// gart panics for a Bytes call over a defined slice type such as
	// asn1.RawContent. The 2 writes give the same DER, thus the signature
	// stands. Read docs/parity.md.

	signature, err := signTBS(tbsCertListContents, key, signatureAlgorithm, rand)
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(pkix.CertificateList{
		TBSCertList:        tbsCertList,
		SignatureAlgorithm: algorithmIdentifier,
		SignatureValue:     asn1.BitString{Bytes: signature, BitLength: len(signature) * 8},
	})
}

// newRawAttributes converts AttributeTypeAndValueSETs from a template
// CertificateRequest's Attributes into tbsCertificateRequest RawAttributes.
func newRawAttributes(attributes []pkix.AttributeTypeAndValueSET) ([]asn1.RawValue, error) {
	var rawAttributes []asn1.RawValue
	b, err := asn1.Marshal(attributes)
	if err != nil {
		return nil, err
	}
	rest, err := asn1.Unmarshal(b, &rawAttributes)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, errors.New("x509: failed to unmarshal raw CSR Attributes")
	}
	return rawAttributes, nil
}

// CreateCertificateRequest creates a new certificate request based on a
// template. The following members of template are used:
//
//   - SignatureAlgorithm
//   - Subject
//   - DNSNames
//   - EmailAddresses
//   - IPAddresses
//   - URIs
//   - ExtraExtensions
//   - Attributes (deprecated)
//
// priv is the private key to sign the CSR with, and the corresponding public
// key will be included in the CSR. It must implement crypto.Signer or
// crypto.MessageSigner and its Public() method must return a *rsa.PublicKey or
// a *ecdsa.PublicKey or a ed25519.PublicKey.
// (A *rsa.PrivateKey, *ecdsa.PrivateKey or ed25519.PrivateKey or
// each hold that method.)
//
// The returned slice is the certificate request in DER encoding.
func CreateCertificateRequest(rand io.Reader, template *CertificateRequest, priv any) (csr []byte, err error) {
	key, ok := priv.(crypto.Signer)
	if !ok {
		return nil, errors.New("x509: certificate private key does not implement crypto.Signer")
	}

	signatureAlgorithm, algorithmIdentifier, err := signingParamsForKey(key, template.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	var publicKeyBytes []byte
	var publicKeyAlgorithm pkix.AlgorithmIdentifier
	publicKeyBytes, publicKeyAlgorithm, err = marshalPublicKey(key.Public())
	if err != nil {
		return nil, err
	}

	extensions, err := buildCSRExtensions(template)
	if err != nil {
		return nil, err
	}

	// Make a copy of template.Attributes because we may alter it below.
	attributes := make([]pkix.AttributeTypeAndValueSET, 0, len(template.Attributes))
	for _, attr := range template.Attributes {
		values := make([][]pkix.AttributeTypeAndValue, len(attr.Value))
		copy(values, attr.Value)
		attributes = append(attributes, pkix.AttributeTypeAndValueSET{
			Type:  attr.Type,
			Value: values,
		})
	}

	extensionsAppended := false
	if len(extensions) > 0 {
		// Append the extensions to an existing attribute if possible.
		for _, atvSet := range attributes {
			if !atvSet.Type.Equal(oidExtensionRequest) || len(atvSet.Value) == 0 {
				continue
			}

			// specifiedExtensions contains all the extensions that we
			// found specified via template.Attributes.
			specifiedExtensions := make(map[string]bool)

			for _, atvs := range atvSet.Value {
				for _, atv := range atvs {
					specifiedExtensions[atv.Type.String()] = true
				}
			}

			newValue := make([]pkix.AttributeTypeAndValue, 0, len(atvSet.Value[0])+len(extensions))
			newValue = append(newValue, atvSet.Value[0]...)

			for _, e := range extensions {
				if specifiedExtensions[e.Id.String()] {
					// Attributes already contained a value for
					// this extension and it takes priority.
					continue
				}

				newValue = append(newValue, pkix.AttributeTypeAndValue{
					// There is no place for the critical
					// flag in an AttributeTypeAndValue.
					Type:  e.Id,
					Value: e.Value,
				})
			}

			atvSet.Value[0] = newValue
			extensionsAppended = true
			break
		}
	}

	rawAttributes, err := newRawAttributes(attributes)
	if err != nil {
		return nil, err
	}

	// If not included in attributes, add a new attribute for the
	// extensions.
	if len(extensions) > 0 && !extensionsAppended {
		attr := struct {
			Type  asn1.ObjectIdentifier
			Value [][]pkix.Extension `asn1:"set"`
		}{
			Type:  oidExtensionRequest,
			Value: [][]pkix.Extension{extensions},
		}

		b, err := asn1.Marshal(attr)
		if err != nil {
			return nil, errors.New("x509: failed to serialise extensions attribute: " + err.Error())
		}

		var rawValue asn1.RawValue
		if _, err := asn1.Unmarshal(b, &rawValue); err != nil {
			return nil, err
		}

		rawAttributes = append(rawAttributes, rawValue)
	}

	asn1Subject := template.RawSubject
	if len(asn1Subject) == 0 {
		asn1Subject, err = asn1.Marshal(template.Subject.ToRDNSequence())
		if err != nil {
			return nil, err
		}
	}

	tbsCSR := tbsCertificateRequest{
		Version: 0, // PKCS #10, RFC 2986
		Subject: asn1.RawValue{FullBytes: asn1Subject},
		PublicKey: publicKeyInfo{
			Algorithm: publicKeyAlgorithm,
			PublicKey: asn1.BitString{
				Bytes:     publicKeyBytes,
				BitLength: len(publicKeyBytes) * 8,
			},
		},
		RawAttributes: rawAttributes,
	}

	tbsCSRContents, err := asn1.Marshal(tbsCSR)
	if err != nil {
		return nil, err
	}
	// The write of the whole certificate reads the fields of c again. Go keeps
	// tbsCertContents in c.Raw to save that work, and the reflect library of
	// gart panics for a Bytes call over a defined slice type such as
	// asn1.RawContent. The 2 writes give the same DER, thus the signature
	// stands. Read docs/parity.md.

	signature, err := signTBS(tbsCSRContents, key, signatureAlgorithm, rand)
	if err != nil {
		return nil, err
	}

	cr := certificateRequest{}
	cr.TBSCSR = tbsCSR
	cr.SignatureAlgorithm.Algorithm = algorithmIdentifier.Algorithm
	cr.SignatureAlgorithm.Parameters = algorithmIdentifier.Parameters
	cr.SignatureValue = asn1.BitString{Bytes: signature, BitLength: len(signature) * 8}
	return asn1.Marshal(cr)
}

// CreateRevocationList creates a new X.509 v2 [Certificate] Revocation List,
// according to RFC 5280, based on template.
//
// The CRL is signed by priv which should be a crypto.Signer or
// crypto.MessageSigner associated with the public key in the issuer
// certificate.
//
// The issuer may not be nil, and the crlSign bit must be set in [KeyUsage] in
// order to use it as a CRL issuer.
//
// The issuer distinguished name CRL field and authority key identifier
// extension are populated using the issuer certificate. issuer must have
// SubjectKeyId set.
func CreateRevocationList(rand io.Reader, template *RevocationList, issuer *Certificate, priv crypto.Signer) ([]byte, error) {
	if template == nil {
		return nil, errors.New("x509: template can not be nil")
	}
	if issuer == nil {
		return nil, errors.New("x509: issuer can not be nil")
	}
	if (issuer.KeyUsage & KeyUsageCRLSign) == 0 {
		return nil, errors.New("x509: issuer must have the crlSign key usage bit set")
	}
	if len(issuer.SubjectKeyId) == 0 {
		return nil, errors.New("x509: issuer certificate doesn't contain a subject key identifier")
	}
	if template.NextUpdate.Before(template.ThisUpdate) {
		return nil, errors.New("x509: template.ThisUpdate is after template.NextUpdate")
	}
	if template.Number == nil {
		return nil, errors.New("x509: template contains nil Number field")
	}

	signatureAlgorithm, algorithmIdentifier, err := signingParamsForKey(priv, template.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	var revokedCerts []revokedCertificate
	// Only process the deprecated RevokedCertificates field if it is populated
	// and the new RevokedCertificateEntries field is not populated.
	if len(template.RevokedCertificates) > 0 && len(template.RevokedCertificateEntries) == 0 {
		// Force revocation times to UTC per RFC 5280.
		revokedCerts = make([]revokedCertificate, len(template.RevokedCertificates))
		for i, rc := range template.RevokedCertificates {
			revokedCerts[i] = revokedCertificate{
				SerialNumber:   rc.SerialNumber,
				RevocationTime: rc.RevocationTime.UTC(),
				Extensions:     rc.Extensions,
			}
		}
	} else {
		// Convert the ReasonCode field to a proper extension, and force revocation
		// times to UTC per RFC 5280.
		revokedCerts = make([]revokedCertificate, len(template.RevokedCertificateEntries))
		for i, rce := range template.RevokedCertificateEntries {
			if rce.SerialNumber == nil {
				return nil, errors.New("x509: template contains entry with nil SerialNumber field")
			}
			if rce.RevocationTime.IsZero() {
				return nil, errors.New("x509: template contains entry with zero RevocationTime field")
			}

			rc := revokedCertificate{
				SerialNumber:   rce.SerialNumber,
				RevocationTime: rce.RevocationTime.UTC(),
			}

			// Copy over any extra extensions, except for a Reason Code extension,
			// because we'll synthesize that ourselves to ensure it is correct.
			exts := make([]pkix.Extension, 0, len(rce.ExtraExtensions))
			for _, ext := range rce.ExtraExtensions {
				if ext.Id.Equal(oidExtensionReasonCode) {
					return nil, errors.New("x509: template contains entry with ReasonCode ExtraExtension; use ReasonCode field instead")
				}
				exts = append(exts, ext)
			}

			// Only add a reasonCode extension if the reason is non-zero, as per
			// RFC 5280 Section 5.3.1.
			if rce.ReasonCode != 0 {
				reasonBytes, err := asn1.Marshal(asn1.Enumerated(rce.ReasonCode))
				if err != nil {
					return nil, err
				}

				exts = append(exts, pkix.Extension{
					Id:    oidExtensionReasonCode,
					Value: reasonBytes,
				})
			}

			if len(exts) > 0 {
				rc.Extensions = exts
			}
			revokedCerts[i] = rc
		}
	}

	aki, err := asn1.Marshal(authKeyId{Id: issuer.SubjectKeyId})
	if err != nil {
		return nil, err
	}

	if numBytes := template.Number.Bytes(); len(numBytes) > 20 || (len(numBytes) == 20 && numBytes[0]&0x80 != 0) {
		return nil, errors.New("x509: CRL number exceeds 20 octets")
	}
	crlNum, err := asn1.Marshal(template.Number)
	if err != nil {
		return nil, err
	}

	// Correctly use the issuer's subject sequence if one is specified.
	issuerSubject, err := subjectBytes(issuer)
	if err != nil {
		return nil, err
	}

	tbsCertList := tbsCertificateList{
		Version: x509v2Version,
		Signature: rawAlgorithmIdentifier{
			Algorithm:  algorithmIdentifier.Algorithm,
			Parameters: algorithmIdentifier.Parameters,
		},
		Issuer:     asn1.RawValue{FullBytes: issuerSubject},
		ThisUpdate: template.ThisUpdate.UTC(),
		NextUpdate: template.NextUpdate.UTC(),
		Extensions: []pkix.Extension{
			{
				Id:    oidExtensionAuthorityKeyId,
				Value: aki,
			},
			{
				Id:    oidExtensionCRLNumber,
				Value: crlNum,
			},
		},
	}
	if len(revokedCerts) > 0 {
		tbsCertList.RevokedCertificates = revokedCerts
	}

	if len(template.ExtraExtensions) > 0 {
		tbsCertList.Extensions = append(tbsCertList.Extensions, template.ExtraExtensions...)
	}

	tbsCertListContents, err := asn1.Marshal(tbsCertList)
	if err != nil {
		return nil, err
	}

	// The write of the whole certificate reads the fields of c again. Go keeps
	// tbsCertContents in c.Raw to save that work, and the reflect library of
	// gart panics for a Bytes call over a defined slice type such as
	// asn1.RawContent. The 2 writes give the same DER, thus the signature
	// stands. Read docs/parity.md.

	signature, err := signTBS(tbsCertListContents, priv, signatureAlgorithm, rand)
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(certificateList{
		TBSCertList: tbsCertList,
		SignatureAlgorithm: rawAlgorithmIdentifier{
			Algorithm:  algorithmIdentifier.Algorithm,
			Parameters: algorithmIdentifier.Parameters,
		},
		SignatureValue: asn1.BitString{Bytes: signature, BitLength: len(signature) * 8},
	})
}

func (algo SignatureAlgorithm) hashFunc() crypto.Hash {
	for _, details := range signatureAlgorithmDetails {
		if details.algo == algo {
			return details.hash
		}
	}
	return crypto.Hash(0)
}
