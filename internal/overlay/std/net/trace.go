//go:build dart

package net

import (
	"context"
	"internal/nettrace"
)

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
