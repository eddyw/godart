//go:build dart

package net

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"syscall/dart"
	"time"
)

const (
	errNoSuchHost      = "no such host"
	errServerMisbehave = "server misbehaving"
)

// DNSError reports a failure of a lookup.
type DNSError struct {
	UnwrapErr   error
	Err         string
	Name        string
	Server      string
	IsTimeout   bool
	IsTemporary bool
	IsNotFound  bool
}

func (e *DNSError) Unwrap() error { return e.UnwrapErr }

func (e *DNSError) Error() string {
	if e == nil {
		return "<nil>"
	}
	s := "lookup " + e.Name
	if e.Server != "" {
		s += " on " + e.Server
	}
	s += ": " + e.Err
	return s
}

// Timeout reports whether the failure came from a deadline.
func (e *DNSError) Timeout() bool { return e != nil && e.IsTimeout }

// Temporary reports whether a second try may work.
func (e *DNSError) Temporary() bool { return e != nil && (e.IsTimeout || e.IsTemporary) }

// An MX holds 1 mail exchange record.
type MX struct {
	Host string
	Pref uint16
}

// An NS holds 1 name server record.
type NS struct {
	Host string
}

// An SRV holds 1 service record.
type SRV struct {
	Target   string
	Port     uint16
	Priority uint16
	Weight   uint16
}

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

