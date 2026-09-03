//go:build dart

package x509

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"

	"crypto/x509/pkix"
	"encoding/asn1"
)

// This file reads the name constraints extension of a certificate. RFC 5280,
// 4.2.1.10.
//
// The work reads bytes alone, and it holds no crypto. Go reads the extension
// with golang.org/x/crypto/cryptobyte, and that package stays outside the
// surface. The parse below reads the same bytes with encoding/asn1, and it
// gives the same values and the same errors.
//
// The overlay runs no chain check, thus a constraint below rejects no
// certificate here. Certificate.Verify owns that work in Go, and the secure
// socket of Dart owns it in this project.
//
//	NameConstraints ::= SEQUENCE {
//	     permittedSubtrees       [0]     GeneralSubtrees OPTIONAL,
//	     excludedSubtrees        [1]     GeneralSubtrees OPTIONAL }
//
//	GeneralSubtrees ::= SEQUENCE SIZE (1..MAX) OF GeneralSubtree
//
//	GeneralSubtree ::= SEQUENCE {
//	     base                    GeneralName,
//	     minimum         [0]     BaseDistance DEFAULT 0,
//	     maximum         [1]     BaseDistance OPTIONAL }

// errBadNameConstraints reports an extension that holds no valid form.
var errBadNameConstraints = errors.New("x509: invalid NameConstraints extension")

// parseNameConstraints reads the extension into the certificate. It reports
// whether a subtree named a form that the parse does not read.
func parseNameConstraints(out *Certificate, e pkix.Extension) (bool, error) {
	var top asn1.RawValue
	rest, err := asn1.Unmarshal(e.Value, &top)
	if err != nil {
		return false, errBadNameConstraints
	}
	if len(rest) != 0 || !top.IsCompound ||
		top.Class != asn1.ClassUniversal || top.Tag != asn1.TagSequence {
		return false, errBadNameConstraints
	}

	var permitted, excluded []byte
	var havePermitted, haveExcluded bool
	body := top.Bytes
	last := -1
	for len(body) > 0 {
		var v asn1.RawValue
		body, err = asn1.Unmarshal(body, &v)
		if err != nil {
			return false, errBadNameConstraints
		}
		if v.Class != asn1.ClassContextSpecific || !v.IsCompound {
			return false, errBadNameConstraints
		}
		// Go reads the 2 items in order, thus a repeat and a wrong order each
		// give an error.
		if v.Tag != 0 && v.Tag != 1 || int(v.Tag) <= last {
			return false, errBadNameConstraints
		}
		last = int(v.Tag)
		if v.Tag == 0 {
			permitted, havePermitted = v.Bytes, true
			continue
		}
		excluded, haveExcluded = v.Bytes, true
	}

	// RFC 5280, 4.2.1.10 names 1 of the 2 fields as required.
	if !havePermitted && !haveExcluded || len(permitted) == 0 && len(excluded) == 0 {
		return false, errors.New("x509: empty name constraints extension")
	}

	unhandled := false
	dns, ips, emails, uris, skipped, err := nameConstraintValues(permitted)
	if err != nil {
		return false, err
	}
	unhandled = unhandled || skipped
	out.PermittedDNSDomains = dns
	out.PermittedIPRanges = ips
	out.PermittedEmailAddresses = emails
	out.PermittedURIDomains = uris

	dns, ips, emails, uris, skipped, err = nameConstraintValues(excluded)
	if err != nil {
		return false, err
	}
	unhandled = unhandled || skipped
	out.ExcludedDNSDomains = dns
	out.ExcludedIPRanges = ips
	out.ExcludedEmailAddresses = emails
	out.ExcludedURIDomains = uris

	out.PermittedDNSDomainsCritical = e.Critical
	return unhandled, nil
}

