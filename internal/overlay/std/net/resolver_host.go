//go:build dart

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
)

func (r *Resolver) goLookupHostOrder(ctx context.Context, name string, order hostLookupOrder, conf *dnsConfig) (addrs []string, err error) {
	if order == hostLookupFilesDNS || order == hostLookupFiles {
		// Use entries from /etc/hosts if they match.
		addrs, _ = lookupStaticHost(name)
		if len(addrs) > 0 {
			return
		}

		if order == hostLookupFiles {
			return nil, newDNSError(errNoSuchHost, name, "")
		}
	}
	ips, _, err := r.goLookupIPCNAMEOrder(ctx, "ip", name, order, conf)
	if err != nil {
		return
	}
	addrs = make([]string, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, ip.String())
	}
	return
}

func (r *Resolver) lookupHost(ctx context.Context, host string) ([]string, error) {
	order, conf := systemConf().hostLookupOrder(r, host)
	if order == hostLookupCgo {
		ips, err := systemLookupIP("ip", host)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(ips))
		for i, ip := range ips {
			out[i] = ip.String()
		}
		return out, nil
	}
	return r.goLookupHostOrder(ctx, host, order, conf)
}
