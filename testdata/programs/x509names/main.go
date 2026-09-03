// Command x509names parses 2 certificates that hold a name constraints
// extension. It writes each of the 9 fields that the extension fills.
//
// The first certificate holds each of the 4 forms of a subtree: a DNS domain,
// an IP range, an electronic mail address, and a URI domain. The second holds
// a subtree of the form directoryName. No parse reads that form, thus the
// extension joins the list of the unhandled critical extensions.
package main

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
)

// namesDER holds a self signed CA certificate with an Ed25519 key. Its name
// constraints extension carries the critical mark.
const namesDER = "MIICFDCCAcagAwIBAgICEJIwBQYDK2VwMCIxIDAeBgNVBAMTF2dhcnQgbmFtZSBjb25zdHJh" +
	"aW50IENBMB4XDTIwMDEwMTAwMDAwMFoXDTQwMDEwMTAwMDAwMFowIjEgMB4GA1UEAxMXZ2Fy" +
	"dCBuYW1lIGNvbnN0cmFpbnQgQ0EwKjAFBgMrZXADIQB5tVYuj+ZU+UB4sRLoqYunkB+FOuaV" +
	"vtfg45ELrQSWZKOCAR4wggEaMA4GA1UdDwEB/wQEAwICBDAPBgNVHRMBAf8EBTADAQH/MB0G" +
	"A1UdDgQWBBRltgZz1u2IS/AcLCItgq2gdA8prDCB1wYDVR0eAQH/BIHMMIHJoIGBMA2CC2V4" +
	"YW1wbGUuY29tMA6CDC5leGFtcGxlLm9yZzAKhwgKAAAA/wAAADAihyAgAQ24AAAAAAAAAAAA" +
	"AAAA/////wAAAAAAAAAAAAAAADANgQtleGFtcGxlLmNvbTASgRB1c2VyQGV4YW1wbGUuY29t" +
	"MA2GC2V4YW1wbGUuY29toUMwEYIPYmFkLmV4YW1wbGUuY29tMAqHCMCoAAD//wAAMBKBEHNw" +
	"YW0uZXhhbXBsZS5jb20wDoYMLmJhZC5leGFtcGxlMAUGAytlcANBALhfbzysGGCVoSf2G29I" +
	"J+0NepVH/ZQgAm6R+MSbBBg77S+vQUtUURVqpja36oWijB0aKg6s8sFv9pTy+fXmlA4="

// subtreeDER holds a certificate whose name constraints extension names 1
// subtree of the form directoryName. No parse of Go reads that form.
const subtreeDER = "MIIBPjCB8aADAgECAgFjMAUGAytlcDAkMSIwIAYDVQQDExlnYXJ0IHVuaGFuZGxlZCBzdWJ0" +
	"cmVlIENBMB4XDTIwMDEwMTAwMDAwMFoXDTQwMDEwMTAwMDAwMFowJDEiMCAGA1UEAxMZZ2Fy" +
	"dCB1bmhhbmRsZWQgc3VidHJlZSBDQTAqMAUGAytlcAMhAGd+3Zx6b7ql1gnccAXOgBHVTgMU" +
	"rPicoXIFe8FA7eYqo0gwRjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQU+Is6q4d+Q5bR" +
	"53TpFfuPnc+UajAUBgNVHR4BAf8ECjAIoAYwBKQCMAAwBQYDK2VwA0EAHs34kaEaId27PqOj" +
	"g+P77u9LtN/vZv79judBFjEztk43n5nDaMq90+GsuHuL0Bsqg9nobLh3GYBgNosAvaYADg=="

// parse reads a certificate of its base64 text.
func parse(text string) *x509.Certificate {
	der, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		fmt.Println("base64:", err)
		return nil
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		fmt.Println("parse:", err)
		return nil
	}
	return c
}

// ranges writes each network of a list, because a print of a slice of
// pointers gives an address.
func ranges(name string, list []*net.IPNet) {
	fmt.Print(name, ":")
	for _, n := range list {
		fmt.Print(" ", n.String())
	}
	fmt.Println()
}

func main() {
	c := parse(namesDER)
	if c == nil {
		return
	}
	fmt.Println("critical:", c.PermittedDNSDomainsCritical)
	fmt.Println("permitted dns:", c.PermittedDNSDomains)
	fmt.Println("excluded dns:", c.ExcludedDNSDomains)
	ranges("permitted ip", c.PermittedIPRanges)
	ranges("excluded ip", c.ExcludedIPRanges)
	fmt.Println("permitted mail:", c.PermittedEmailAddresses)
	fmt.Println("excluded mail:", c.ExcludedEmailAddresses)
	fmt.Println("permitted uri:", c.PermittedURIDomains)
	fmt.Println("excluded uri:", c.ExcludedURIDomains)
	fmt.Println("unhandled:", len(c.UnhandledCriticalExtensions))

	s := parse(subtreeDER)
	if s == nil {
		return
	}
	fmt.Println("subtree critical:", s.PermittedDNSDomainsCritical)
	fmt.Println("subtree permitted dns:", len(s.PermittedDNSDomains))
	fmt.Println("subtree unhandled:", s.UnhandledCriticalExtensions)
}
