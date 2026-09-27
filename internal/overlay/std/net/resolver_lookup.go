//go:build dart

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"golang.org/x/net/dns/dnsmessage"
	"net/netip"
	"strings"
	"sync"
)

// dnsWaitGroup can be used by tests to wait for all DNS goroutines to
// complete. This avoids races on the test hooks.
var dnsWaitGroup sync.WaitGroup

// ipVersion returns the provided network's IP version: '4', '6' or 0
// if network does not end in a '4' or '6' byte.
func ipVersion(network string) byte {
	if network == "" {
		return 0
	}
	n := network[len(network)-1]
	if n != '4' && n != '6' {
		n = 0
	}
	return n
}

func (r *Resolver) strictErrors() bool { return r != nil && r.StrictErrors }

// LookupCNAME returns the canonical name for the given host.
// Callers that do not care about the canonical name can call
// [LookupHost] or [LookupIP] directly; both take care of resolving
// the canonical name as part of the lookup.
//
// A canonical name is the final name after following zero
// or more CNAME records.
// LookupCNAME does not return an error if host does not
// contain DNS "CNAME" records, as long as host resolves to
// address records.
//
// The returned canonical name is validated to be a properly
// formatted presentation-format domain name.
func (r *Resolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	order, conf := systemConf().hostLookupOrder(r, host)
	cname, err := r.goLookupCNAME(ctx, host, order, conf)
	if err != nil {
		return "", err
	}
	if !isDomainName(cname) {
		return "", &DNSError{Err: errMalformedDNSRecordsDetail, Name: host}
	}
	return cname, nil
}

// LookupSRV tries to resolve an [SRV] query of the given service,
// protocol, and domain name. The proto is "tcp" or "udp".
// The returned records are sorted by priority and randomized
// by weight within a priority.
//
// LookupSRV constructs the DNS name to look up following RFC 2782.
// That is, it looks up _service._proto.name. To accommodate services
// publishing SRV records under non-standard names, if both service
// and proto are empty strings, LookupSRV looks up name directly.
//
// The returned cname is the owner name from the first SRV answer
// record, which is typically the constructed DNS name
// (_service._proto.name) but may differ if CNAME records redirect
// the query to another name.
//
// The returned service names are validated to be properly
// formatted presentation-format domain names. If the response contains
// invalid names, those records are filtered out and an error
// will be returned alongside the remaining results, if any.
func (r *Resolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*SRV, error) {
	cname, addrs, err := r.lookupSRV(ctx, service, proto, name)
	if err != nil {
		return "", nil, err
	}
	if cname != "" && !isDomainName(cname) {
		return "", nil, &DNSError{Err: "SRV header name is invalid", Name: name}
	}
	filteredAddrs := make([]*SRV, 0, len(addrs))
	for _, addr := range addrs {
		if addr == nil {
			continue
		}
		if !isDomainName(addr.Target) {
			continue
		}
		filteredAddrs = append(filteredAddrs, addr)
	}
	if len(addrs) != len(filteredAddrs) {
		return cname, filteredAddrs, &DNSError{Err: errMalformedDNSRecordsDetail, Name: name}
	}
	return cname, filteredAddrs, nil
}

// LookupMX returns the DNS MX records for the given domain name sorted by preference.
//
// The returned mail server names are validated to be properly
// formatted presentation-format domain names, or numeric IP addresses.
// If the response contains invalid names, those records are filtered out
// and an error will be returned alongside the remaining results, if any.
func (r *Resolver) LookupMX(ctx context.Context, name string) ([]*MX, error) {
	records, err := r.lookupMX(ctx, name)
	if err != nil {
		return nil, err
	}
	filteredMX := make([]*MX, 0, len(records))
	for _, mx := range records {
		if mx == nil {
			continue
		}
		if !isDomainName(mx.Host) {
			// Check for IP address. In practice we observe
			// these with a trailing dot, so strip that.
			ip, err := netip.ParseAddr(strings.TrimSuffix(mx.Host, "."))
			if err != nil || ip.Zone() != "" {
				continue
			}
		}
		filteredMX = append(filteredMX, mx)
	}
	if len(records) != len(filteredMX) {
		return filteredMX, &DNSError{Err: errMalformedDNSRecordsDetail, Name: name}
	}
	return filteredMX, nil
}

// LookupNS returns the DNS NS records for the given domain name.
//
// The returned name server names are validated to be properly
// formatted presentation-format domain names. If the response contains
// invalid names, those records are filtered out and an error
// will be returned alongside the remaining results, if any.
func (r *Resolver) LookupNS(ctx context.Context, name string) ([]*NS, error) {
	records, err := r.lookupNS(ctx, name)
	if err != nil {
		return nil, err
	}
	filteredNS := make([]*NS, 0, len(records))
	for _, ns := range records {
		if ns == nil {
			continue
		}
		if !isDomainName(ns.Host) {
			continue
		}
		filteredNS = append(filteredNS, ns)
	}
	if len(records) != len(filteredNS) {
		return filteredNS, &DNSError{Err: errMalformedDNSRecordsDetail, Name: name}
	}
	return filteredNS, nil
}

// LookupTXT returns the DNS TXT records for the given domain name.
//
// If a DNS TXT record holds multiple strings, they are concatenated as a
// single string.
func (r *Resolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.lookupTXT(ctx, name)
}

