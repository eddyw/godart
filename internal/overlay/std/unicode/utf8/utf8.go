//go:build dart

// Package utf8 holds the API of the standard library package unicode/utf8.
//
// The compiler cannot read the real source. The function word of that package
// takes a type parameter with the constraint `string | []byte`, and the 2 terms
// give 2 Dart types. This file holds the same behavior inside the subset.
package utf8

const (
	RuneError = '�'
	RuneSelf  = 0x80
	MaxRune   = '\U0010FFFF'
	UTFMax    = 4
)

// surrogateMin and surrogateMax bound the code points that UTF-16 reserves.
// UTF-8 never encodes such a value.
const (
	surrogateMin = 0xD800
	surrogateMax = 0xDFFF
)

const (
	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1
)

// need gives the count of bytes of a sequence that starts with b. The result is
// 0 when b cannot start a sequence.
func need(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b < 0xC2:
		return 0
	case b < 0xE0:
		return 2
	case b < 0xF0:
		return 3
	case b < 0xF5:
		return 4
	}
	return 0
}

// low gives the smallest code point that a sequence of n bytes may hold. A
// shorter sequence holds the same value, and Go rejects such a form.
func low(n int) rune {
	switch n {
	case 2:
		return rune1Max + 1
	case 3:
		return rune2Max + 1
	case 4:
		return rune3Max + 1
	}
	return 0
}

func continuation(b byte) bool { return b&0xC0 == 0x80 }

// decode reads 1 rune from the bytes. It gives RuneError with the size 1 when
// the bytes hold no valid sequence, and it gives the size 0 for an empty input.
func decode(read func(int) byte, length int) (rune, int) {
	if length < 1 {
		return RuneError, 0
	}
	size := need(read(0))
	if size == 0 {
		return RuneError, 1
	}
	if size == 1 {
		return rune(read(0)), 1
	}
	if length < size {
		return RuneError, 1
	}
	r := rune(read(0) & byte(0x7F>>uint(size)))
	for i := 1; i < size; i++ {
		b := read(i)
		if !continuation(b) {
			return RuneError, 1
		}
		r = r<<6 | rune(b&0x3F)
	}
	if r < low(size) || !ValidRune(r) {
		return RuneError, 1
	}
	return r, size
}

// decodeLast reads the last rune of the bytes.
//
// A sequence takes 4 bytes at most, thus the search walks back 4 bytes. Go
// gives RuneError with the size 1 when no start byte holds the tail.
func decodeLast(read func(int) byte, length int) (rune, int) {
	if length < 1 {
		return RuneError, 0
	}
	start := length - 1
	if read(start) < RuneSelf {
		return rune(read(start)), 1
	}
	limit := length - UTFMax
	if limit < 0 {
		limit = 0
	}
	for start--; start >= limit; start-- {
		if !continuation(read(start)) {
			break
		}
	}
	if start < limit {
		return RuneError, 1
	}
	r, size := decode(func(i int) byte { return read(start + i) }, length-start)
	if start+size != length {
		return RuneError, 1
	}
	return r, size
}

func byteAt(p []byte) func(int) byte {
	return func(i int) byte { return p[i] }
}

func charAt(s string) func(int) byte {
	return func(i int) byte { return s[i] }
}

func DecodeRune(p []byte) (rune, int) { return decode(byteAt(p), len(p)) }

func DecodeRuneInString(s string) (rune, int) { return decode(charAt(s), len(s)) }

func DecodeLastRune(p []byte) (rune, int) { return decodeLast(byteAt(p), len(p)) }

func DecodeLastRuneInString(s string) (rune, int) { return decodeLast(charAt(s), len(s)) }

// full reports whether the bytes start with a complete sequence. An invalid
// sequence counts as complete, because a longer input does not repair it.
func full(read func(int) byte, length int) bool {
	if length == 0 {
		return false
	}
	size := need(read(0))
	if size == 0 {
		return true
	}
	if length >= size {
		return true
	}
	for i := 1; i < length; i++ {
		if !continuation(read(i)) {
			return true
		}
	}
	return false
}

func FullRune(p []byte) bool { return full(byteAt(p), len(p)) }

func FullRuneInString(s string) bool { return full(charAt(s), len(s)) }

func RuneLen(r rune) int {
	switch {
	case r < 0:
		return -1
	case r <= rune1Max:
		return 1
	case r <= rune2Max:
		return 2
	case surrogateMin <= r && r <= surrogateMax:
		return -1
	case r <= rune3Max:
		return 3
	case r <= MaxRune:
		return 4
	}
	return -1
}

func EncodeRune(p []byte, r rune) int {
	if !ValidRune(r) {
		r = RuneError
	}
	switch {
	case r <= rune1Max:
		p[0] = byte(r)
		return 1
	case r <= rune2Max:
		p[0] = 0xC0 | byte(r>>6)
		p[1] = 0x80 | byte(r)&0x3F
		return 2
	case r <= rune3Max:
		p[0] = 0xE0 | byte(r>>12)
		p[1] = 0x80 | byte(r>>6)&0x3F
		p[2] = 0x80 | byte(r)&0x3F
		return 3
	}
	p[0] = 0xF0 | byte(r>>18)
	p[1] = 0x80 | byte(r>>12)&0x3F
	p[2] = 0x80 | byte(r>>6)&0x3F
	p[3] = 0x80 | byte(r)&0x3F
	return 4
}

func AppendRune(p []byte, r rune) []byte {
	if uint32(r) <= rune1Max {
		return append(p, byte(r))
	}
	var buf [UTFMax]byte
	n := EncodeRune(buf[:], r)
	return append(p, buf[:n]...)
}

func RuneCount(p []byte) int {
	n := 0
	for i := 0; i < len(p); n++ {
		_, size := decode(func(k int) byte { return p[i+k] }, len(p)-i)
		i += size
	}
	return n
}

func RuneCountInString(s string) int {
	n := 0
	for i := 0; i < len(s); n++ {
		_, size := decode(func(k int) byte { return s[i+k] }, len(s)-i)
		i += size
	}
	return n
}

func RuneStart(b byte) bool { return b&0xC0 != 0x80 }

func Valid(p []byte) bool {
	for i := 0; i < len(p); {
		r, size := decode(func(k int) byte { return p[i+k] }, len(p)-i)
		if r == RuneError && size == 1 {
			return false
		}
		i += size
	}
	return true
}

func ValidString(s string) bool {
	for i := 0; i < len(s); {
		r, size := decode(func(k int) byte { return s[i+k] }, len(s)-i)
		if r == RuneError && size == 1 {
			return false
		}
		i += size
	}
	return true
}

func ValidRune(r rune) bool {
	return 0 <= r && r < surrogateMin || surrogateMax < r && r <= MaxRune
}
