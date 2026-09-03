//go:build dart

package x509

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"

	"encoding/asn1"
)

var errInvalidOID = errors.New("invalid oid")

// An OID holds an ASN.1 OBJECT IDENTIFIER.
type OID struct {
	der []byte
}

// ParseOID reads an Object Identifier from a text of ASCII numbers. A dot
// stands between each 2 numbers.
func ParseOID(oid string) (OID, error) {
	var o OID
	// Go writes `return o, o.unmarshalOIDText(oid)`. That form reads o and it
	// calls the method in an order that the specification leaves open, thus
	// the 2 statements below hold the order that the result needs.
	err := o.unmarshalOIDText(oid)
	return o, err
}

// newOIDFromDER builds an OID from the DER content of an OBJECT IDENTIFIER.
// It reports whether the content holds a valid form.
func newOIDFromDER(der []byte) (OID, bool) {
	if len(der) == 0 || der[len(der)-1]&0x80 != 0 {
		return OID{}, false
	}

	start := 0
	for i, v := range der {
		// ITU-T X.690, section 8.19.2:
		// A subidentifier takes the fewest possible octets. Thus the first
		// octet of a subidentifier never holds the value 0x80.
		if i == start && v == 0x80 {
			return OID{}, false
		}
		if v&0x80 == 0 {
			start = i + 1
		}
	}

	return OID{der}, true
}

// OIDFromInts builds an OID from a list of integers. Each integer gives 1
// component.
func OIDFromInts(oid []uint64) (OID, error) {
	if len(oid) < 2 || oid[0] > 2 || (oid[0] < 2 && oid[1] >= 40) {
		return OID{}, errInvalidOID
	}

	length := base128IntLength(oid[0]*40 + oid[1])
	for _, v := range oid[2:] {
		length += base128IntLength(v)
	}

	der := make([]byte, 0, length)
	der = appendBase128Int(der, oid[0]*40+oid[1])
	for _, v := range oid[2:] {
		der = appendBase128Int(der, v)
	}
	return OID{der}, nil
}

// base128IntLength gives the count of octets that the base 128 form of n
// takes.
func base128IntLength(n uint64) int {
	if n == 0 {
		return 1
	}
	return (bits.Len64(n) + 6) / 7
}

// appendBase128Int adds the base 128 form of n to dst.
func appendBase128Int(dst []byte, n uint64) []byte {
	for i := base128IntLength(n) - 1; i >= 0; i-- {
		o := byte(n >> uint(i*7))
		o &= 0x7f
		if i != 0 {
			o |= 0x80
		}
		dst = append(dst, o)
	}
	return dst
}

// base128BigIntLength gives the count of octets that the base 128 form of n
// takes.
func base128BigIntLength(n *big.Int) int {
	if n.Cmp(big.NewInt(0)) == 0 {
		return 1
	}
	return (n.BitLen() + 6) / 7
}

// appendBase128BigInt adds the base 128 form of n to dst.
func appendBase128BigInt(dst []byte, n *big.Int) []byte {
	if n.Cmp(big.NewInt(0)) == 0 {
		return append(dst, 0)
	}

	for i := base128BigIntLength(n) - 1; i >= 0; i-- {
		o := byte(big.NewInt(0).Rsh(n, uint(i)*7).Bits()[0])
		o &= 0x7f
		if i != 0 {
			o |= 0x80
		}
		dst = append(dst, o)
	}
	return dst
}

// AppendText adds the text form of the Object Identifier to b. It gives the
// method of [encoding.TextAppender].
func (o OID) AppendText(b []byte) ([]byte, error) {
	return append(b, o.String()...), nil
}

// MarshalText gives the text form of the Object Identifier. It gives the
// method of [encoding.TextMarshaler].
func (o OID) MarshalText() ([]byte, error) {
	return o.AppendText(nil)
}

// UnmarshalText reads the text form of an Object Identifier. It gives the
// method of [encoding.TextUnmarshaler].
func (o *OID) UnmarshalText(text []byte) error {
	return o.unmarshalOIDText(string(text))
}

