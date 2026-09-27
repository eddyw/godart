//go:build dart

package net

import (
	"context"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"syscall/dart"
)

// A Resolver resolves host names and service names.
type Resolver struct {
	PreferGo     bool
	StrictErrors bool
	// Dial opens the connection to 1 server. A nil field takes Dial of this
	// package. A test names its own server through this field.
	Dial func(ctx context.Context, network, address string) (Conn, error)
}

// DefaultResolver is the resolver that each package level Lookup function
// reads.
var DefaultResolver = &Resolver{}

// LookupHost gives each address of a host, as text.
func (r *Resolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if ip := ParseIP(host); ip != nil {
		return []string{host}, nil
	}
	addrs, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.IP.String())
	}
	return out, nil
}

// LookupIPAddr gives each address of a host.
func (r *Resolver) LookupIPAddr(ctx context.Context, host string) ([]IPAddr, error) {
	ips, err := r.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	out := make([]IPAddr, 0, len(ips))
	for _, ip := range ips {
		out = append(out, IPAddr{IP: ip})
	}
	return out, nil
}

// LookupIP gives each address of a host. The network takes ip, ip4, or ip6.
func (r *Resolver) LookupIP(ctx context.Context, network, host string) ([]IP, error) {
	switch network {
	case "ip", "ip4", "ip6":
	default:
		return nil, UnknownNetworkError(network)
	}
	// A numeric address needs no lookup, thus it reports no event. Go holds
	// the same order.
	if ip := ParseIP(host); ip != nil {
		if keepIP(network, ip) {
			return []IP{ip}, nil
		}
		return nil, &DNSError{Err: errNoSuchHost.Error(), Name: host, IsNotFound: true}
	}
	trace := traceOf(ctx)
	traceDNSStart(trace, host)
	order, conf := systemConf().hostLookupOrder(r, host)
	out, err := r.lookupIP(ctx, network, host, order, conf)
	traceDNSDone(trace, out, err)
	return out, err
}

// lookupAddrs reads each address of a host name through the resolver of the
// system, and it gives the addresses in the order that the system holds.
//
// family holds 4 for the network ip4, 6 for the network ip6, and 0 for each
// family. The first line of the result names the cause of a failure, and each
// line after it holds 1 address. An empty first line means a success.
//
//gart:dart package:goruntime/goruntime.dart $lookupAddrs blocking
func lookupAddrs(host string, family int) string

func (r *Resolver) preferGo() bool { return r != nil && r.PreferGo }

// familyOf gives the number of the family of a network of the lookup.
func familyOf(network string) int {
	switch network {
	case "ip4":
		return 4
	case "ip6":
		return 6
	}
	return 0
}

// systemLookupIP reads each address of a host through the resolver of the
// platform.
func systemLookupIP(network, host string) ([]IP, error) {
	lines := strings.Split(lookupAddrs(host, familyOf(network)), "\n")
	if cause := lines[0]; cause != "" {
		return nil, &DNSError{
			Err:         cause,
			Name:        host,
			IsNotFound:  cause == errNoSuchHost.Error(),
			IsTemporary: cause == errServerMisbehaving.Error(),
		}
	}
	var out []IP
	for _, text := range lines[1:] {
		if text == "" {
			continue
		}
		if ip := ParseIP(text); ip != nil && keepIP(network, ip) {
			out = append(out, ip)
		}
	}
	if len(out) == 0 {
		return nil, &DNSError{Err: errNoSuchHost.Error(), Name: host, IsNotFound: true}
	}
	return out, nil
}

func (r *Resolver) lookupIP(ctx context.Context, network, host string, order hostLookupOrder, conf *dnsConfig) ([]IP, error) {
	if order == hostLookupCgo {
		return systemLookupIP(network, host)
	}
	addresses, _, err := r.goLookupIPCNAMEOrder(ctx, network, host, order, conf)
	if err != nil {
		return nil, err
	}
	out := make([]IP, 0, len(addresses))
	for _, address := range addresses {
		if keepIP(network, address.IP) {
			out = append(out, address.IP)
		}
	}
	return out, nil
}

// keepIP reports whether the address fits the network.
func keepIP(network string, ip IP) bool {
	switch network {
	case "ip4":
		return ip.To4() != nil
	case "ip6":
		return ip.To4() == nil && len(ip.To16()) == 16
	}
	return true
}

// LookupNetIP gives each address of a host, in the form of net/netip.
func (r *Resolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	ips, err := r.LookupIP(ctx, network, host)
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		if a, ok := netip.AddrFromSlice(ip); ok {
			out = append(out, a.Unmap())
		}
	}
	return out, nil
}

// LookupPort gives the port of a service. A service that holds a number gives
// that number.
func (r *Resolver) LookupPort(ctx context.Context, network, service string) (int, error) {
	port, needsLookup := parsePort(service)
	if needsLookup {
		switch network {
		case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "ip":
		case "":
			// An empty network is a wildcard, and Go 1.0 held the same rule.
			network = "ip"
		default:
			return 0, &AddrError{Err: "unknown network", Addr: network}
		}
		var ok bool
		port, ok = servicePort(network, service)
		if !ok {
			if systemConf().mustUseGoResolver(r) {
				switch network {
				case "tcp4", "tcp6", "udp4", "udp6":
					network = network[:3]
				}
			}
			name := network + "/" + service
			if !systemConf().mustUseGoResolver(r) && strings.IndexByte(service, 0) >= 0 {
				return 0, &DNSError{Err: syscall.EINVAL.Error(), Name: name}
			}
			return 0, &DNSError{Err: errUnknownPort.Error(), Name: name, IsNotFound: true}
		}
	}
	if 0 > port || port > 65535 {
		return 0, &AddrError{Err: "invalid port", Addr: service}
	}
	return port, nil
}