// nameConstraintValues reads the subtrees of 1 side of the extension. The
// last result of the 6 reports whether a subtree named a form that the parse
// does not read.
func nameConstraintValues(subtrees []byte) ([]string, []*net.IPNet, []string, []string, bool, error) {
	var dnsNames []string
	var ips []*net.IPNet
	var emails []string
	var uriDomains []string
	unhandled := false
	var err error
	for len(subtrees) > 0 {
		var seq asn1.RawValue
		subtrees, err = asn1.Unmarshal(subtrees, &seq)
		if err != nil || !seq.IsCompound ||
			seq.Class != asn1.ClassUniversal || seq.Tag != asn1.TagSequence {
			return nil, nil, nil, nil, false, errBadNameConstraints
		}
		// A GeneralSubtree holds the base and 2 optional distances. Go reads
		// the base alone, and it drops the 2 distances.
		var v asn1.RawValue
		if _, err = asn1.Unmarshal(seq.Bytes, &v); err != nil {
			return nil, nil, nil, nil, false, errBadNameConstraints
		}
		if v.Class != asn1.ClassContextSpecific || v.IsCompound {
			unhandled = true
			continue
		}
		switch v.Tag {
		case 2:
			// dNSName
			domain := string(v.Bytes)
			if err := isIA5String(domain); err != nil {
				return nil, nil, nil, nil, false, err
			}
			if !domainNameValid(domain, true) {
				return nil, nil, nil, nil, false,
					fmt.Errorf("x509: failed to parse dnsName constraint %q", domain)
			}
			dnsNames = append(dnsNames, domain)
		case 7:
			// iPAddress. The value holds the address and the mask.
			network, err := ipConstraint(v.Bytes)
			if err != nil {
				return nil, nil, nil, nil, false, err
			}
			ips = append(ips, network)
		case 1:
			// rfc822Name
			constraint := string(v.Bytes)
			if err := isIA5String(constraint); err != nil {
				return nil, nil, nil, nil, false, err
			}
			// A constraint that holds an at sign names 1 exact mailbox.
			if strings.Contains(constraint, "@") {
				if !validRFC2821Mailbox(constraint) {
					return nil, nil, nil, nil, false,
						fmt.Errorf("x509: failed to parse rfc822Name constraint %q", constraint)
				}
			} else if !domainNameValid(constraint, true) {
				return nil, nil, nil, nil, false,
					fmt.Errorf("x509: failed to parse rfc822Name constraint %q", constraint)
			}
			emails = append(emails, constraint)
		case 6:
			// uniformResourceIdentifier
			domain := string(v.Bytes)
			if err := isIA5String(domain); err != nil {
				return nil, nil, nil, nil, false, err
			}
			if net.ParseIP(domain) != nil {
				return nil, nil, nil, nil, false,
					fmt.Errorf("x509: failed to parse URI constraint %q: cannot be IP address", domain)
			}
			if !domainNameValid(domain, true) {
				return nil, nil, nil, nil, false,
					fmt.Errorf("x509: failed to parse URI constraint %q", domain)
			}
			uriDomains = append(uriDomains, domain)
		default:
			unhandled = true
		}
	}
	return dnsNames, ips, emails, uriDomains, unhandled, nil
}

// ipConstraint reads the address and the mask of an iPAddress constraint. The
// value holds 8 bytes for IPv4 and 32 bytes for IPv6.
func ipConstraint(value []byte) (*net.IPNet, error) {
	var ip, mask []byte
	switch len(value) {
	case 8:
		ip, mask = value[:4], value[4:]
	case 32:
		ip, mask = value[:16], value[16:]
	default:
		return nil, errors.New("x509: IP constraint contained value of length " +
			strconv.Itoa(len(value)))
	}
	if !isValidIPMask(mask) {
		return nil, fmt.Errorf("x509: IP constraint contained invalid mask %x", mask)
	}
	if len(ip) == net.IPv6len && net.IP(ip).To4() != nil {
		return nil, errors.New("x509: IP constraint contained IPv4-mapped IPv6 address")
	}
	return &net.IPNet{IP: net.IP(ip), Mask: net.IPMask(mask)}, nil
}

// isValidIPMask reports whether the mask holds a block of ones and then a
// block of zeros.
func isValidIPMask(mask []byte) bool {
	seenZero := false
	for _, b := range mask {
		if seenZero {
			if b != 0 {
				return false
			}
			continue
		}
		switch b {
		case 0x00, 0x80, 0xc0, 0xe0, 0xf0, 0xf8, 0xfc, 0xfe:
			seenZero = true
		case 0xff:
		default:
			return false
		}
	}
	return true
}