// unmarshalOIDText reads a text of ASCII numbers into the receiver.
func (o *OID) unmarshalOIDText(oid string) error {
	// (*big.Int).SetString accepts a + sign and a - sign. The text form of an
	// Object Identifier holds no sign, thus this loop rejects each such text.
	for _, c := range oid {
		isDigit := c >= '0' && c <= '9'
		if !isDigit && c != '.' {
			return errInvalidOID
		}
	}

	var (
		firstNum  string
		secondNum string
	)

	var nextComponentExists bool
	firstNum, oid, nextComponentExists = strings.Cut(oid, ".")
	if !nextComponentExists {
		return errInvalidOID
	}
	secondNum, oid, nextComponentExists = strings.Cut(oid, ".")

	var (
		first  = big.NewInt(0)
		second = big.NewInt(0)
	)

	if _, ok := first.SetString(firstNum, 10); !ok {
		return errInvalidOID
	}
	if _, ok := second.SetString(secondNum, 10); !ok {
		return errInvalidOID
	}

	if first.Cmp(big.NewInt(2)) > 0 || (first.Cmp(big.NewInt(2)) < 0 && second.Cmp(big.NewInt(40)) >= 0) {
		return errInvalidOID
	}

	firstComponent := first.Mul(first, big.NewInt(40))
	firstComponent.Add(firstComponent, second)

	der := appendBase128BigInt(make([]byte, 0, 32), firstComponent)

	for nextComponentExists {
		var strNum string
		strNum, oid, nextComponentExists = strings.Cut(oid, ".")
		b, ok := big.NewInt(0).SetString(strNum, 10)
		if !ok {
			return errInvalidOID
		}
		der = appendBase128BigInt(der, b)
	}

	o.der = der
	return nil
}

// AppendBinary adds the DER form of the Object Identifier to b. It gives the
// method of [encoding.BinaryAppender].
func (o OID) AppendBinary(b []byte) ([]byte, error) {
	return append(b, o.der...), nil
}

// MarshalBinary gives the DER form of the Object Identifier. It gives the
// method of [encoding.BinaryMarshaler].
func (o OID) MarshalBinary() ([]byte, error) {
	return o.AppendBinary(nil)
}

// UnmarshalBinary reads the DER form of an Object Identifier. It gives the
// method of [encoding.BinaryUnmarshaler].
func (o *OID) UnmarshalBinary(b []byte) error {
	oid, ok := newOIDFromDER(bytes.Clone(b))
	if !ok {
		return errInvalidOID
	}
	*o = oid
	return nil
}

// Equal reports whether the 2 values hold the same Object Identifier.
func (oid OID) Equal(other OID) bool {
	// Each unique Object Identifier holds 1 DER form alone.
	return bytes.Equal(oid.der, other.der)
}

// parseBase128Int reads 1 base 128 integer of bytes at initOffset. It gives
// the value, the offset after the integer, and a report of a failure.
func parseBase128Int(bytes []byte, initOffset int) (ret, offset int, failed bool) {
	offset = initOffset
	var ret64 int64
	for shifted := 0; offset < len(bytes); shifted++ {
		// 5 octets hold 35 bits of data. Thus the form is not minimal, or the
		// value is too large for an int32.
		if shifted == 5 {
			failed = true
			return
		}
		ret64 <<= 7
		b := bytes[offset]
		// An integer takes the fewest possible octets. Thus the first octet
		// never holds the value 0x80.
		if shifted == 0 && b == 0x80 {
			failed = true
			return
		}
		ret64 |= int64(b & 0x7f)
		offset++
		if b&0x80 == 0 {
			ret = int(ret64)
			// The value must fit an int on each platform.
			if ret64 > math.MaxInt32 {
				failed = true
			}
			return
		}
	}
	failed = true
	return
}

