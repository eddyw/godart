//go:build dart

package net

import (
	"context"
	"internal/nettrace"
)

// This file carries the hooks of internal/nettrace, exactly as the real net
// package does. The package net/http/httptrace puts a Trace in the context,
// and the dial and the lookup below report each event to it.
//
// A dial of a host name reports no event of the lookup, because Socket.connect
// of dart:io reads the name itself. A dial that reads a Resolver of Go first
// reports the lookup, because that code runs here.

// traceOf gives the trace that a context carries, or nil.
func traceOf(ctx context.Context) *nettrace.Trace {
	if ctx == nil {
		return nil
	}
	t, _ := ctx.Value(nettrace.TraceKey{}).(*nettrace.Trace)
	return t
}

// traceConnectStart reports the start of a dial.
func traceConnectStart(t *nettrace.Trace, network, address string) {
	if t != nil && t.ConnectStart != nil {
		t.ConnectStart(network, address)
	}
}

// traceConnectDone reports the end of a dial with its error.
func traceConnectDone(t *nettrace.Trace, network, address string, err error) {
	if t != nil && t.ConnectDone != nil {
		t.ConnectDone(network, address, err)
	}
}

// traceDNSStart reports the start of a lookup of a host name.
func traceDNSStart(t *nettrace.Trace, host string) {
	if t != nil && t.DNSStart != nil {
		t.DNSStart(host)
	}
}

// traceDNSDone reports the end of a lookup with its addresses and its error.
//
// The hook takes a list of any, because internal/nettrace stands below net and
// it names no type of this package.
func traceDNSDone(t *nettrace.Trace, ips []IP, err error) {
	if t == nil || t.DNSDone == nil {
		return
	}
	out := make([]any, 0, len(ips))
	for _, ip := range ips {
		out = append(out, IPAddr{IP: ip})
	}
	t.DNSDone(out, false, err)
}
