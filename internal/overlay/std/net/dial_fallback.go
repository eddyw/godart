//go:build dart

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"time"
)

type sysDialer struct {
	Dialer
	network, address string
	testHookDialTCP  func(context.Context, string, *TCPAddr, *TCPAddr) (*TCPConn, error)
}

func (d *Dialer) dualStack() bool { return d.FallbackDelay >= 0 }

// partialDeadline returns the deadline to use for a single address,
// when multiple addresses are pending.
func partialDeadline(now, deadline time.Time, addrsRemaining int) (time.Time, error) {
	if deadline.IsZero() {
		return deadline, nil
	}
	timeRemaining := deadline.Sub(now)
	if timeRemaining <= 0 {
		return time.Time{}, errTimeout
	}
	// Tentatively allocate equal time to each remaining address.
	timeout := timeRemaining / time.Duration(addrsRemaining)
	// If the time per address is too short, steal from the end of the list.
	const saneMinimum = 2 * time.Second
	if timeout < saneMinimum {
		if timeRemaining < saneMinimum {
			timeout = timeRemaining
		} else {
			timeout = saneMinimum
		}
	}
	return now.Add(timeout), nil
}

func (d *Dialer) fallbackDelay() time.Duration {
	if d.FallbackDelay > 0 {
		return d.FallbackDelay
	} else {
		return 300 * time.Millisecond
	}
}

// dialParallel races two copies of dialSerial, giving the first a
// head start. It returns the first established connection and
// closes the others. Otherwise it returns an error from the first
// primary address.
func (sd *sysDialer) dialParallel(ctx context.Context, primaries, fallbacks addrList) (Conn, error) {
	if len(fallbacks) == 0 {
		return sd.dialSerial(ctx, primaries)
	}

	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		Conn
		error   error
		primary bool
		done    bool
	}
	results := make(chan dialResult) // unbuffered

	startRacer := func(ctx context.Context, primary bool) {
		ras := primaries
		if !primary {
			ras = fallbacks
		}
		c, err := sd.dialSerial(ctx, ras)
		select {
		case results <- dialResult{Conn: c, error: err, primary: primary, done: true}:
		case <-returned:
			if c != nil {
				c.Close()
			}
		}
	}

	var primary, fallback dialResult

	// Start the main racer.
	primaryCtx, primaryCancel := context.WithCancel(ctx)
	defer primaryCancel()
	go startRacer(primaryCtx, true)

	// Start the timer for the fallback racer.
	fallbackTimer := time.NewTimer(sd.fallbackDelay())
	defer fallbackTimer.Stop()

	for {
		select {
		case <-fallbackTimer.C:
			fallbackCtx, fallbackCancel := context.WithCancel(ctx)
			defer fallbackCancel()
			go startRacer(fallbackCtx, false)

		case res := <-results:
			if res.error == nil {
				return res.Conn, nil
			}
			if res.primary {
				primary = res
			} else {
				fallback = res
			}
			if primary.done && fallback.done {
				return nil, primary.error
			}
			if res.primary && fallbackTimer.Stop() {
				// If we were able to stop the timer, that means it
				// was running (hadn't yet started the fallback), but
				// we just got an error on the primary path, so start
				// the fallback immediately (in 0 nanoseconds).
				fallbackTimer.Reset(0)
			}
		}
	}
}

// dialSerial connects to a list of addresses in sequence, returning
// either the first successful connection, or the first error.
func (sd *sysDialer) dialSerial(ctx context.Context, ras addrList) (Conn, error) {
	var firstErr error // The error from the first address is most relevant.

	for i, ra := range ras {
		select {
		case <-ctx.Done():
			return nil, &OpError{Op: "dial", Net: sd.network, Source: sd.LocalAddr, Addr: ra, Err: mapErr(ctx.Err())}
		default:
		}

		dialCtx := ctx
		if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
			partialDeadline, err := partialDeadline(time.Now(), deadline, len(ras)-i)
			if err != nil {
				// Ran out of time.
				if firstErr == nil {
					firstErr = &OpError{Op: "dial", Net: sd.network, Source: sd.LocalAddr, Addr: ra, Err: err}
				}
				break
			}
			if partialDeadline.Before(deadline) {
				var cancel context.CancelFunc
				dialCtx, cancel = context.WithDeadline(ctx, partialDeadline)
				defer cancel()
			}
		}

		c, err := sd.dialSingle(dialCtx, ra)
		if err == nil {
			return c, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}

	if firstErr == nil {
		firstErr = &OpError{Op: "dial", Net: sd.network, Source: nil, Addr: nil, Err: errMissingAddress}
	}
	return nil, firstErr
}

var testHookDialTCP func(context.Context, string, *TCPAddr, *TCPAddr) (*TCPConn, error)

func (sd *sysDialer) dialSingle(ctx context.Context, ra Addr) (c Conn, err error) {
	address := ra.String()
	trace := traceOf(ctx)
	traceConnectStart(trace, sd.network, address)
	defer func() { traceConnectDone(trace, sd.network, address, err) }()
	local, _ := sd.LocalAddr.(*TCPAddr)
	c, err = sd.dialTCP(ctx, local, ra.(*TCPAddr))
	if err != nil {
		var source Addr
		if local != nil {
			source = local
		}
		return nil, &OpError{Op: "dial", Net: sd.network, Source: source, Addr: ra, Err: err}
	}
	return c, nil
}

func (sd *sysDialer) dialTCP(ctx context.Context, laddr, raddr *TCPAddr) (*TCPConn, error) {
	if h := sd.testHookDialTCP; h != nil {
		return h(ctx, sd.network, laddr, raddr)
	}
	if h := testHookDialTCP; h != nil {
		return h(ctx, sd.network, laddr, raddr)
	}
	return sd.doDialTCP(ctx, laddr, raddr)
}

func (sd *sysDialer) doDialTCP(ctx context.Context, laddr, raddr *TCPAddr) (*TCPConn, error) {
	dialer := sd.Dialer
	dialer.LocalAddr = nil
	if laddr != nil {
		dialer.LocalAddr = laddr
	}
	c, err := dialer.dialOneAddr(ctx, sd.network, raddr.String())
	if err != nil {
		return nil, err.(*OpError).Err
	}
	return c.(*TCPConn), nil
}