// EqualASN1OID reports whether the OID equals an asn1.ObjectIdentifier. An
// asn1.ObjectIdentifier holds a component of 31 bits at most. The method
// gives false when a component of the OID needs more bits.
func (oid OID) EqualASN1OID(other asn1.ObjectIdentifier) bool {
	if len(other) < 2 {
		return false
	}
	v, offset, failed := parseBase128Int(oid.der, 0)
	if failed {
		// The parse of the OID came first, thus this branch never runs. The
		// guard stays for safety.
		return false
	}
	if v < 80 {
		a, b := v/40, v%40
		if other[0] != a || other[1] != b {
			return false
		}
	} else {
		a, b := 2, v-80
		if other[0] != a || other[1] != b {
			return false
		}
	}

	i := 2
	for ; offset < len(oid.der); i++ {
		v, offset, failed = parseBase128Int(oid.der, offset)
		if failed {
			// This branch never runs, for the same cause as above.
			return false
		}
		if i >= len(other) || v != other[i] {
			return false
		}
	}

	return i == len(other)
}

// String gives the text form of the Object Identifier.
func (oid OID) String() string {
	var b strings.Builder
	b.Grow(32)
	const (
		valSize         = 64 // The count of bits of val.
		bitsPerByte     = 7
		maxValSafeShift = (1 << (valSize - bitsPerByte)) - 1
	)
	var (
		start    = 0
		val      = uint64(0)
		numBuf   = make([]byte, 0, 21)
		bigVal   *big.Int
		overflow bool
	)
	for i, v := range oid.der {
		curVal := v & 0x7F
		valEnd := v&0x80 == 0
		if valEnd {
			if start != 0 {
				b.WriteByte('.')
			}
		}
		if !overflow && val > maxValSafeShift {
			if bigVal == nil {
				bigVal = new(big.Int)
			}
			bigVal = bigVal.SetUint64(val)
			overflow = true
		}
		if overflow {
			bigVal = bigVal.Lsh(bigVal, bitsPerByte).Or(bigVal, big.NewInt(int64(curVal)))
			if valEnd {
				if start == 0 {
					b.WriteString("2.")
					bigVal = bigVal.Sub(bigVal, big.NewInt(80))
				}
				numBuf = bigVal.Append(numBuf, 10)
				b.Write(numBuf)
				numBuf = numBuf[:0]
				val = 0
				start = i + 1
				overflow = false
			}
			continue
		}
		val <<= bitsPerByte
		val |= uint64(curVal)
		if valEnd {
			if start == 0 {
				if val < 80 {
					b.Write(strconv.AppendUint(numBuf, val/40, 10))
					b.WriteByte('.')
					b.Write(strconv.AppendUint(numBuf, val%40, 10))
				} else {
					b.WriteString("2.")
					b.Write(strconv.AppendUint(numBuf, val-80, 10))
				}
			} else {
				b.Write(strconv.AppendUint(numBuf, val, 10))
			}
			val = 0
			start = i + 1
		}
	}
	return b.String()
}

// toASN1OID gives the asn1.ObjectIdentifier form of the OID. It reports
// whether each component fits 31 bits.
func (oid OID) toASN1OID() (asn1.ObjectIdentifier, bool) {
	out := make([]int, 0, len(oid.der)+1)

	const (
		valSize         = 31 // The count of usable bits of val for an OID.
		bitsPerByte     = 7
		maxValSafeShift = (1 << (valSize - bitsPerByte)) - 1
	)

	val := 0

	for _, v := range oid.der {
		if val > maxValSafeShift {
			return nil, false
		}

		val <<= bitsPerByte
		val |= int(v & 0x7F)

		if v&0x80 == 0 {
			if len(out) == 0 {
				if val < 80 {
					out = append(out, val/40)
					out = append(out, val%40)
				} else {
					out = append(out, 2)
					out = append(out, val-80)
				}
				val = 0
				continue
			}
			out = append(out, val)
			val = 0
		}
	}

	return out, true
}

// OIDFromASN1OID builds an OID from an asn1.ObjectIdentifier.
func OIDFromASN1OID(asn1OID asn1.ObjectIdentifier) (OID, error) {
	uint64OID := make([]uint64, 0, len(asn1OID))
	for _, component := range asn1OID {
		if component < 0 {
			return OID{}, errors.New("x509: OID components must be non-negative")
		}
		uint64OID = append(uint64OID, uint64(component))
	}
	return OIDFromInts(uint64OID)
}
