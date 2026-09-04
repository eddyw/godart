//go:build dart

package x509

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"time"

	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
)

// This file reads a certificate revocation list. RFC 5280, 5.1 names the
// shape.
//
// Go reads a list with golang.org/x/crypto/cryptobyte, and that package stays
// outside the surface. The parse below reads the same bytes with
// encoding/asn1, and it gives the same values and the same errors.
//
//	CertificateList ::= SEQUENCE {
//	     tbsCertList          TBSCertList,
//	     signatureAlgorithm   AlgorithmIdentifier,
//	     signatureValue       BIT STRING }
//
//	TBSCertList ::= SEQUENCE {
//	     version              Version OPTIONAL,
//	     signature            AlgorithmIdentifier,
//	     issuer               Name,
//	     thisUpdate           Time,
//	     nextUpdate           Time OPTIONAL,
//	     revokedCertificates  SEQUENCE OF SEQUENCE {
//	          userCertificate     CertificateSerialNumber,
//	          revocationDate      Time,
//	          crlEntryExtensions  Extensions OPTIONAL } OPTIONAL,
//	     crlExtensions        [0] EXPLICIT Extensions OPTIONAL }
//
// The overlay holds the parse path alone. RevocationList.CheckSignatureFrom,
// Certificate.CheckCRLSignature, CreateRevocationList, and
// Certificate.CreateCRL each need an RSA key or an ECDSA key, thus each stays
// out. The package doc names the cause.

// The X.509 standard counts a version name from 1, and it writes the number
// on the wire from 0, thus the number of an X.509 v2 list is 1.
const x509v2Version = 1

type certificateList struct {
	Raw                asn1.RawContent
	TBSCertList        tbsCertificateList
	SignatureAlgorithm rawAlgorithmIdentifier
	SignatureValue     asn1.BitString
}

// tbsCertificateList holds the same fields as pkix.TBSCertificateList. It
// keeps the issuer as a raw value, exactly as tbsCertificate does, and the
// default of the version marks an absent field.
type tbsCertificateList struct {
	Raw                 asn1.RawContent
	Version             int `asn1:"optional,default:-1"`
	Signature           rawAlgorithmIdentifier
	Issuer              asn1.RawValue
	ThisUpdate          time.Time
	NextUpdate          time.Time            `asn1:"optional"`
	RevokedCertificates []revokedCertificate `asn1:"optional"`
	Extensions          []pkix.Extension     `asn1:"tag:0,optional,explicit"`
}

// revokedCertificate holds the same fields as pkix.RevokedCertificate, and it
// also holds the DER of the entry. RevocationListEntry.Raw reads that DER.
type revokedCertificate struct {
	Raw            asn1.RawContent
	SerialNumber   *big.Int
	RevocationTime time.Time
	Extensions     []pkix.Extension `asn1:"optional"`
}

var (
	oidExtensionCRLNumber  = asn1.ObjectIdentifier{2, 5, 29, 20}
	oidExtensionReasonCode = asn1.ObjectIdentifier{2, 5, 29, 21}
)

// A RevocationListEntry holds 1 entry of the revokedCertificates sequence of
// a certificate revocation list.
type RevocationListEntry struct {
	// Raw holds the DER of the entry. A parse fills the field.
	Raw []byte

	// SerialNumber holds the serial number of the revoked certificate.
	SerialNumber *big.Int
	// RevocationTime holds the moment of the revocation.
	RevocationTime time.Time
	// ReasonCode holds the reason of the revocation. RFC 5280, 5.3.1 names
	// the value of each reason.
	//
	// A value of 0 holds 2 meanings. The reasonCode extension can stand
	// absent, and the default reason of an absent extension is 0. The
	// extension can also stand present with a value of 0. The DER rules
	// reject the second form, and a real list holds it anyway.
	ReasonCode int

	// Extensions holds each extension of the entry in its raw form. A parse
	// fills the field.
	Extensions []pkix.Extension
	// ExtraExtensions holds each extension that a writer puts into a new
	// entry. A parse leaves the field empty, exactly as in Go. The overlay
	// holds no CreateRevocationList, thus no code writes the field here.
	ExtraExtensions []pkix.Extension
}