// servicePort reads the table of /etc/services. A network of ip reads each
// protocol.
func servicePort(network, service string) (int, bool) {
	readServices()
	name := strings.ToLower(service)
	switch network {
	case "tcp", "tcp4", "tcp6":
		port, ok := servicePorts["tcp/"+name]
		return port, ok
	case "udp", "udp4", "udp6":
		port, ok := servicePorts["udp/"+name]
		return port, ok
	}
	for _, proto := range []string{"tcp", "udp"} {
		if port, ok := servicePorts[proto+"/"+name]; ok {
			return port, true
		}
	}
	return 0, false
}

// parsePort reads a service as a decimal number. The body follows the source
// of Go, because a resolver of the system gives a valid port for a number
// above 65536, thus the parse reads the whole text before it rejects one.
func parsePort(service string) (port int, needsLookup bool) {
	if service == "" {
		// An empty service means the port 0, and Go 1.0 held the same rule.
		return 0, false
	}
	const (
		max    = uint32(1<<32 - 1)
		cutoff = uint32(1 << 30)
	)
	neg := false
	if service[0] == '+' {
		service = service[1:]
	} else if service[0] == '-' {
		neg = true
		service = service[1:]
	}
	var n uint32
	for _, d := range service {
		if '0' <= d && d <= '9' {
			d -= '0'
		} else {
			return 0, true
		}
		if n >= cutoff {
			n = max
			break
		}
		n *= 10
		nn := n + uint32(d)
		if nn < n || nn > max {
			n = max
			break
		}
		n = nn
	}
	if !neg && n >= cutoff {
		port = int(cutoff - 1)
	} else if neg && n > cutoff {
		port = int(cutoff)
	} else {
		port = int(n)
	}
	if neg {
		port = -port
	}
	return port, false
}

var (
	servicesOnce sync.Once
	servicePorts map[string]int
)

// readServices reads /etc/services 1 time for each process. Go reads the same
// file on each platform of Unix.
func readServices() {
	servicesOnce.Do(func() {
		// Go provides these ports even when /etc/services is absent.
		servicePorts = map[string]int{
			"udp/domain":      53,
			"tcp/ftp":         21,
			"tcp/ftps":        990,
			"tcp/gopher":      70,
			"tcp/http":        80,
			"tcp/https":       443,
			"tcp/imap2":       143,
			"tcp/imap3":       220,
			"tcp/imaps":       993,
			"tcp/pop3":        110,
			"tcp/pop3s":       995,
			"tcp/smtp":        25,
			"tcp/submissions": 465,
			"tcp/ssh":         22,
			"tcp/telnet":      23,
		}
		text, cause := dart.ReadFile("/etc/services")
		if cause != "" {
			return
		}
		for _, line := range strings.Split(string(text), "\n") {
			if i := strings.IndexByte(line, '#'); i >= 0 {
				line = line[:i]
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			portText, proto, found := strings.Cut(fields[1], "/")
			if !found {
				continue
			}
			port, err := strconv.Atoi(portText)
			if err != nil {
				continue
			}
			for _, name := range append([]string{fields[0]}, fields[2:]...) {
				servicePorts[proto+"/"+strings.ToLower(name)] = port
			}
		}
	})
}

// The package level functions each read DefaultResolver.

// LookupHost gives each address of a host, as text.
func LookupHost(host string) ([]string, error) {
	return DefaultResolver.LookupHost(context.Background(), host)
}

// LookupIP gives each address of a host.
func LookupIP(host string) ([]IP, error) {
	return DefaultResolver.LookupIP(context.Background(), "ip", host)
}

// LookupCNAME gives the canonical name of a host.
func LookupCNAME(host string) (string, error) {
	return DefaultResolver.LookupCNAME(context.Background(), host)
}

// LookupMX gives each mail exchange record of a name.
func LookupMX(name string) ([]*MX, error) {
	return DefaultResolver.LookupMX(context.Background(), name)
}

// LookupNS gives each name server record of a name.
func LookupNS(name string) ([]*NS, error) {
	return DefaultResolver.LookupNS(context.Background(), name)
}

// LookupTXT gives the text of each TXT record of a name.
func LookupTXT(name string) ([]string, error) {
	return DefaultResolver.LookupTXT(context.Background(), name)
}

// LookupSRV gives the canonical name and each service record of a service.
func LookupSRV(service, proto, name string) (string, []*SRV, error) {
	return DefaultResolver.LookupSRV(context.Background(), service, proto, name)
}

// LookupAddr gives each name of an address.
func LookupAddr(addr string) ([]string, error) {
	return DefaultResolver.LookupAddr(context.Background(), addr)
}

// LookupPort gives the port of a service.
func LookupPort(network, service string) (int, error) {
	return DefaultResolver.LookupPort(context.Background(), network, service)
}
