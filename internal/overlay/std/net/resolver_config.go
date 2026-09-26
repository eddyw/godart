//go:build dart

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"internal/bytealg"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// A resolverConfig represents a DNS stub resolver configuration.
type resolverConfig struct {
	initOnce sync.Once // guards init of resolverConfig

	// ch is used as a semaphore that only allows one lookup at a
	// time to recheck resolv.conf.
	ch          chan struct{} // guards lastChecked and modTime
	lastChecked time.Time     // last time resolv.conf was checked

	dnsConfig atomic.Pointer[dnsConfig] // parsed resolv.conf structure used in lookups
}

var resolvConf resolverConfig

func getSystemDNSConfig() *dnsConfig {
	return getSystemDNSConfigNamed("/etc/resolv.conf")
}

func getSystemDNSConfigNamed(path string) *dnsConfig {
	resolvConf.tryUpdate(path)
	return resolvConf.dnsConfig.Load()
}

// init initializes conf and is only called via conf.initOnce.
func (conf *resolverConfig) init() {
	// Set dnsConfig and lastChecked so we don't parse
	// resolv.conf twice the first time.
	conf.dnsConfig.Store(dnsReadConfig("/etc/resolv.conf"))
	conf.lastChecked = time.Now()

	// Prepare ch so that only one update of resolverConfig may
	// run at once.
	conf.ch = make(chan struct{}, 1)
}

// distantFuture is a sentinel time used for tests to signal that
// resolv.conf should not be rechecked.
var distantFuture = time.Date(3000, 1, 2, 3, 4, 5, 6, time.UTC)

// tryUpdate tries to update conf with the named resolv.conf file.
// The name variable only exists for testing. It is otherwise always
// "/etc/resolv.conf".
func (conf *resolverConfig) tryUpdate(name string) {
	conf.initOnce.Do(conf.init)

	dc := conf.dnsConfig.Load()

	// Currently we should never have a config that does not have any
	// available servers to query, since in such cases the servers field
	// is set to [defaultNS], see dnsReadConfig.
	// This assertion main purpose is for testing, such that we never set
	// the mocked dnsConfig in such way.
	if len(dc.servers) == 0 {
		panic("unreachable")
	}

	if dc.noReload {
		return
	}

	// Ensure only one update at a time checks resolv.conf.
	if !conf.tryAcquireSema() {
		return
	}
	defer conf.releaseSema()

	now := time.Now()

	// Only recheck the resolv.conf when:
	// - expired (last re-check was more that 5 seconds ago)
	// - the default nameservers are used (the last parse could not load
	//   resolv.conf, or it did not contain any nameservers)
	// - rechecks are not disabled (only possible in case of testing)
	//
	// Note: We only do one check at a time. Other concurrent requests might
	// still use the previous (outdated) version of the configuration file.
	expired := now.After(conf.lastChecked.Add(5 * time.Second))
	rechecksEnabled := conf.lastChecked != distantFuture // for testing purposes
	recheck := (expired || dc.isDefaultNS()) && rechecksEnabled
	if !recheck {
		return
	}
	conf.lastChecked = now

	switch runtime.GOOS {
	case "windows":
		// There's no file on disk, so don't bother checking
		// and failing.
		//
		// The Windows implementation of dnsReadConfig (called
		// below) ignores the name.
	default:
		var mtime time.Time
		if fi, err := os.Stat(name); err == nil {
			mtime = fi.ModTime()
		}
		if mtime.Equal(conf.dnsConfig.Load().mtime) {
			return
		}
	}

	dnsConf := dnsReadConfig(name)
	conf.dnsConfig.Store(dnsConf)
}

func (conf *resolverConfig) tryAcquireSema() bool {
	select {
	case conf.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (conf *resolverConfig) releaseSema() {
	<-conf.ch
}

// avoidDNS reports whether this is a hostname for which we should not
// use DNS. Currently this includes only .onion, per RFC 7686. See
// golang.org/issue/13705. Does not cover .local names (RFC 6762),
// see golang.org/issue/16739.
func avoidDNS(name string) bool {
	if name == "" {
		return true
	}
	name = strings.TrimSuffix(name, ".")
	return stringsHasSuffixFold(name, ".onion")
}

// nameList returns a list of names for sequential DNS queries.
func (conf *dnsConfig) nameList(name string) []string {
	// Check name length (see isDomainName).
	rooted := len(name) > 0 && name[len(name)-1] == '.'
	if len(name) > 254 || len(name) == 254 && !rooted {
		return nil
	}

	// If name is rooted (trailing dot), try only that name.
	if rooted {
		if avoidDNS(name) {
			return nil
		}
		return []string{name}
	}

	hasNdots := bytealg.CountString(name, '.') >= conf.ndots
	name += "."

	// Build list of search choices.
	names := make([]string, 0, 1+len(conf.search))
	// If name has enough dots, try unsuffixed first.
	if hasNdots && !avoidDNS(name) {
		names = append(names, name)
	}
	// Try suffixes that are not too long (see isDomainName).
	for _, suffix := range conf.search {
		fqdn := name + suffix
		if !avoidDNS(fqdn) && len(fqdn) <= 254 {
			names = append(names, fqdn)
		}
	}
	// Try unsuffixed, if not tried first above.
	if !hasNdots && !avoidDNS(name) {
		names = append(names, name)
	}
	return names
}

// hostLookupOrder specifies the order of LookupHost lookup strategies.
// It is basically a simplified representation of nsswitch.conf.
// "files" means /etc/hosts.
type hostLookupOrder int

const (
	// hostLookupCgo means defer to cgo.
	hostLookupCgo      hostLookupOrder = iota
	hostLookupFilesDNS                 // files first
	hostLookupDNSFiles                 // dns first
	hostLookupFiles                    // only files
	hostLookupDNS                      // only DNS
)

var lookupOrderName = map[hostLookupOrder]string{
	hostLookupCgo:      "cgo",
	hostLookupFilesDNS: "files,dns",
	hostLookupDNSFiles: "dns,files",
	hostLookupFiles:    "files",
	hostLookupDNS:      "dns",
}

func (o hostLookupOrder) String() string {
	if s, ok := lookupOrderName[o]; ok {
		return s
	}
	return "hostLookupOrder=" + strconv.Itoa(int(o)) + "??"
}
