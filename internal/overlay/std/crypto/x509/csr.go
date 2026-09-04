//go:build dart

package x509

import (
	"errors"
	"net"
	"net/url"

	"crypto/x509/pkix"
	"encoding/asn1"
)

// This file reads a PKCS #10 certificate request. RFC 2986 names the shape.
//
// Go reads a request with golang.org/x/crypto/cryptobyte, and that package
// stays outside the surface. The parse below reads the same bytes with
// encoding/asn1, and it gives the same values and the same errors.
//
//	CertificationRequest ::= SEQUENCE {
//	     certificationRequestInfo  CertificationRequestInfo,
//	     signatureAlgorithm        AlgorithmIdentifier,
//	     signature                 BIT STRING }
//
//	CertificationRequestInfo ::= SEQUENCE {
//	     version        INTEGER,
//	     subject        Name,
//	     subjectPKInfo  SubjectPublicKeyInfo,
//	     attributes     [0] IMPLICIT SET OF Attribute }

type tbsCertificateRequest struct {
	Raw           asn1.RawContent
	Version       int
	Subject       asn1.RawValue
	PublicKey     publicKeyInfo
	RawAttributes []asn1.RawValue `asn1:"tag:0"`
}

type certificateRequest struct {
	Raw                asn1.RawContent
	TBSCSR             tbsCertificateRequest
	SignatureAlgorithm rawAlgorithmIdentifier
	SignatureValue     asn1.BitString
}

// pkcs10Attribute holds 1 attribute of a request. RFC 2986, 4.1.
type pkcs10Attribute struct {
	Id     asn1.ObjectIdentifier
	Values []asn1.RawValue `asn1:"set"`
}

// oidExtensionRequest names the extensionRequest attribute of PKCS #9. That
// attribute holds each extension that the request asks for.
var oidExtensionRequest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 14}

// A CertificateRequest holds 1 PKCS #10 certificate request.
type CertificateRequest struct {
	// Raw holds the whole DER of the request.
	Raw []byte
	// RawTBSCertificateRequest holds the DER of the certificationRequestInfo.
	// The signature covers that part.
	RawTBSCertificateRequest []byte
	// RawSubjectPublicKeyInfo holds the DER of the SubjectPublicKeyInfo.
	RawSubjectPublicKeyInfo []byte
	// RawSubject holds the DER of the subject.
	RawSubject []byte
	// RawSignatureAlgorithm holds the DER of the AlgorithmIdentifier.
	RawSignatureAlgorithm []byte

	Version            int
	Signature          []byte
	SignatureAlgorithm SignatureAlgorithm

	PublicKeyAlgorithm PublicKeyAlgorithm
	// PublicKey holds the key of the subject. It is a *rsa.PublicKey, a
	// *dsa.PublicKey, a *ecdsa.PublicKey, an ed25519.PublicKey, or a
	// *ecdh.PublicKey for X25519.
	PublicKey any

	Subject pkix.Name

	// Attributes holds each attribute of the request that reads as a
	// pkix.AttributeTypeAndValueSET. An attribute of another shape, such as
	// challengePassword, drops out.
	//
	// Deprecated: read Extensions for the requestedExtensions attribute.
	Attributes []pkix.AttributeTypeAndValueSET

	// Extensions holds each requested extension in its raw form. A program
	// reads it for an extension that this package does not parse.
	Extensions []pkix.Extension

	// ExtraExtensions holds each extension that CreateCertificateRequest puts
	// into a new request. A parse leaves the field empty, exactly as in Go.
	// The overlay holds no CreateCertificateRequest, thus no code writes the
	// field here.
	ExtraExtensions []pkix.Extension

	// The subject alternative names.
	DNSNames       []string
	EmailAddresses []string
	IPAddresses    []net.IP
	URIs           []*url.URL
}

// ParseCertificateRequest reads 1 certificate request of the DER data.
func ParseCertificateRequest(asn1Data []byte) (*CertificateRequest, error) {
	var csr certificateRequest
	rest, err := asn1.Unmarshal(asn1Data, &csr)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, asn1.SyntaxError{Msg: "trailing data"}
	}
	return buildCertificateRequest(&csr)
}