// LookupAddr performs a reverse lookup for the given address, returning a list
// of names mapping to that address.
//
// The returned names are validated to be properly formatted presentation-format
// domain names. If the response contains invalid names, those records are filtered
// out and an error will be returned alongside the remaining results, if any.
func (r *Resolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	names, err := r.lookupAddr(ctx, addr)
	if err != nil {
		return nil, err
	}
	filteredNames := make([]string, 0, len(names))
	for _, name := range names {
		if isDomainName(name) {
			filteredNames = append(filteredNames, name)
		}
	}
	if len(names) != len(filteredNames) {
		return filteredNames, &DNSError{Err: errMalformedDNSRecordsDetail, Name: addr}
	}
	return filteredNames, nil
}

// errMalformedDNSRecordsDetail is the DNSError detail which is returned when a Resolver.Lookup...
// method receives DNS records which contain invalid DNS names. This may be returned alongside
// results which have had the malformed records filtered out.
var errMalformedDNSRecordsDetail = "DNS response contained records which contain invalid names"

// dial makes a new connection to the provided server (which must be
// an IP address) with the provided network type, using either r.Dial
// (if both r and r.Dial are non-nil) or else Dialer.DialContext.
func (r *Resolver) dial(ctx context.Context, network, server string) (Conn, error) {
	// Calling Dial here is scary -- we have to be sure not to
	// dial a name that will require a DNS lookup, or Dial will
	// call back here to translate it. The DNS config parser has
	// already checked that all the cfg.servers are IP
	// addresses, which Dial will use without a DNS lookup.
	var c Conn
	var err error
	if r != nil && r.Dial != nil {
		c, err = r.Dial(ctx, network, server)
	} else {
		var d Dialer
		c, err = d.DialContext(ctx, network, server)
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return c, nil
}

// goLookupSRV returns the SRV records for a target name, built either
// from its component service ("sip"), protocol ("tcp"), and name
// ("example.com."), or from name directly (if service and proto are
// both empty).
//
// In either case, the returned target name ("_sip._tcp.example.com.")
// is also returned on success.
//
// The records are sorted by weight.
func (r *Resolver) goLookupSRV(ctx context.Context, service, proto, name string) (target string, srvs []*SRV, err error) {
	if service == "" && proto == "" {
		target = name
	} else {
		target = "_" + service + "._" + proto + "." + name
	}
	p, server, err := r.lookup(ctx, target, dnsmessage.TypeSRV, nil)
	if err != nil {
		return "", nil, err
	}
	var cname dnsmessage.Name
	for {
		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			return "", nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		if h.Type != dnsmessage.TypeSRV {
			if err := p.SkipAnswer(); err != nil {
				return "", nil, &DNSError{
					Err:    "cannot unmarshal DNS message",
					Name:   name,
					Server: server,
				}
			}
			continue
		}
		if cname.Length == 0 && h.Name.Length != 0 {
			cname = h.Name
		}
		srv, err := p.SRVResource()
		if err != nil {
			return "", nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		srvs = append(srvs, &SRV{Target: srv.Target.String(), Port: srv.Port, Priority: srv.Priority, Weight: srv.Weight})
	}
	byPriorityWeight(srvs).sort()
	return cname.String(), srvs, nil
}

// goLookupMX returns the MX records for name.
func (r *Resolver) goLookupMX(ctx context.Context, name string) ([]*MX, error) {
	p, server, err := r.lookup(ctx, name, dnsmessage.TypeMX, nil)
	if err != nil {
		return nil, err
	}
	var mxs []*MX
	for {
		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			return nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		if h.Type != dnsmessage.TypeMX {
			if err := p.SkipAnswer(); err != nil {
				return nil, &DNSError{
					Err:    "cannot unmarshal DNS message",
					Name:   name,
					Server: server,
				}
			}
			continue
		}
		mx, err := p.MXResource()
		if err != nil {
			return nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		mxs = append(mxs, &MX{Host: mx.MX.String(), Pref: mx.Pref})

	}
	byPref(mxs).sort()
	return mxs, nil
}

// goLookupNS returns the NS records for name.
func (r *Resolver) goLookupNS(ctx context.Context, name string) ([]*NS, error) {
	p, server, err := r.lookup(ctx, name, dnsmessage.TypeNS, nil)
	if err != nil {
		return nil, err
	}
	var nss []*NS
	for {
		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			return nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		if h.Type != dnsmessage.TypeNS {
			if err := p.SkipAnswer(); err != nil {
				return nil, &DNSError{
					Err:    "cannot unmarshal DNS message",
					Name:   name,
					Server: server,
				}
			}
			continue
		}
		ns, err := p.NSResource()
		if err != nil {
			return nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		nss = append(nss, &NS{Host: ns.NS.String()})
	}
	return nss, nil
}

// goLookupTXT returns the TXT records from name.
func (r *Resolver) goLookupTXT(ctx context.Context, name string) ([]string, error) {
	p, server, err := r.lookup(ctx, name, dnsmessage.TypeTXT, nil)
	if err != nil {
		return nil, err
	}
	var txts []string
	for {
		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			return nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		if h.Type != dnsmessage.TypeTXT {
			if err := p.SkipAnswer(); err != nil {
				return nil, &DNSError{
					Err:    "cannot unmarshal DNS message",
					Name:   name,
					Server: server,
				}
			}
			continue
		}
		txt, err := p.TXTResource()
		if err != nil {
			return nil, &DNSError{
				Err:    "cannot unmarshal DNS message",
				Name:   name,
				Server: server,
			}
		}
		// Multiple strings in one TXT record need to be
		// concatenated without separator to be consistent
		// with previous Go resolver.
		n := 0
		for _, s := range txt.TXT {
			n += len(s)
		}
		txtJoin := make([]byte, 0, n)
		for _, s := range txt.TXT {
			txtJoin = append(txtJoin, s...)
		}
		if len(txts) == 0 {
			txts = make([]string, 0, 1)
		}
		txts = append(txts, string(txtJoin))
	}
	return txts, nil
}
