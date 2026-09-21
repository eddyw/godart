//go:build dart

package net

import (
	"errors"
	"io"
)

// This file holds the value types and the errors of the package that need no
// socket. Each 1 reads bytes or text alone, thus the Dart output matches Go
// exactly.

// Buffers holds 0 or more byte slices for a write in 1 operation.
//
// Go gives the slices to the writev call of the system for a connection of
// the network. Dart holds no writev, thus WriteTo always writes 1 slice at a
// time. The count and the error of the result stay equal.
type Buffers [][]byte

// WriteTo writes the content of the buffers to the writer, and it gives the
// count of bytes.
//
// WriteTo changes the slice v and each item v[i], and it changes no byte of
// an item.
func (v *Buffers) WriteTo(w io.Writer) (int64, error) {
	var n int64
	for _, b := range *v {
		nb, err := w.Write(b)
		n += int64(nb)
		if err != nil {
			v.consume(n)
			return n, err
		}
	}
	v.consume(n)
	return n, nil
}

// Read moves bytes of the buffers into the slice, and it gives the count. An
// empty buffer list gives io.EOF.
//
// Read changes the slice v and each item v[i], and it changes no byte of an
// item.
func (v *Buffers) Read(p []byte) (int, error) {
	var n int
	for len(p) > 0 && len(*v) > 0 {
		n0 := copy(p, (*v)[0])
		v.consume(int64(n0))
		p = p[n0:]
		n += n0
	}
	if len(*v) == 0 {
		return n, io.EOF
	}
	return n, nil
}

// consume drops the first n bytes of the buffers.
func (v *Buffers) consume(n int64) {
	for len(*v) > 0 {
		ln0 := int64(len((*v)[0]))
		if ln0 > n {
			(*v)[0] = (*v)[0][n:]
			return
		}
		n -= ln0
		(*v)[0] = nil
		*v = (*v)[1:]
	}
}

// hexDigit holds the 16 digits of the hexadecimal form.
const hexDigit = "0123456789abcdef"

// A HardwareAddr is a physical address of a network interface.
type HardwareAddr []byte

// String gives the text of the address, for example 00:00:5e:00:53:01.
func (a HardwareAddr) String() string {
	if len(a) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(a)*3-1)
	for i, b := range a {
		if i > 0 {
			buf = append(buf, ':')
		}
		buf = append(buf, hexDigit[b>>4])
		buf = append(buf, hexDigit[b&0xF])
	}
	return string(buf)
}

// ParseMAC reads the text of a hardware address. It takes a MAC-48 address,
// an EUI-48 address, an EUI-64 address, and a 20-octet address of IP over
// InfiniBand. It reads each of the 7 forms of Go:
//
//	00:00:5e:00:53:01
//	02:00:5e:10:00:00:00:01
//	00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01
//	00-00-5e-00-53-01
//	02-00-5e-10-00-00-00-01
//	00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01
//	0000.5e00.5301
//	0200.5e10.0000.0001
//	0000.0000.fe80.0000.0000.0000.0200.5e10.0000.0001
//	00005e005301
//
// Go writes the body with a goto, and the subset holds no goto. The body
// below reads the same 3 forms in a helper, and it gives the same values.
func ParseMAC(s string) (HardwareAddr, error) {
	hw, ok := parseMAC(s)
	if !ok {
		return nil, &AddrError{Err: "invalid MAC address", Addr: s}
	}
	return hw, nil
}