func buildCertificateRequest(in *certificateRequest) (*CertificateRequest, error) {
	out := new(CertificateRequest)
	out.Raw = in.Raw
	out.RawTBSCertificateRequest = in.TBSCSR.Raw
	out.RawSubjectPublicKeyInfo = in.TBSCSR.PublicKey.Raw
	out.RawSubject = in.TBSCSR.Subject.FullBytes
	out.RawSignatureAlgorithm = in.SignatureAlgorithm.Raw

	out.Signature = in.SignatureValue.RightAlign()
	out.SignatureAlgorithm = getSignatureAlgorithmFromAI(in.SignatureAlgorithm.ident())
	out.PublicKeyAlgorithm = getPublicKeyAlgorithmFromOID(in.TBSCSR.PublicKey.Algorithm.Algorithm)
	if out.PublicKeyAlgorithm != UnknownPublicKeyAlgorithm {
		key, err := parsePublicKey(&in.TBSCSR.PublicKey)
		if err != nil {
			return nil, err
		}
		out.PublicKey = key
	}

	out.Version = in.TBSCSR.Version
	out.Attributes = parseRawAttributes(in.TBSCSR.RawAttributes)

	var subject pkix.RDNSequence
	if rest, err := asn1.Unmarshal(in.TBSCSR.Subject.FullBytes, &subject); err != nil {
		return nil, err
	} else if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after X.509 subject")
	}
	out.Subject.FillFromRDNSequence(&subject)

	extensions, err := parseCSRExtensions(in.TBSCSR.RawAttributes)
	if err != nil {
		return nil, err
	}
	out.Extensions = extensions

	for _, extension := range out.Extensions {
		if !extension.Id.Equal(oidExtensionSubjectAltName) {
			continue
		}
		if err := parseCSRSANExtension(out, extension.Value); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// parseCSRSANExtension reads the subject alternative names of a request.
// parseSANExtension takes a *Certificate, thus the read passes 1 scratch
// certificate, and then it moves each list into the request.
func parseCSRSANExtension(out *CertificateRequest, value []byte) error {
	scratch := new(Certificate)
	if err := parseSANExtension(scratch, value); err != nil {
		return err
	}
	out.DNSNames = append(out.DNSNames, scratch.DNSNames...)
	out.EmailAddresses = append(out.EmailAddresses, scratch.EmailAddresses...)
	out.IPAddresses = append(out.IPAddresses, scratch.IPAddresses...)
	out.URIs = append(out.URIs, scratch.URIs...)
	return nil
}

// parseRawAttributes gives each raw attribute that holds the shape of a
// pkix.AttributeTypeAndValueSET. An attribute of another shape drops out.
func parseRawAttributes(rawAttributes []asn1.RawValue) []pkix.AttributeTypeAndValueSET {
	var attributes []pkix.AttributeTypeAndValueSET
	for _, rawAttr := range rawAttributes {
		var attr pkix.AttributeTypeAndValueSET
		rest, err := asn1.Unmarshal(rawAttr.FullBytes, &attr)
		if err == nil && len(rest) == 0 {
			attributes = append(attributes, attr)
		}
	}
	return attributes
}

// parseCSRExtensions gives each extension of the extensionRequest attribute.
// An attribute that the parse does not understand drops out, and a repeated
// extension gives an error.
func parseCSRExtensions(rawAttributes []asn1.RawValue) ([]pkix.Extension, error) {
	var ret []pkix.Extension
	requestedExts := make(map[string]bool)
	for _, rawAttr := range rawAttributes {
		var attr pkcs10Attribute
		rest, err := asn1.Unmarshal(rawAttr.FullBytes, &attr)
		if err != nil || len(rest) != 0 || len(attr.Values) == 0 {
			continue
		}
		if !attr.Id.Equal(oidExtensionRequest) {
			continue
		}

		var extensions []pkix.Extension
		if _, err := asn1.Unmarshal(attr.Values[0].FullBytes, &extensions); err != nil {
			return nil, err
		}
		for _, ext := range extensions {
			oidStr := ext.Id.String()
			if requestedExts[oidStr] {
				return nil, errors.New("x509: certificate request contains duplicate requested extensions")
			}
			requestedExts[oidStr] = true
		}
		ret = append(ret, extensions...)
	}
	return ret, nil
}

// CheckSignature reports whether the signature on c is valid.
func (c *CertificateRequest) CheckSignature() error {
	return checkSignature(c.SignatureAlgorithm, c.RawTBSCertificateRequest, c.Signature, c.PublicKey, true)
}