// absName gives the name with a trailing dot.
func absName(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

// dialServer opens the connection to 1 server.
func (r *Resolver) dialServer(ctx context.Context, network, server string) (Conn, error) {
	if r != nil && r.Dial != nil {
		return r.Dial(ctx, network, server)
	}
	d := Dialer{}
	return d.DialContext(ctx, network, server)
}

// exchange sends 1 query to 1 server and reads the answer. A truncated answer
// takes a second exchange over TCP.
func (r *Resolver) exchange(ctx context.Context, server, name string, qtype uint16) (*dnsMessage, error) {
	id := uint16(rand.Uint32())
	query, err := dnsQuery(id, name, qtype)
	if err != nil {
		return nil, err
	}
	m, err := r.roundTrip(ctx, "udp", server, query, id)
	if err != nil {
		return nil, err
	}
	if !m.truncated {
		return m, nil
	}
	return r.roundTrip(ctx, "tcp", server, query, id)
}

// roundTrip writes 1 query and reads 1 answer. The TCP form carries a length
// of 2 bytes before the message, and the UDP form carries none.
func (r *Resolver) roundTrip(ctx context.Context, network, server string, query []byte, id uint16) (*dnsMessage, error) {
	c, err := r.dialServer(ctx, network, server)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if deadline, ok := ctx.Deadline(); ok {
		c.SetDeadline(deadline)
	} else {
		c.SetDeadline(time.Now().Add(getSystemDNSConfig().timeout))
	}
	body := query
	if network == "tcp" {
		body = append([]byte{byte(len(query) >> 8), byte(len(query))}, query...)
	}
	if _, err := c.Write(body); err != nil {
		return nil, err
	}
	answer, err := readAnswer(c, network)
	if err != nil {
		return nil, err
	}
	m, err := dnsParse(answer)
	if err != nil {
		return nil, err
	}
	if m.id != id {
		return nil, errors.New(errServerMisbehave)
	}
	return m, nil
}

// readAnswer reads 1 message of a connection.
func readAnswer(c Conn, network string) ([]byte, error) {
	if network == "udp" {
		// A datagram holds the whole message. 1232 bytes is the size that the
		// DNS flag day of 2020 names.
		buf := make([]byte, 1232)
		n, err := c.Read(buf)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}
	head := make([]byte, 2)
	if err := readFull(c, head); err != nil {
		return nil, err
	}
	length := int(head[0])<<8 | int(head[1])
	if length == 0 || length > dnsMaxMessage {
		return nil, errDNSMalformed
	}
	body := make([]byte, length)
	if err := readFull(c, body); err != nil {
		return nil, err
	}
	return body, nil
}

func readFull(c Conn, b []byte) error {
	for off := 0; off < len(b); {
		n, err := c.Read(b[off:])
		if n == 0 && err == nil {
			return errDNSMalformed
		}
		off += n
		if err != nil && off < len(b) {
			return err
		}
	}
	return nil
}

// query asks each candidate name at each server, and it gives the answers of
// the first name that holds one.
func (r *Resolver) query(ctx context.Context, name string, qtype uint16) ([]dnsRecord, string, error) {
	c := getSystemDNSConfig()
	var lastErr error
	lastServer := ""
	for _, candidate := range c.nameList(name) {
		for _, server := range c.servers {
			lastServer = server
			var m *dnsMessage
			var err error
			for try := 0; try < c.attempts; try++ {
				m, err = r.exchange(ctx, server, candidate, qtype)
				if err == nil {
					break
				}
				if ctx.Err() != nil {
					return nil, server, dnsError(ctx.Err().Error(), name, server, ctx.Err())
				}
			}
			if err != nil {
				lastErr = err
				continue
			}
			switch m.rcode {
			case dnsRcodeSuccess:
				if len(m.answers) == 0 {
					lastErr = errors.New(errNoSuchHost)
					continue
				}
				return m.answers, server, nil
			case dnsRcodeName:
				lastErr = errors.New(errNoSuchHost)
				continue
			default:
				lastErr = errors.New(errServerMisbehave)
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New(errNoSuchHost)
	}
	notFound := lastErr.Error() == errNoSuchHost
	return nil, lastServer, &DNSError{
		UnwrapErr:  lastErr,
		Err:        lastErr.Error(),
		Name:       name,
		Server:     lastServer,
		IsNotFound: notFound,
	}
}

func dnsError(text, name, server string, cause error) *DNSError {
	return &DNSError{UnwrapErr: cause, Err: text, Name: name, Server: server, IsTimeout: true}
}

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
	var types []uint16
	switch network {
	case "ip":
		types = []uint16{dnsTypeA, dnsTypeAAAA}
	case "ip4":
		types = []uint16{dnsTypeA}
	case "ip6":
		types = []uint16{dnsTypeAAAA}
	default:
		return nil, UnknownNetworkError(network)
	}
	// A numeric address needs no lookup, thus it reports no event. Go holds
	// the same order.
	if ip := ParseIP(host); ip != nil {
		if keepIP(network, ip) {
			return []IP{ip}, nil
		}
		return nil, &DNSError{Err: errNoSuchHost, Name: host, IsNotFound: true}
	}
	trace := traceOf(ctx)
	traceDNSStart(trace, host)
	order, _ := systemConf().hostLookupOrder(r, host)
	out, err := r.lookupIP(ctx, types, network, host, order)
	if order != hostLookupCgo && len(out) > 1 {
		addrs := make([]IPAddr, len(out))
		for i, ip := range out {
			addrs[i] = IPAddr{IP: ip}
		}
		sortByRFC6724(addrs)
		for i, addr := range addrs {
			out[i] = addr.IP
		}
	}
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
			IsNotFound:  cause == errNoSuchHost,
			IsTemporary: cause == errServerMisbehave,
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
		return nil, &DNSError{Err: errNoSuchHost, Name: host, IsNotFound: true}
	}
	return out, nil
}

func (r *Resolver) lookupIP(ctx context.Context, types []uint16, network, host string, order hostLookupOrder) ([]IP, error) {
	if order == hostLookupCgo {
		return systemLookupIP(network, host)
	}
	if order == hostLookupFilesDNS || order == hostLookupFiles {
		if out := lookupStaticIPs(network, host); len(out) > 0 {
			return out, nil
		}
		if order == hostLookupFiles {
			return nil, &DNSError{Err: errNoSuchHost, Name: host, IsNotFound: true}
		}
	}
	var out []IP
	var lastErr error
	for _, qtype := range types {
		answers, _, err := r.query(ctx, host, qtype)
		if err != nil {
			lastErr = err
			continue
		}
		for _, a := range answers {
			if a.rtype == qtype {
				out = append(out, a.ip)
			}
		}
	}
	if len(out) == 0 {
		if order == hostLookupDNSFiles {
			if out := lookupStaticIPs(network, host); len(out) > 0 {
				return out, nil
			}
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, &DNSError{Err: errNoSuchHost, Name: host, IsNotFound: true}
	}
	return out, nil
}

func lookupStaticIPs(network, host string) []IP {
	var out []IP
	addresses, _ := lookupStaticHost(host)
	for _, text := range addresses {
		if ip := ParseIP(text); ip != nil && keepIP(network, ip) {
			out = append(out, ip)
		}
	}
	return out
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

// LookupCNAME gives the canonical name of a host. A host that holds no CNAME
// record gives its own name.
func (r *Resolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	answers, _, err := r.query(ctx, host, dnsTypeCNAME)
	if err != nil {
		var dnsErr *DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return absName(host), nil
		}
		return "", err
	}
	for _, a := range answers {
		if a.rtype == dnsTypeCNAME {
			return a.target, nil
		}
	}
	return absName(host), nil
}

// LookupMX gives each mail exchange record of a name, in the order of the
// preference.
func (r *Resolver) LookupMX(ctx context.Context, name string) ([]*MX, error) {
	answers, _, err := r.query(ctx, name, dnsTypeMX)
	if err != nil {
		return nil, err
	}
	var out []*MX
	for _, a := range answers {
		if a.rtype == dnsTypeMX {
			out = append(out, &MX{Host: a.target, Pref: a.pref})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Pref < out[j].Pref })
	return out, nil
}

// LookupNS gives each name server record of a name.
func (r *Resolver) LookupNS(ctx context.Context, name string) ([]*NS, error) {
	answers, _, err := r.query(ctx, name, dnsTypeNS)
	if err != nil {
		return nil, err
	}
	var out []*NS
	for _, a := range answers {
		if a.rtype == dnsTypeNS {
			out = append(out, &NS{Host: a.target})
		}
	}
	return out, nil
}

// LookupTXT gives the text of each TXT record of a name.
func (r *Resolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	answers, _, err := r.query(ctx, name, dnsTypeTXT)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, a := range answers {
		if a.rtype == dnsTypeTXT {
			out = append(out, a.txt)
		}
	}
	return out, nil
}

// LookupSRV gives the canonical name and each service record of a service.
//
// A service and a protocol that each hold a text give the name
// `_service._proto.name`. A pair of empty texts reads the name itself, thus
// the caller writes the whole name.
func (r *Resolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*SRV, error) {
	target := name
	if service != "" || proto != "" {
		target = "_" + service + "._" + proto + "." + name
	}
	answers, _, err := r.query(ctx, target, dnsTypeSRV)
	if err != nil {
		return "", nil, err
	}
	var out []*SRV
	for _, a := range answers {
		if a.rtype == dnsTypeSRV {
			out = append(out, &SRV{
				Target:   a.target,
				Port:     a.port,
				Priority: a.priority,
				Weight:   a.weight,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Priority < out[j].Priority })
	cname := absName(target)
	if len(answers) > 0 {
		cname = absName(answers[0].name)
	}
	return cname, out, nil
}

// LookupAddr gives each name of an address.
func (r *Resolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	if held := lookupStaticAddr(addr); len(held) > 0 {
		return held, nil
	}
	name, err := reverseName(addr)
	if err != nil {
		return nil, &DNSError{Err: err.Error(), Name: addr}
	}
	answers, _, err := r.query(ctx, name, dnsTypePTR)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, a := range answers {
		if a.rtype == dnsTypePTR {
			out = append(out, a.target)
		}
	}
	return out, nil
}

// reverseName gives the name of the reverse zone of an address.
func reverseName(addr string) (string, error) {
	ip := ParseIP(addr)
	if ip == nil {
		return "", errors.New("unrecognized address " + addr)
	}
	if v4 := ip.To4(); v4 != nil {
		return strconv.Itoa(int(v4[3])) + "." + strconv.Itoa(int(v4[2])) + "." +
			strconv.Itoa(int(v4[1])) + "." + strconv.Itoa(int(v4[0])) + ".in-addr.arpa.", nil
	}
	const digits = "0123456789abcdef"
	v6 := ip.To16()
	out := make([]byte, 0, 72)
	for i := len(v6) - 1; i >= 0; i-- {
		out = append(out, digits[v6[i]&0x0f], '.', digits[v6[i]>>4], '.')
	}
	return string(out) + "ip6.arpa.", nil
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
			return 0, &DNSError{Err: "unknown port", Name: name, IsNotFound: true}
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