// parseMAC reads the 3 forms of a hardware address. It reports whether the
// text holds an address.
func parseMAC(s string) (HardwareAddr, bool) {
	if len(s) < 12 {
		return nil, false
	}
	if s[2] == ':' || s[2] == '-' {
		if (len(s)+1)%3 != 0 {
			return nil, false
		}
		n := (len(s) + 1) / 3
		if n != 6 && n != 8 && n != 20 {
			return nil, false
		}
		hw := make(HardwareAddr, n)
		for x, i := 0, 0; i < n; i++ {
			b, ok := xtoi2(s[x:], s[2])
			if !ok {
				return nil, false
			}
			hw[i] = b
			x += 3
		}
		return hw, true
	}
	if s[4] == '.' {
		if (len(s)+1)%5 != 0 {
			return nil, false
		}
		n := 2 * (len(s) + 1) / 5
		if n != 6 && n != 8 && n != 20 {
			return nil, false
		}
		hw := make(HardwareAddr, n)
		for x, i := 0, 0; i < n; i += 2 {
			b, ok := xtoi2(s[x:x+2], 0)
			if !ok {
				return nil, false
			}
			hw[i] = b
			b, ok = xtoi2(s[x+2:], s[4])
			if !ok {
				return nil, false
			}
			hw[i+1] = b
			x += 5
		}
		return hw, true
	}
	if len(s)%2 != 0 {
		return nil, false
	}
	n := len(s) / 2
	if n != 6 && n != 8 && n != 20 {
		return nil, false
	}
	hw := make(HardwareAddr, n)
	for x, i := 0, 0; i < n; i++ {
		b, ok := xtoi2(s[x:x+2], 0)
		if !ok {
			return nil, false
		}
		hw[i] = b
		x += 2
	}
	return hw, true
}

// xtoi2 reads the 2 hexadecimal digits at the start of the text. The byte e
// names the separator that must follow, and a value of 0 names no separator.
func xtoi2(s string, e byte) (byte, bool) {
	if len(s) > 2 && s[2] != e {
		return 0, false
	}
	if len(s) < 2 {
		return 0, false
	}
	hi, ok := hexValue(s[0])
	if !ok {
		return 0, false
	}
	lo, ok := hexValue(s[1])
	if !ok {
		return 0, false
	}
	return hi<<4 | lo, true
}

// hexValue gives the value of 1 hexadecimal digit.
func hexValue(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// Flags holds the state of a network interface.
type Flags uint

// The 6 flags of a network interface.
const (
	FlagUp           Flags = 1 << iota // the interface stands up
	FlagBroadcast                      // the interface holds a broadcast address
	FlagLoopback                       // the interface is a loopback interface
	FlagPointToPoint                   // the interface holds a point to point link
	FlagMulticast                      // the interface takes a multicast address
	FlagRunning                        // the interface runs
)

// flagNames names each of the 6 flags, in the order of the constants.
var flagNames = []string{
	"up",
	"broadcast",
	"loopback",
	"pointtopoint",
	"multicast",
	"running",
}

// String gives the names of the flags, with a bar between 2 names. A value of
// 0 gives the text "0".
func (f Flags) String() string {
	s := ""
	for i, name := range flagNames {
		if f&(1<<uint(i)) != 0 {
			if s != "" {
				s += "|"
			}
			s += name
		}
	}
	if s == "" {
		s = "0"
	}
	return s
}

// errClosed carries the message of a closed connection. The runtime gives the
// same text, thus netError wraps this value and errors.Is finds ErrClosed.
var errClosed = errors.New("use of closed network connection")

// ErrClosed is the error of an operation on a socket that a close already
// ended. A connection, a listener, and a datagram socket each give it. A test
// of the error takes the form errors.Is(err, net.ErrClosed).
var ErrClosed error = errClosed

// ErrWriteToConnected is the error of a WriteTo on a datagram socket that a
// dial connected.
var ErrWriteToConnected = errors.New("use of WriteTo with pre-connected connection")

// A DNSConfigError reports a failure of the read of the configuration of the
// resolver.
type DNSConfigError struct {
	Err error
}

// Unwrap gives the error of the read.
func (e *DNSConfigError) Unwrap() error { return e.Err }

// Error gives the message of the failure.
func (e *DNSConfigError) Error() string { return "error reading DNS config: " + e.Err.Error() }

// Timeout reports whether the failure came from a deadline. A configuration
// error holds none.
func (e *DNSConfigError) Timeout() bool { return false }

// Temporary reports whether a second try may work. A configuration error
// gives false.
func (e *DNSConfigError) Temporary() bool { return false }