// A RevocationList holds 1 certificate revocation list. RFC 5280 names the
// shape.
//
// The overlay holds no CheckSignatureFrom method, because a check of the
// signature needs crypto/rsa or crypto/ecdsa, and each package stays outside
// the surface.
type RevocationList struct {
	// Raw holds the whole DER of the list.
	Raw []byte
	// RawTBSRevocationList holds the DER of the tbsCertList. The signature
	// covers that part.
	RawTBSRevocationList []byte
	// RawIssuer holds the DER of the issuer.
	RawIssuer []byte
	// RawSignatureAlgorithm holds the DER of the AlgorithmIdentifier.
	RawSignatureAlgorithm []byte

	// Issuer holds the distinguished name of the certificate that signs the
	// list.
	Issuer pkix.Name
	// AuthorityKeyId names the public key of the certificate that signs the
	// list. A parse reads it from the authorityKeyIdentifier extension.
	AuthorityKeyId []byte

	Signature []byte
	// SignatureAlgorithm names the algorithm of the signature.
	SignatureAlgorithm SignatureAlgorithm

	// RevokedCertificateEntries holds the revokedCertificates sequence of the
	// list.
	RevokedCertificateEntries []RevocationListEntry

	// RevokedCertificates holds the same sequence in the old shape. A parse
	// fills it beside RevokedCertificateEntries.
	//
	// Deprecated: read RevokedCertificateEntries.
	RevokedCertificates []pkix.RevokedCertificate

	// Number holds the cRLNumber extension of the list. That number counts up
	// for each list of 1 issuer and 1 scope.
	Number *big.Int

	// ThisUpdate holds the moment of the issue of the list.
	ThisUpdate time.Time
	// NextUpdate holds the moment of the next list. The field stays at the
	// zero time when the list holds no such moment.
	NextUpdate time.Time

	// Extensions holds each extension of the list in its raw form.
	Extensions []pkix.Extension
	// ExtraExtensions holds each extension that a writer puts into a new
	// list. A parse leaves the field empty, exactly as in Go. The overlay
	// holds no CreateRevocationList, thus no code writes the field here.
	ExtraExtensions []pkix.Extension
}

// ParseRevocationList reads 1 X.509 v2 certificate revocation list of the DER
// data.
func ParseRevocationList(der []byte) (*RevocationList, error) {
	var cl certificateList
	// Go reads the first element alone, and it drops each byte after that
	// element. The overlay copies that rule, thus it takes the same input.
	if _, err := asn1.Unmarshal(der, &cl); err != nil {
		return nil, err
	}
	return buildRevocationList(&cl)
}

func buildRevocationList(in *certificateList) (*RevocationList, error) {
	if in.TBSCertList.Version == -1 {
		return nil, errors.New("x509: unsupported crl version")
	}
	if in.TBSCertList.Version != x509v2Version {
		return nil, fmt.Errorf("x509: unsupported crl version: %d", in.TBSCertList.Version)
	}
	if !bytes.Equal(in.TBSCertList.Signature.Raw, in.SignatureAlgorithm.Raw) {
		return nil, errors.New("x509: inner and outer signature algorithm identifiers don't match")
	}

	out := new(RevocationList)
	out.Raw = in.Raw
	out.RawTBSRevocationList = in.TBSCertList.Raw
	out.RawIssuer = in.TBSCertList.Issuer.FullBytes
	out.RawSignatureAlgorithm = in.TBSCertList.Signature.Raw

	out.Signature = in.SignatureValue.RightAlign()
	out.SignatureAlgorithm = getSignatureAlgorithmFromAI(in.TBSCertList.Signature.ident())

	var issuer pkix.RDNSequence
	if rest, err := asn1.Unmarshal(in.TBSCertList.Issuer.FullBytes, &issuer); err != nil {
		return nil, err
	} else if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after X.509 CRL issuer")
	}
	out.Issuer.FillFromRDNSequence(&issuer)

	out.ThisUpdate = in.TBSCertList.ThisUpdate
	out.NextUpdate = in.TBSCertList.NextUpdate

	for _, rc := range in.TBSCertList.RevokedCertificates {
		entry, err := buildRevocationListEntry(rc)
		if err != nil {
			return nil, err
		}
		out.RevokedCertificateEntries = append(out.RevokedCertificateEntries, entry)

		var deprecated pkix.RevokedCertificate
		deprecated.SerialNumber = entry.SerialNumber
		deprecated.RevocationTime = entry.RevocationTime
		deprecated.Extensions = entry.Extensions
		out.RevokedCertificates = append(out.RevokedCertificates, deprecated)
	}

	out.Extensions = in.TBSCertList.Extensions
	for _, ext := range out.Extensions {
		switch {
		case ext.Id.Equal(oidExtensionAuthorityKeyId):
			id, err := parseCRLAuthorityKeyId(ext)
			if err != nil {
				return nil, err
			}
			out.AuthorityKeyId = id
		case ext.Id.Equal(oidExtensionCRLNumber):
			var number *big.Int
			rest, err := asn1.Unmarshal(ext.Value, &number)
			if err != nil || len(rest) != 0 {
				return nil, errors.New("x509: malformed crl number")
			}
			out.Number = number
		}
	}
	return out, nil
}

