//go:build dart

// This file holds the parse of the name constraints of a certificate and the
// check of a chain against them. The parse comes from the overlay, and the
// check comes from the file constraints.go of crypto/x509 of Go.
package x509

import (
	"bytes"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// This file reads the name constraints extension of a certificate. RFC 5280,
// 4.2.1.10.
//
// The work reads bytes alone, and it holds no crypto. Go reads the extension
// with golang.org/x/crypto/cryptobyte, and that package stays outside the
// surface. The parse below reads the same bytes with encoding/asn1, and it
// gives the same values and the same errors.
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

// The lines below come from the file constraints.go of crypto/x509 of Go.

// This file contains the data structures and functions necessary for
// efficiently checking X.509 name constraints. The method for constraint
// checking implemented in this file is based on a technique originally
// described by davidben@google.com.
//
// The basic concept is based on the fact that constraints describe possibly
// overlapping subtrees that we need to match against. If sorted in lexicographic
// order, and then pruned, removing any subtrees that overlap with preceding
// subtrees, a simple binary search can be used to find the nearest matching
// prefix. This reduces the complexity of name constraint checking from
// quadratic to log linear complexity.
//
// A close reading of RFC 5280 may suggest that constraints could also be
// implemented as a trie (or radix tree), which would present the possibility of
// doing construction and matching in linear time, but the memory cost of
// implementing them is actually quite high, and in the worst case (where each
// node has a high number of children) can be abused to require a program to use
// significant amounts of memory. The log linear approach taken here is
// extremely cheap in terms of memory because we directly alias the already
// parsed constraints, thus avoiding the need to do significant additional
// allocations.
//
// The basic data structure is nameConstraintsSet, which implements the sorting,
// pruning, and querying of the prefix sets.
//
// In order to check IP, DNS, URI, and email constraints, we need to use two
// different techniques, one for IP addresses, which is quite simple, and one
// for DNS names, which additionally compose the portions of URIs and emails we
// care about (technically we also need some special logic for email addresses
// as well for when constraints comprise of full email addresses) which is
// slightly more complex.
//
// IP addresses use two nameConstraintsSets, one for IPv4 addresses and one for
// IPv6 addresses, with no additional logic.
//
// DNS names require some extra logic in order to handle the distinctions
// between permitted and excluded subtrees, as well as for wildcards, and the
// semantics of leading period constraints (i.e. '.example.com'). This logic is
// implemented in the dnsConstraints type.
//
// Email addresses also require some additional logic, which does not make use
// of nameConstraintsSet, to handle constraints which define full email
// addresses (i.e. 'test@example.com'). For bare domain constraints, we use the
// dnsConstraints type described above, querying the domain portion of the email
// address. For full email addresses, we also hold a map of email addresses with
// the domain portion of the email lowercased, since it is case insensitive. When
// looking up an email address in the constraint set, we first check the full
// email address map, and if we don't find anything, we check the domain portion
// of the email address against the dnsConstraints.

type nameConstraintsSet[T *net.IPNet | string, V net.IP | string] struct {
	set []T
}

// sortAndPrune sorts the constraints using the provided comparison function, and then
// prunes any constraints that are subsets of preceding constraints using the
// provided subset function.
func (nc *nameConstraintsSet[T, V]) sortAndPrune(cmp func(T, T) int, subset func(T, T) bool) {
	if len(nc.set) < 2 {
		return
	}

	slices.SortFunc(nc.set, cmp)

	if len(nc.set) < 2 {
		return
	}
	writeIndex := 1
	for readIndex := 1; readIndex < len(nc.set); readIndex++ {
		if !subset(nc.set[writeIndex-1], nc.set[readIndex]) {
			nc.set[writeIndex] = nc.set[readIndex]
			writeIndex++
		}
	}
	nc.set = nc.set[:writeIndex]
}

// search does a binary search over the constraints set for the provided value
// s, using the provided comparison function cmp to find the lower bound, and
// the match function to determine if the found constraint is a prefix of s. If
// a matching constraint is found, it is returned along with true. If no
// matching constraint is found, the zero value of T and false are returned.
func (nc *nameConstraintsSet[T, V]) search(s V, cmp func(T, V) int, match func(T, V) bool) (lowerBound T, exactMatch bool) {
	if len(nc.set) == 0 {
		return lowerBound, false
	}
	// Look for the lower bound of s in the set.
	i, found := slices.BinarySearchFunc(nc.set, s, cmp)
	// If we found an exact match, return it
	if found {
		return nc.set[i], true
	}

	if i < 0 {
		return lowerBound, false
	}

	var constraint T
	if i == 0 {
		constraint = nc.set[0]
	} else {
		constraint = nc.set[i-1]
	}
	if match(constraint, s) {
		return constraint, true
	}
	return lowerBound, false
}

func ipNetworkSubset(a, b *net.IPNet) bool {
	if !a.Contains(b.IP) {
		return false
	}
	broadcast := make(net.IP, len(b.IP))
	for i := range b.IP {
		broadcast[i] = b.IP[i] | (^b.Mask[i])
	}
	return a.Contains(broadcast)
}

func ipNetworkCompare(a, b *net.IPNet) int {
	i := bytes.Compare(a.IP, b.IP)
	if i != 0 {
		return i
	}
	return bytes.Compare(a.Mask, b.Mask)
}

func ipBinarySearch(constraint *net.IPNet, target net.IP) int {
	return bytes.Compare(constraint.IP, target)
}

func ipMatch(constraint *net.IPNet, target net.IP) bool {
	return constraint.Contains(target)
}

type ipConstraints struct {
	// NOTE: we could store IP network prefixes as a pre-processed byte slice
	// (i.e. by masking the IP) and doing the byte prefix checking using faster
	// techniques, but this would require allocating new byte slices, which is
	// likely significantly more expensive than just operating on the
	// pre-allocated *net.IPNet and net.IP objects directly.

	ipv4 *nameConstraintsSet[*net.IPNet, net.IP]
	ipv6 *nameConstraintsSet[*net.IPNet, net.IP]
}

func newIPNetConstraints(l []*net.IPNet) interface {
	query(net.IP) (*net.IPNet, bool)
} {
	if len(l) == 0 {
		return nil
	}
	var ipv4, ipv6 []*net.IPNet
	for _, n := range l {
		// Subtrees may carry non-zero host bits. Sort and search need the masked
		// network address, so use a copy and leave the parsed constraint as encoded.
		if masked := n.IP.Mask(n.Mask); masked != nil && !masked.Equal(n.IP) {
			n = &net.IPNet{IP: masked, Mask: n.Mask}
		}
		if len(n.IP) == net.IPv4len {
			ipv4 = append(ipv4, n)
		} else {
			ipv6 = append(ipv6, n)
		}
	}
	var v4c, v6c *nameConstraintsSet[*net.IPNet, net.IP]
	if len(ipv4) > 0 {
		v4c = &nameConstraintsSet[*net.IPNet, net.IP]{
			set: ipv4,
		}
		v4c.sortAndPrune(ipNetworkCompare, ipNetworkSubset)
	}
	if len(ipv6) > 0 {
		v6c = &nameConstraintsSet[*net.IPNet, net.IP]{
			set: ipv6,
		}
		v6c.sortAndPrune(ipNetworkCompare, ipNetworkSubset)
	}
	return &ipConstraints{ipv4: v4c, ipv6: v6c}
}

func (ipc *ipConstraints) query(ip net.IP) (*net.IPNet, bool) {
	var c *nameConstraintsSet[*net.IPNet, net.IP]
	if len(ip) == net.IPv4len {
		c = ipc.ipv4
	} else {
		c = ipc.ipv6
	}
	if c == nil {
		return nil, false
	}
	return c.search(ip, ipBinarySearch, ipMatch)
}

// dnsHasSuffix case-insensitively checks if DNS name b is a label suffix of DNS
// name a, meaning that example.com is not considered a suffix of
// testexample.com, but is a suffix of test.example.com.
//
// dnsHasSuffix supports the URI "leading period" constraint semantics, which
// while not explicitly defined for dNSNames in RFC 5280, are widely supported
// (see errata 5997). In particular, a constraint of ".example.com" is
// considered to only match subdomains of example.com, but not example.com
// itself.
//
// a and b must both be non-empty strings representing (mostly) valid DNS names.
func dnsHasSuffix(a, b string) bool {
	lenA := len(a)
	lenB := len(b)
	if lenA > lenB {
		return false
	}
	i := lenA - 1
	offset := lenA - lenB
	for ; i >= 0; i-- {
		ar, br := a[i], b[i-(offset)]
		if ar == br {
			continue
		}
		if br < ar {
			ar, br = br, ar
		}
		if 'A' <= ar && ar <= 'Z' && br == ar+'a'-'A' {
			continue
		}
		return false
	}

	if a[0] != '.' && lenB > lenA && b[lenB-lenA-1] != '.' {
		return false
	}

	return true
}

// dnsCompareTable contains the ASCII alphabet mapped from a characters index in
// the table to its lowercased form.
var dnsCompareTable [256]byte

func init() {
	// NOTE: we don't actually need the
	// full alphabet, but calculating offsets would be more expensive than just
	// having redundant characters.
	for i := 0; i < 256; i++ {
		c := byte(i)
		if 'A' <= c && c <= 'Z' {
			// Lowercase uppercase characters A-Z.
			c += 'a' - 'A'
		}
		dnsCompareTable[i] = c
	}
	// Set the period character to 0 so that we get the right sorting behavior.
	//
	// In particular, we need the period character to sort before the only
	// other valid DNS name character which isn't a-z or 0-9, the hyphen,
	// otherwise a name with a dash would be incorrectly sorted into the middle
	// of another tree.
	//
	// For example, imagine a certificate with the constraints "a.com", "a.a.com", and
	// "a-a.com". These would sort as "a.com", "a-a.com", "a.a.com", which would break
	// the pruning step since we wouldn't see that "a.a.com" is a subset of "a.com".
	// Sorting the period before the hyphen ensures that "a.a.com" sorts before "a-a.com".
	dnsCompareTable['.'] = 0
}

// dnsCompare is a case-insensitive reversed implementation of strings.Compare
// that operates from the end to the start of the strings. This is more
// efficient that allocating reversed version of a and b and using
// strings.Compare directly (even though it is highly optimized).
//
// NOTE: this function treats the period character ('.') as sorting above every
// other character, which is necessary for us to properly sort names into their
// correct order. This is further discussed in the init function above.
func dnsCompare(a, b string) int {
	idxA := len(a) - 1
	idxB := len(b) - 1

	for idxA >= 0 && idxB >= 0 {
		byteA := dnsCompareTable[a[idxA]]
		byteB := dnsCompareTable[b[idxB]]
		if byteA == byteB {
			idxA--
			idxB--
			continue
		}
		ret := 1
		if byteA < byteB {
			ret = -1
		}
		return ret
	}

	ret := 0
	if idxA < idxB {
		ret = -1
	} else if idxB < idxA {
		ret = 1
	}
	return ret
}

type dnsConstraints struct {
	// all lets us short circuit the query logic if we see a zero length
	// constraint which permits or excludes everything.
	all bool

	// permitted indicates if these constraints are for permitted or excluded
	// names.
	permitted bool

	constraints *nameConstraintsSet[string, string]

	// parentConstraints contains a subset of constraints which are used for
	// wildcard SAN queries, which are constructed by removing the first label
	// from the constraints in constraints. parentConstraints is only populated
	// if permitted is false.
	parentConstraints map[string]string
}

func newDNSConstraints(l []string, permitted bool) interface{ query(string) (string, bool) } {
	if len(l) == 0 {
		return nil
	}
	for _, n := range l {
		if len(n) == 0 {
			return &dnsConstraints{all: true}
		}
	}
	constraints := slices.Clone(l)

	nc := &dnsConstraints{
		constraints: &nameConstraintsSet[string, string]{
			set: constraints,
		},
		permitted: permitted,
	}

	nc.constraints.sortAndPrune(dnsCompare, dnsHasSuffix)

	if !permitted {
		parentConstraints := map[string]string{}
		for _, name := range nc.constraints.set {
			name = strings.ToLower(name)
			trimmedName := trimFirstLabel(name)
			if trimmedName == "" {
				continue
			}
			parentConstraints[trimmedName] = name
		}
		if len(parentConstraints) > 0 {
			nc.parentConstraints = parentConstraints
		}
	}

	return nc
}

func (dnc *dnsConstraints) query(s string) (string, bool) {
	if dnc.all {
		return "", true
	}

	constraint, match := dnc.constraints.search(s, dnsCompare, dnsHasSuffix)
	if match {
		return constraint, true
	}

	if !dnc.permitted && len(s) > 0 && s[0] == '*' {
		s = strings.ToLower(s)
		trimmed := trimFirstLabel(s)
		if constraint, found := dnc.parentConstraints[trimmed]; found {
			return constraint, true
		}
	}
	return "", false
}

type emailConstraints struct {
	dnsConstraints interface{ query(string) (string, bool) }

	// fullEmails is map of rfc2821Mailboxs that are fully specified in the
	// constraints, which we need to check for separately since they don't
	// follow the same matching rules as the domain-based constraints. The
	// domain portion of the rfc2821Mailbox has been lowercased, since the
	// domain portion is case insensitive. When checking the map for an email,
	// the domain portion of the query should also be lowercased.
	fullEmails map[rfc2821Mailbox]struct{}
}

func newEmailConstraints(l []string, permitted bool) interface {
	query(rfc2821Mailbox) (string, bool)
} {
	if len(l) == 0 {
		return nil
	}
	exactMap := map[rfc2821Mailbox]struct{}{}
	var domains []string
	for _, c := range l {
		if !strings.ContainsRune(c, '@') {
			domains = append(domains, c)
			continue
		}
		parsed, ok := parseRFC2821Mailbox(c)
		if !ok {
			// We've already parsed these addresses in parseCertificate, and
			// treat failures as a hard failure for parsing. The only way we can
			// get a parse failure here is if the caller has mutated the
			// certificate since parsing.
			continue
		}
		parsed.domain = strings.ToLower(parsed.domain)
		exactMap[parsed] = struct{}{}
	}
	ec := &emailConstraints{
		fullEmails: exactMap,
	}
	if len(domains) > 0 {
		ec.dnsConstraints = newDNSConstraints(domains, permitted)
	}
	return ec
}

func (ec *emailConstraints) query(s rfc2821Mailbox) (string, bool) {
	if len(ec.fullEmails) > 0 {
		if _, ok := ec.fullEmails[s]; ok {
			return fmt.Sprintf("%s@%s", s.local, s.domain), true
		}
	}
	if ec.dnsConstraints == nil {
		return "", false
	}
	constraint, found := ec.dnsConstraints.query(s.domain)
	return constraint, found
}

type constraints[T any, V any] struct {
	constraintType string
	permitted      interface{ query(V) (T, bool) }
	excluded       interface{ query(V) (T, bool) }
}

func checkConstraints[T string | *net.IPNet, V any, P string | net.IP | parsedURI | rfc2821Mailbox](c constraints[T, V], s V, p P) error {
	if c.permitted != nil {
		if _, found := c.permitted.query(s); !found {
			return fmt.Errorf("%s %q is not permitted by any constraint", c.constraintType, p)
		}
	}
	if c.excluded != nil {
		if constraint, found := c.excluded.query(s); found {
			return fmt.Errorf("%s %q is excluded by constraint %q", c.constraintType, p, constraint)
		}
	}
	return nil
}

type chainConstraints struct {
	ip    constraints[*net.IPNet, net.IP]
	dns   constraints[string, string]
	uri   constraints[string, string]
	email constraints[string, rfc2821Mailbox]

	index int
	next  *chainConstraints
}

func (cc *chainConstraints) check(dns []string, uris []parsedURI, emails []rfc2821Mailbox, ips []net.IP) error {
	for _, ip := range ips {
		if err := checkConstraints(cc.ip, ip, ip); err != nil {
			return err
		}
	}
	for _, d := range dns {
		if !domainNameValid(d, false) {
			return fmt.Errorf("x509: cannot parse dnsName %q", d)
		}
		if err := checkConstraints(cc.dns, d, d); err != nil {
			return err
		}
	}
	for _, u := range uris {
		if !domainNameValid(u.domain, false) {
			return fmt.Errorf("x509: internal error: URI SAN %q failed to parse", u)
		}
		if err := checkConstraints(cc.uri, u.domain, u); err != nil {
			return err
		}
	}
	for _, e := range emails {
		if !domainNameValid(e.domain, false) {
			return fmt.Errorf("x509: cannot parse rfc822Name %q", e)
		}
		if err := checkConstraints(cc.email, e, e); err != nil {
			return err
		}
	}
	return nil
}

func checkChainConstraints(chain []*Certificate) error {
	var currentConstraints *chainConstraints
	var last *chainConstraints
	for i, c := range chain {
		if !c.hasNameConstraints() {
			continue
		}
		cc := &chainConstraints{
			ip:    constraints[*net.IPNet, net.IP]{"IP address", newIPNetConstraints(c.PermittedIPRanges), newIPNetConstraints(c.ExcludedIPRanges)},
			dns:   constraints[string, string]{"DNS name", newDNSConstraints(c.PermittedDNSDomains, true), newDNSConstraints(c.ExcludedDNSDomains, false)},
			uri:   constraints[string, string]{"URI", newDNSConstraints(c.PermittedURIDomains, true), newDNSConstraints(c.ExcludedURIDomains, false)},
			email: constraints[string, rfc2821Mailbox]{"email address", newEmailConstraints(c.PermittedEmailAddresses, true), newEmailConstraints(c.ExcludedEmailAddresses, false)},
			index: i,
		}
		if currentConstraints == nil {
			currentConstraints = cc
			last = cc
		} else if last != nil {
			last.next = cc
			last = cc
		}
	}
	if currentConstraints == nil {
		return nil
	}

	for i, c := range chain {
		if !c.hasSANExtension() {
			continue
		}
		if i >= currentConstraints.index {
			for currentConstraints.index <= i {
				if currentConstraints.next == nil {
					return nil
				}
				currentConstraints = currentConstraints.next
			}
		}

		uris, err := parseURIs(c.URIs)
		if err != nil {
			return err
		}
		emails, err := parseMailboxes(c.EmailAddresses)
		if err != nil {
			return err
		}

		for n := currentConstraints; n != nil; n = n.next {
			if err := n.check(c.DNSNames, uris, emails, c.IPAddresses); err != nil {
				return err
			}
		}
	}

	return nil
}

type parsedURI struct {
	uri    *url.URL
	domain string
}

func (u parsedURI) String() string {
	return u.uri.String()
}

func parseURIs(uris []*url.URL) ([]parsedURI, error) {
	parsed := make([]parsedURI, 0, len(uris))
	for _, uri := range uris {
		host := strings.ToLower(uri.Host)
		if len(host) == 0 {
			return nil, fmt.Errorf("URI with empty host (%q) cannot be matched against constraints", uri.String())
		}
		if strings.Contains(host, ":") && !strings.HasSuffix(host, "]") {
			var err error
			host, _, err = net.SplitHostPort(uri.Host)
			if err != nil {
				return nil, fmt.Errorf("cannot parse URI host %q: %v", uri.Host, err)
			}
		}

		// netip.ParseAddr will reject the URI IPv6 literal form "[...]", so we
		// check if _either_ the string parses as an IP, or if it is enclosed in
		// square brackets.
		if _, err := netip.ParseAddr(host); err == nil || (strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]")) {
			return nil, fmt.Errorf("URI with IP (%q) cannot be matched against constraints", uri.String())
		}

		parsed = append(parsed, parsedURI{uri, host})
	}
	return parsed, nil
}

func parseMailboxes(emails []string) ([]rfc2821Mailbox, error) {
	parsed := make([]rfc2821Mailbox, 0, len(emails))
	for _, email := range emails {
		mailbox, ok := parseRFC2821Mailbox(email)
		if !ok {
			return nil, fmt.Errorf("cannot parse rfc822Name %q", email)
		}
		mailbox.domain = strings.ToLower(mailbox.domain)
		parsed = append(parsed, mailbox)
	}
	return parsed, nil
}

func trimFirstLabel(dnsName string) string {
	firstDotInd := strings.IndexByte(dnsName, '.')
	if firstDotInd < 0 {
		// Constraint is a single label, we cannot trim it.
		return ""
	}
	return dnsName[firstDotInd:]
}