// isIA5String reports an error for a text that holds a rune outside ASCII.
// RFC 5280 limits an IA5String to the ASCII characters.
func isIA5String(s string) error {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return fmt.Errorf("x509: %q cannot be encoded as an IA5String", s)
		}
	}
	return nil
}

// domainNameValid reports whether the text holds a domain name. A constraint
// takes a leading period, and a name of a certificate does not.
//
// Go holds the same body, and issue 75835 of Go names the checks that the
// body drops.
func domainNameValid(s string, constraint bool) bool {
	if len(s) == 0 {
		return true
	}
	// A trailing period names the root, and no name of a certificate and no
	// constraint takes that form.
	if s[len(s)-1] == '.' {
		return false
	}
	lastDot := -1
	if constraint && s[0] == '.' {
		s = s[1:]
	}
	for i := 0; i <= len(s); i++ {
		if i < len(s) && (s[i] < 33 || s[i] > 126) {
			return false
		}
		if i == len(s) || s[i] == '.' {
			labelLen := i
			if lastDot >= 0 {
				labelLen -= lastDot + 1
			}
			if labelLen == 0 {
				return false
			}
			lastDot = i
		}
	}
	return true
}

// validRFC2821Mailbox reports whether the text holds a mailbox of RFC 2821,
// section 4.1.2. RFC 5280, 4.2.1.6 names that form for an rfc822Name.
//
// Go gives the 2 parts of the mailbox. No reader of this overlay needs them,
// thus the body reports the validity alone.
func validRFC2821Mailbox(in string) bool {
	if len(in) == 0 {
		return false
	}
	localPartBytes := make([]byte, 0, len(in)/2)
	if in[0] == '"' {
		// Quoted-string = DQUOTE *qcontent DQUOTE
		in = in[1:]
		closed := false
		for !closed {
			if len(in) == 0 {
				return false
			}
			c := in[0]
			in = in[1:]
			switch {
			case c == '"':
				closed = true
			case c == '\\':
				// quoted-pair
				if len(in) == 0 {
					return false
				}
				if in[0] == 11 || in[0] == 12 ||
					(1 <= in[0] && in[0] <= 9) || (14 <= in[0] && in[0] <= 127) {
					localPartBytes = append(localPartBytes, in[0])
					in = in[1:]
					continue
				}
				return false
			case c == 11 || c == 12 ||
				// The form of RFC 2821 holds no space, and RFC 3696 gives an
				// example that holds 1. Go takes the space, and so does this
				// body.
				c == 32 || c == 33 || c == 127 ||
				(1 <= c && c <= 8) || (14 <= c && c <= 31) ||
				(35 <= c && c <= 91) || (93 <= c && c <= 126):
				// qtext
				localPartBytes = append(localPartBytes, c)
			default:
				return false
			}
		}
	} else {
		// Atom ("." Atom)*
		for len(in) > 0 {
			c := in[0]
			if c == '\\' {
				// RFC 3696 gives an example of an escape outside a quoted
				// string. Go takes it, and so does this body.
				in = in[1:]
				if len(in) == 0 {
					return false
				}
				localPartBytes = append(localPartBytes, in[0])
				in = in[1:]
				continue
			}
			if !atomChar(c) {
				break
			}
			localPartBytes = append(localPartBytes, c)
			in = in[1:]
		}
		if len(localPartBytes) == 0 {
			return false
		}
		// RFC 3696, section 3: a period starts no local part, it ends no
		// local part, and 2 periods never stand together.
		if localPartBytes[0] == '.' ||
			localPartBytes[len(localPartBytes)-1] == '.' ||
			bytes.Contains(localPartBytes, []byte{'.', '.'}) {
			return false
		}
	}
	if len(in) == 0 || in[0] != '@' {
		return false
	}
	in = in[1:]
	// A domain of the real world breaks the form of the RFC. Thus Go takes
	// each text after the at sign as the domain.
	if !domainNameValid(in, false) {
		return false
	}
	return !strings.ContainsRune(in, '@')
}

// atomChar reports whether the byte belongs to an atom of RFC 2822, section
// 3.2.4. The period joins 2 atoms, and this body takes it too.
func atomChar(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'z':
		return true
	case 'A' <= c && c <= 'Z':
		return true
	}
	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+',
		'-', '/', '=', '?', '^', '_', '`', '{',
		'|', '}', '~', '.':
		return true
	}
	return false
}