func buildRevocationListEntry(rc revokedCertificate) (RevocationListEntry, error) {
	var entry RevocationListEntry
	entry.Raw = rc.Raw
	entry.SerialNumber = rc.SerialNumber
	entry.RevocationTime = rc.RevocationTime
	entry.Extensions = rc.Extensions
	for _, ext := range rc.Extensions {
		if !ext.Id.Equal(oidExtensionReasonCode) {
			continue
		}
		// RFC 5280, 5.3.1 names an ENUMERATED value.
		var reason asn1.Enumerated
		rest, err := asn1.Unmarshal(ext.Value, &reason)
		if err != nil || len(rest) != 0 {
			return entry, errors.New("x509: malformed reasonCode extension")
		}
		entry.ReasonCode = int(reason)
	}
	return entry, nil
}

// parseCRLAuthorityKeyId gives the key identifier of the extension. RFC 5280,
// 4.2.1.1 orders a CA to mark the extension as not critical.
func parseCRLAuthorityKeyId(e pkix.Extension) ([]byte, error) {
	if e.Critical {
		return nil, errors.New("x509: authority key identifier incorrectly marked critical")
	}
	var a authKeyId
	rest, err := asn1.Unmarshal(e.Value, &a)
	if err != nil || len(rest) != 0 {
		return nil, errors.New("x509: invalid authority key identifier")
	}
	return a.Id, nil
}

// pemCRLPrefix names the first bytes of a PEM revocation list.
var pemCRLPrefix = []byte("-----BEGIN X509 CRL")

// pemCRLType names the type of a PEM block that holds a revocation list.
var pemCRLType = "X509 CRL"

// ParseCRL reads a revocation list of the bytes. A PEM list often stands
// where a DER list belongs, thus the function reads a PEM block too. The PEM
// block must stand first, with no byte before it.
//
// Deprecated: use ParseRevocationList.
func ParseCRL(crlBytes []byte) (*pkix.CertificateList, error) {
	if bytes.HasPrefix(crlBytes, pemCRLPrefix) {
		block, _ := pem.Decode(crlBytes)
		if block != nil && block.Type == pemCRLType {
			crlBytes = block.Bytes
		}
	}
	return ParseDERCRL(crlBytes)
}

// ParseDERCRL reads a DER revocation list of the bytes.
//
// Deprecated: use ParseRevocationList.
func ParseDERCRL(derBytes []byte) (*pkix.CertificateList, error) {
	certList := new(pkix.CertificateList)
	if rest, err := asn1.Unmarshal(derBytes, certList); err != nil {
		return nil, err
	} else if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after CRL")
	}
	return certList, nil
}

// CheckSignatureFrom reports whether the signature on rl is a valid signature
// of parent.
func (rl *RevocationList) CheckSignatureFrom(parent *Certificate) error {
	if parent.Version == 3 && !parent.BasicConstraintsValid ||
		parent.BasicConstraintsValid && !parent.IsCA {
		return ConstraintViolationError{}
	}

	if parent.KeyUsage != 0 && parent.KeyUsage&KeyUsageCRLSign == 0 {
		return ConstraintViolationError{}
	}

	if parent.PublicKeyAlgorithm == UnknownPublicKeyAlgorithm {
		return ErrUnsupportedAlgorithm
	}

	return parent.CheckSignature(rl.SignatureAlgorithm, rl.RawTBSRevocationList, rl.Signature)
}
