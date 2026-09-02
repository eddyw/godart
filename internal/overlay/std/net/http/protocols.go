//go:build dart

package http

import "strings"

// Protocols is a set of HTTP protocols. The zero value holds no protocol.
//
// The 3 protocols are:
//
//   - HTTP1 is HTTP/1.0 and HTTP/1.1, over TCP and over TLS.
//   - HTTP2 is HTTP/2 over TLS.
//   - UnencryptedHTTP2 is HTTP/2 over TCP.
//
// The type holds a set only. Go also reads the set on `Server.Protocols` and
// on `Transport.Protocols`. The overlay holds neither field: the server reads
// the ALPN of the TLS listener that `crypto/tls` opened, and no path of the
// runtime carries HTTP/2 over a plain socket. A field that reads no value
// lies, thus the overlay leaves the 2 out.
type Protocols struct {
	bits uint8
}

// The bit of each protocol of the set.
const (
	protoHTTP1 = 1 << iota
	protoHTTP2
	protoUnencryptedHTTP2
)

// HTTP1 reports whether the set holds HTTP/1.
func (p Protocols) HTTP1() bool { return p.bits&protoHTTP1 != 0 }

// SetHTTP1 adds HTTP/1 to the set, or it takes HTTP/1 out.
func (p *Protocols) SetHTTP1(ok bool) { p.setBit(protoHTTP1, ok) }

// HTTP2 reports whether the set holds HTTP/2.
func (p Protocols) HTTP2() bool { return p.bits&protoHTTP2 != 0 }

// SetHTTP2 adds HTTP/2 to the set, or it takes HTTP/2 out.
func (p *Protocols) SetHTTP2(ok bool) { p.setBit(protoHTTP2, ok) }

// UnencryptedHTTP2 reports whether the set holds HTTP/2 over TCP.
func (p Protocols) UnencryptedHTTP2() bool { return p.bits&protoUnencryptedHTTP2 != 0 }

// SetUnencryptedHTTP2 adds HTTP/2 over TCP to the set, or it takes it out.
func (p *Protocols) SetUnencryptedHTTP2(ok bool) { p.setBit(protoUnencryptedHTTP2, ok) }

func (p *Protocols) setBit(bit uint8, ok bool) {
	if ok {
		p.bits |= bit
		return
	}
	p.bits &^= bit
}

// String gives the names of the set inside braces, for example {HTTP1,HTTP2}.
func (p Protocols) String() string {
	var s []string
	if p.HTTP1() {
		s = append(s, "HTTP1")
	}
	if p.HTTP2() {
		s = append(s, "HTTP2")
	}
	if p.UnencryptedHTTP2() {
		s = append(s, "UnencryptedHTTP2")
	}
	return "{" + strings.Join(s, ",") + "}"
}
