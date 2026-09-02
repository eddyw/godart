//go:build dart

// Package strconv converts between a Go string and a basic value.
package strconv

import (
	"errors"
	"syscall/dart"
	"unicode"
	"unicode/utf8"
)

// IntSize is the count of bits of an int. A Dart int holds 64 bits on the VM
// and with AOT.
const IntSize = 64

// ErrSyntax reports text that holds no value of the type.
var ErrSyntax = errors.New("invalid syntax")

// ErrRange reports a value outside the range of the type.
var ErrRange = errors.New("value out of range")

// A NumError names the function, the text, and the reason of a failure.
type NumError struct {
	Func string
	Num  string
	Err  error
}

func (e *NumError) Error() string {
	return "strconv." + e.Func + ": parsing " + Quote(e.Num) + ": " + e.Err.Error()
}

func (e *NumError) Unwrap() error { return e.Err }

// Itoa gives the decimal text of i.
func Itoa(i int) string {
	return FormatInt(int64(i), 10)
}

// FormatInt gives the text of i in the base.
func FormatInt(i int64, base int) string {
	if base < 2 || base > 36 {
		panic("strconv: illegal base")
	}
	if i == 0 {
		return "0"
	}
	negative := i < 0
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	out := ""
	v := i
	if negative {
		v = -v
	}
	for v > 0 {
		out = string(digits[v%int64(base)]) + out
		v /= int64(base)
	}
	if negative {
		return "-" + out
	}
	return out
}

// AppendInt adds the text of i in the base to dst.
func AppendInt(dst []byte, i int64, base int) []byte {
	return append(dst, FormatInt(i, base)...)
}

// AppendUint adds the text of i in the base to dst.
func AppendUint(dst []byte, i uint64, base int) []byte {
	return append(dst, FormatUint(i, base)...)
}

// AppendBool adds "true" or "false" to dst.
func AppendBool(dst []byte, b bool) []byte {
	return append(dst, FormatBool(b)...)
}

// AppendFloat adds the text of f to dst.
func AppendFloat(dst []byte, f float64, verb byte, precision, bitSize int) []byte {
	return append(dst, FormatFloat(f, verb, precision, bitSize)...)
}

// FormatBool gives "true" or "false".
func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// FormatFloat gives the text of f. The verb is one of 'f', 'e', and 'g', and a
// negative precision gives the shortest text.
func FormatFloat(f float64, verb byte, precision, bitSize int) string {
	// The verb f with the precision -1 gives the shortest fixed text with no
	// exponent. The bridge verb f writes 6 digits, exactly as fmt does.
	if precision < 0 && verb == 'f' {
		if bitSize == 32 {
			return expandExponent(dart.Format("v", float32(f)))
		}
		return expandExponent(dart.Format("v", f))
	}
	form := string(rune(verb))
	if precision >= 0 {
		form = "." + Itoa(precision) + form
	}
	// A float32 value reaches the bridge as a float32, thus the shortest text
	// of the verb g holds the digits of 32 bits and no more.
	if bitSize == 32 {
		return dart.Format(form, float32(f))
	}
	return dart.Format(form, f)
}

// expandExponent unrolls the exponent form of a shortest text into the fixed
// form.
func expandExponent(text string) string {
	e := -1
	for i := 0; i < len(text); i++ {
		if text[i] == 'e' {
			e = i
			break
		}
	}
	if e < 0 {
		return text
	}
	mant := text[:e]
	exp, err := Atoi(text[e+1:])
	if err != nil {
		return text
	}
	sign := ""
	if len(mant) > 0 && mant[0] == '-' {
		sign = "-"
		mant = mant[1:]
	}
	dot := -1
	for i := 0; i < len(mant); i++ {
		if mant[i] == '.' {
			dot = i
			break
		}
	}
	digits := mant
	point := len(mant)
	if dot >= 0 {
		digits = mant[:dot] + mant[dot+1:]
		point = dot
	}
	point += exp
	if point <= 0 {
		out := sign + "0."
		for i := 0; i < -point; i++ {
			out += "0"
		}
		return out + digits
	}
	if point >= len(digits) {
		out := sign + digits
		for i := len(digits); i < point; i++ {
			out += "0"
		}
		return out
	}
	return sign + digits[:point] + "." + digits[point:]
}

// Atoi reads a decimal int from text.
func Atoi(text string) (int, error) {
	v, err := ParseInt(text, 10, 64)
	if err != nil {
		return 0, &NumError{Func: "Atoi", Num: text, Err: ErrSyntax}
	}
	return int(v), nil
}

// ParseInt reads an int from text in the base.
func ParseInt(text string, base, bitSize int) (int64, error) {
	if len(text) == 0 {
		return 0, &NumError{Func: "ParseInt", Num: text, Err: ErrSyntax}
	}
	i := 0
	negative := false
	if text[0] == '+' || text[0] == '-' {
		negative = text[0] == '-'
		i = 1
	}
	out, err := parseDigits(text, i, base, "ParseInt")
	if err != nil {
		return 0, err
	}
	if negative {
		return -out, nil
	}
	return out, nil
}

// ParseUint reads an unsigned value from text.
func ParseUint(text string, base, bitSize int) (uint64, error) {
	out, err := parseDigits(text, 0, base, "ParseUint")
	if err != nil {
		return 0, err
	}
	return uint64(out), nil
}

// parseDigits reads the digits of a value. A base of 0 reads the prefix of the
// text, exactly as Go does, and it accepts an underscore between 2 digits.
func parseDigits(text string, i, base int, name string) (int64, error) {
	if i >= len(text) {
		return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
	}
	underscore := false
	if base == 0 {
		underscore = true
		base = 10
		if text[i] == '0' {
			if i+1 == len(text) {
				return 0, nil
			}
			switch text[i+1] {
			case 'x', 'X':
				base, i = 16, i+2
			case 'b', 'B':
				base, i = 2, i+2
			case 'o', 'O':
				base, i = 8, i+2
			default:
				base, i = 8, i+1
			}
		}
	}
	if i >= len(text) {
		return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
	}
	var out int64
	digits := 0
	for ; i < len(text); i++ {
		if underscore && text[i] == '_' {
			continue
		}
		digit := digitValue(text[i])
		if digit < 0 || digit >= base {
			return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
		}
		out = out*int64(base) + int64(digit)
		digits++
	}
	if digits == 0 {
		return 0, &NumError{Func: name, Num: text, Err: ErrSyntax}
	}
	return out, nil
}

// ParseBool reads a bool from text.
func ParseBool(text string) (bool, error) {
	switch text {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, &NumError{Func: "ParseBool", Num: text, Err: ErrSyntax}
}

// ParseFloat reads a float from text.
func ParseFloat(text string, bitSize int) (float64, error) {
	v, ok := dart.ParseFloat(text)
	if !ok {
		return 0, &NumError{Func: "ParseFloat", Num: text, Err: ErrSyntax}
	}
	return v, nil
}

// ParseComplex reads a complex value from text. The bit size is 64 for a
// complex64 or 128 for a complex128, and each part takes half of it.
//
// The text holds the form N, Ni, or N±Ni, and a pair of parentheses may wrap
// it. A space anywhere makes the text invalid.
func ParseComplex(text string, bitSize int) (complex128, error) {
	c, err := parseComplex(text, bitSize)
	if err != nil {
		return c, &NumError{Func: "ParseComplex", Num: text, Err: err}
	}
	return c, nil
}

func parseComplex(text string, bitSize int) (complex128, error) {
	size := 64
	if bitSize == 64 {
		size = 32
	}
	s := text
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		s = s[1 : len(s)-1]
	}

	// A part outside the range gives an infinity and a pending error. The other
	// part still reads, thus a bad pair reports the first cause only.
	var pending error
	re, n, err := parseFloatPrefix(s, size)
	if err != nil {
		if err != ErrRange {
			return 0, err
		}
		pending = err
	}
	s = s[n:]
	if len(s) == 0 {
		return complex(re, 0), pending
	}

	switch s[0] {
	case '+':
		// A single plus belongs to the imaginary part, thus "1+NaNi" reads. A
		// double plus stays, because it keeps the error of the syntax.
		if len(s) > 1 && s[1] != '+' {
			s = s[1:]
		}
	case '-':
	case 'i':
		if len(s) == 1 {
			return complex(0, re), pending
		}
		return 0, ErrSyntax
	default:
		return 0, ErrSyntax
	}

	im, n, err := parseFloatPrefix(s, size)
	if err != nil {
		if err != ErrRange {
			return 0, err
		}
		pending = err
	}
	s = s[n:]
	if s != "i" {
		return 0, ErrSyntax
	}
	return complex(re, im), pending
}

// FormatComplex gives the text of c in the form (a+bi). The verb and the
// precision hold the meaning of FormatFloat. The bit size is 64 for a
// complex64 or 128 for a complex128.
func FormatComplex(c complex128, verb byte, precision, bitSize int) string {
	if bitSize != 64 && bitSize != 128 {
		panic("invalid bitSize")
	}
	size := bitSize >> 1
	im := FormatFloat(imag(c), verb, precision, size)
	if im[0] != '+' && im[0] != '-' {
		im = "+" + im
	}
	return "(" + FormatFloat(real(c), verb, precision, size) + im + "i)"
}

// parseFloatPrefix reads the float at the start of s, and it gives the count of
// bytes that it takes. ParseComplex needs the count, because the imaginary part
// follows the real part with no separator.
func parseFloatPrefix(s string, bitSize int) (float64, int, error) {
	if f, n, ok := specialFloat(s); ok {
		return f, n, nil
	}
	mantissa, exp, neg, trunc, isHex, n, ok := readFloat(s)
	if !ok {
		return 0, n, ErrSyntax
	}
	if isHex {
		f, err := hexFloat(mantissa, exp, neg, trunc, bitSize)
		return f, n, err
	}
	f, err := decimalFloat(s[:n], bitSize)
	return f, n, err
}

// specialFloat reads inf, infinity, or NaN at the start of s, in any case of
// the letters. Go accepts a sign before an infinity only.
func specialFloat(s string) (float64, int, bool) {
	if len(s) == 0 {
		return 0, 0, false
	}
	sign := 1
	signLen := 0
	first := s[0]
	if first == '+' || first == '-' {
		if first == '-' {
			sign = -1
		}
		signLen = 1
		s = s[1:]
		first = 'i'
	}
	switch first {
	case 'i', 'I':
		n := prefixLenLower(s, "infinity")
		// Anything longer than inf reads, and a text short of infinity takes
		// the 3 letters of inf only.
		if 3 < n && n < 8 {
			n = 3
		}
		if n == 3 || n == 8 {
			if sign < 0 {
				return dart.Float64FromBits(0xFFF0000000000000), signLen + n, true
			}
			return dart.Float64FromBits(0x7FF0000000000000), signLen + n, true
		}
	case 'n', 'N':
		if prefixLenLower(s, "nan") == 3 {
			return dart.Float64FromBits(0x7FF8000000000001), 3, true
		}
	}
	return 0, 0, false
}

// prefixLenLower gives the length of the common prefix of s and a lower case
// text, and it ignores the case of s.
func prefixLenLower(s, prefix string) int {
	n := len(prefix)
	if len(s) < n {
		n = len(s)
	}
	for i := 0; i < n; i++ {
		if lower(s[i]) != prefix[i] {
			return i
		}
	}
	return n
}

func lower(c byte) byte {
	return c | ('x' - 'X')
}

// readFloat scans a decimal or a hexadecimal float at the start of s.
//
// It gives the mantissa, the exponent, and the sign. The flag trunc marks a
// digit that the mantissa drops, and the flag isHex marks the hexadecimal
// form. The count i holds the bytes that the scan takes, and ok reports a
// valid number.
func readFloat(s string) (mantissa uint64, exp int, neg, trunc, isHex bool, i int, ok bool) {
	underscores := false
	if i >= len(s) {
		return
	}
	if s[i] == '+' {
		i++
	} else if s[i] == '-' {
		i++
		neg = true
	}

	base := uint64(10)
	maxMantDigits := 19
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' {
		base = 16
		maxMantDigits = 16
		i += 2
		expChar = 'p'
		isHex = true
	}
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
	for i < len(s) {
		c := s[i]
		if c == '_' {
			underscores = true
			i++
			continue
		}
		// A second point ends the number, thus "1.2.3" reads "1.2" only.
		if c == '.' && !sawdot {
			sawdot = true
			dp = nd
			i++
			continue
		}
		if c >= '0' && c <= '9' {
			sawdigits = true
			if c == '0' && nd == 0 {
				dp--
				i++
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				mantissa *= base
				mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' {
				trunc = true
			}
			i++
			continue
		}
		if base == 16 && lower(c) >= 'a' && lower(c) <= 'f' {
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				mantissa *= 16
				mantissa += uint64(lower(c) - 'a' + 10)
				ndMant++
			} else {
				trunc = true
			}
			i++
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		dp = nd
	}
	if base == 16 {
		dp *= 4
		ndMant *= 4
	}

	if i < len(s) && lower(s[i]) == expChar {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '_') {
			if s[i] == '_' {
				underscores = true
				i++
				continue
			}
			// A huge exponent needs no exact value, because the result is an
			// infinity or a zero either way.
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
			i++
		}
		dp += e * esign
	} else if base == 16 {
		return
	}

	if mantissa != 0 {
		exp = dp - ndMant
	}
	if underscores && !underscoreOK(s[:i]) {
		return
	}
	ok = true
	return
}

// underscoreOK reports a text where each underscore stands between 2 digits, or
// between a base prefix and a digit.
func underscoreOK(s string) bool {
	// saw holds the class of the last byte. ^ marks the start, 0 a digit or a
	// base prefix, _ an underscore, and ! any other byte.
	saw := '^'
	i := 0
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	hex := false
	if len(s) >= 2 && s[0] == '0' &&
		(lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') {
		i = 2
		saw = '0'
		hex = lower(s[1]) == 'x'
	}
	for ; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' || hex && lower(s[i]) >= 'a' && lower(s[i]) <= 'f' {
			saw = '0'
			continue
		}
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		if saw == '_' {
			return false
		}
		saw = '!'
	}
	return saw != '_'
}

// decimalFloat gives the value of a decimal float that readFloat accepts. The
// bridge of Dart reads no underscore, thus each one goes first.
func decimalFloat(text string, bitSize int) (float64, error) {
	clean := text
	if index(text, '_') >= 0 {
		buf := make([]byte, 0, len(text))
		for i := 0; i < len(text); i++ {
			if text[i] != '_' {
				buf = append(buf, text[i])
			}
		}
		clean = string(buf)
	}
	v, ok := dart.ParseFloat(clean)
	if !ok {
		return 0, ErrSyntax
	}
	// specialFloat holds the text of an infinity, thus an infinity here comes
	// from a value above the range of the type.
	if bitSize == 32 {
		f := float32(v)
		if dart.Float32Bits(f)&0x7FFFFFFF == 0x7F800000 {
			return float64(f), ErrRange
		}
		return float64(f), nil
	}
	if dart.Float64Bits(v)&0x7FFFFFFFFFFFFFFF == 0x7FF0000000000000 {
		return v, ErrRange
	}
	return v, nil
}

// hexFloat gives the value of a hexadecimal float from its mantissa and its
// binary exponent. The steps hold the way Go rounds, thus the last bit agrees.
func hexFloat(mantissa uint64, exp int, neg, trunc bool, bitSize int) (float64, error) {
	mantBits := uint(52)
	expBits := uint(11)
	bias := -1023
	if bitSize == 32 {
		mantBits = 23
		expBits = 8
		bias = -127
	}
	maxExp := 1<<expBits + bias - 2
	minExp := bias + 1
	exp += int(mantBits)

	// The target holds a leading bit, mantBits other bits, and 2 spare bits that
	// round the value. The low spare bit records every bit that a shift drops.
	for mantissa != 0 && mantissa>>(mantBits+2) == 0 {
		mantissa <<= 1
		exp--
	}
	if trunc {
		mantissa |= 1
	}
	for mantissa>>(1+mantBits+2) != 0 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	for mantissa > 1 && exp < minExp-2 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}

	round := mantissa & 3
	mantissa >>= 2
	round |= mantissa & 1
	exp += 2
	if round == 3 {
		mantissa++
		if mantissa == 1<<(1+mantBits) {
			mantissa >>= 1
			exp++
		}
	}
	if mantissa>>mantBits == 0 {
		exp = bias
	}
	var err error
	if exp > maxExp {
		mantissa = 1 << mantBits
		exp = maxExp + 1
		err = ErrRange
	}

	bits := mantissa & (1<<mantBits - 1)
	bits |= uint64((exp-bias)&(1<<expBits-1)) << mantBits
	if neg {
		bits |= 1 << mantBits << expBits
	}
	if bitSize == 32 {
		return float64(dart.Float32FromBits(uint32(bits))), err
	}
	return dart.Float64FromBits(bits), err
}

// FormatUint gives the text of i in the base.
func FormatUint(i uint64, base int) string {
	if base < 2 || base > 36 {
		panic("strconv: illegal base")
	}
	if i == 0 {
		return "0"
	}
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	out := ""
	v := i
	for v > 0 {
		out = string(digits[int(v%uint64(base))]) + out
		v /= uint64(base)
	}
	return out
}

// CanBackquote reports whether s fits inside back quotes with no change.
func CanBackquote(s string) bool {
	for _, r := range s {
		if r == 0xFEFF || r == utf8.RuneError {
			return false
		}
		if r > 0x7F {
			continue
		}
		if r < ' ' && r != '\t' || r == '`' || r == 0x7F {
			return false
		}
	}
	return true
}

// lowerhex holds the digits of the base 16 that an escape form takes.
const lowerhex = "0123456789abcdef"

// Quote gives the Go text of s inside double quotes. A rune outside the set of
// IsPrint takes an escape form.
func Quote(s string) string {
	return quoteWith(s, '"', false, false)
}

// AppendQuote adds the form of Quote to dst.
func AppendQuote(dst []byte, s string) []byte {
	return appendQuotedWith(dst, s, '"', false, false)
}

// QuoteToASCII gives the Go text of s inside double quotes. Each rune outside
// the printable range of ASCII takes an escape form.
func QuoteToASCII(s string) string {
	return quoteWith(s, '"', true, false)
}

// AppendQuoteToASCII adds the form of QuoteToASCII to dst.
func AppendQuoteToASCII(dst []byte, s string) []byte {
	return appendQuotedWith(dst, s, '"', true, false)
}

// QuoteToGraphic gives the Go text of s inside double quotes. A graphic rune
// of Unicode stays, and each other rune takes an escape form.
func QuoteToGraphic(s string) string {
	return quoteWith(s, '"', false, true)
}

// AppendQuoteToGraphic adds the form of QuoteToGraphic to dst.
func AppendQuoteToGraphic(dst []byte, s string) []byte {
	return appendQuotedWith(dst, s, '"', false, true)
}

// QuoteRune gives the Go text of r inside single quotes. An invalid code point
// takes the value of the replacement rune.
func QuoteRune(r rune) string {
	return quoteRuneWith(r, '\'', false, false)
}

// AppendQuoteRune adds the form of QuoteRune to dst.
func AppendQuoteRune(dst []byte, r rune) []byte {
	return appendQuotedRuneWith(dst, r, '\'', false, false)
}

// QuoteRuneToASCII gives the Go text of r inside single quotes. A rune outside
// the printable range of ASCII takes an escape form.
func QuoteRuneToASCII(r rune) string {
	return quoteRuneWith(r, '\'', true, false)
}

// AppendQuoteRuneToASCII adds the form of QuoteRuneToASCII to dst.
func AppendQuoteRuneToASCII(dst []byte, r rune) []byte {
	return appendQuotedRuneWith(dst, r, '\'', true, false)
}

// QuoteRuneToGraphic gives the Go text of r inside single quotes. A rune
// outside the graphic set of Unicode takes an escape form.
func QuoteRuneToGraphic(r rune) string {
	return quoteRuneWith(r, '\'', false, true)
}

// AppendQuoteRuneToGraphic adds the form of QuoteRuneToGraphic to dst.
func AppendQuoteRuneToGraphic(dst []byte, r rune) []byte {
	return appendQuotedRuneWith(dst, r, '\'', false, true)
}

// IsPrint reports a rune that Go defines as printable: a letter, a number, a
// mark of punctuation, a symbol, and the space of ASCII. The package unicode
// holds the tables, thus this package needs no copy of them.
func IsPrint(r rune) bool {
	return unicode.IsPrint(r)
}

// IsGraphic reports a rune that Unicode defines as graphic. Such a rune belongs
// to the category L, M, N, P, S, or Zs.
func IsGraphic(r rune) bool {
	return unicode.IsGraphic(r)
}

// inGraphicList reports a graphic rune that is not printable. Each such rune is
// a space of Unicode outside ASCII.
func inGraphicList(r rune) bool {
	return unicode.IsGraphic(r) && !unicode.IsPrint(r)
}

func quoteWith(s string, quote byte, asciiOnly, graphicOnly bool) string {
	return string(appendQuotedWith(make([]byte, 0, 3*len(s)/2), s, quote, asciiOnly, graphicOnly))
}

func quoteRuneWith(r rune, quote byte, asciiOnly, graphicOnly bool) string {
	return string(appendQuotedRuneWith(nil, r, quote, asciiOnly, graphicOnly))
}

// appendQuotedWith adds the quoted form of s to buf. A byte outside any valid
// rune of UTF-8 keeps its own value in the \x form, thus Unquote gives the
// bytes of s again.
func appendQuotedWith(buf []byte, s string, quote byte, asciiOnly, graphicOnly bool) []byte {
	buf = append(buf, quote)
	for len(s) > 0 {
		r, width := utf8.DecodeRuneInString(s)
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, '\\', 'x', lowerhex[s[0]>>4], lowerhex[s[0]&0xF])
		} else {
			buf = appendEscapedRune(buf, r, quote, asciiOnly, graphicOnly)
		}
		s = s[width:]
	}
	return append(buf, quote)
}

func appendQuotedRuneWith(buf []byte, r rune, quote byte, asciiOnly, graphicOnly bool) []byte {
	buf = append(buf, quote)
	if !utf8.ValidRune(r) {
		r = utf8.RuneError
	}
	buf = appendEscapedRune(buf, r, quote, asciiOnly, graphicOnly)
	return append(buf, quote)
}

func appendEscapedRune(buf []byte, r rune, quote byte, asciiOnly, graphicOnly bool) []byte {
	if r == rune(quote) || r == '\\' {
		return append(buf, '\\', byte(r))
	}
	if asciiOnly {
		if r < utf8.RuneSelf && IsPrint(r) {
			return append(buf, byte(r))
		}
	} else if IsPrint(r) || graphicOnly && inGraphicList(r) {
		return utf8.AppendRune(buf, r)
	}
	switch r {
	case '\a':
		return append(buf, "\\a"...)
	case '\b':
		return append(buf, "\\b"...)
	case '\f':
		return append(buf, "\\f"...)
	case '\n':
		return append(buf, "\\n"...)
	case '\r':
		return append(buf, "\\r"...)
	case '\t':
		return append(buf, "\\t"...)
	case '\v':
		return append(buf, "\\v"...)
	}
	// The order holds the order of Go. A negative rune reaches the \x form
	// first, thus it keeps the low byte of its value.
	if r < ' ' || r == 0x7F {
		return append(buf, '\\', 'x', lowerhex[byte(r)>>4], lowerhex[byte(r)&0xF])
	}
	if !utf8.ValidRune(r) {
		r = 0xFFFD
	}
	if r < 0x10000 {
		buf = append(buf, '\\', 'u')
		for shift := 12; shift >= 0; shift -= 4 {
			buf = append(buf, lowerhex[int(r>>uint(shift))&0xF])
		}
		return buf
	}
	buf = append(buf, '\\', 'U')
	for shift := 28; shift >= 0; shift -= 4 {
		buf = append(buf, lowerhex[int(r>>uint(shift))&0xF])
	}
	return buf
}

func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}

// UnquoteChar decodes the first character or byte in the escaped string
// or character literal represented by the string s.
// It returns four values:
//
//  1. value, the decoded Unicode code point or byte value;
//  2. multibyte, a boolean indicating whether the decoded character requires a multibyte UTF-8 representation;
//  3. tail, the remainder of the string after the character; and
//  4. an error that will be nil if the character is syntactically valid.
//
// The second argument, quote, specifies the type of literal being parsed
// and therefore which escaped quote character is permitted.
// If set to a single quote, it permits the sequence \' and disallows unescaped '.
// If set to a double quote, it permits \" and disallows unescaped ".
// If set to zero, it does not permit either escape and allows both quote characters to appear unescaped.
func UnquoteChar(s string, quote byte) (value rune, multibyte bool, tail string, err error) {
	// easy cases
	if len(s) == 0 {
		err = ErrSyntax
		return
	}
	switch c := s[0]; {
	case c == quote && (quote == '\'' || quote == '"'):
		err = ErrSyntax
		return
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}

	// hard case: c is backslash
	if len(s) <= 1 {
		err = ErrSyntax
		return
	}
	c := s[1]
	s = s[2:]

	switch c {
	case 'a':
		value = '\a'
	case 'b':
		value = '\b'
	case 'f':
		value = '\f'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case 'v':
		value = '\v'
	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var v rune
		if len(s) < n {
			err = ErrSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = ErrSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c == 'x' {
			// single-byte string, possibly not UTF-8
			value = v
			break
		}
		if !utf8.ValidRune(v) {
			err = ErrSyntax
			return
		}
		value = v
		multibyte = true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(c) - '0'
		if len(s) < 2 {
			err = ErrSyntax
			return
		}
		for j := 0; j < 2; j++ { // one digit already; two more
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				err = ErrSyntax
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = ErrSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		if c != quote {
			err = ErrSyntax
			return
		}
		value = rune(c)
	default:
		err = ErrSyntax
		return
	}
	tail = s
	return
}

// Unquote interprets s as a single-quoted, double-quoted,
// or backquoted Go string literal, returning the string value
// that s quotes.  (If s is single-quoted, it would be a Go
// character literal; Unquote returns the corresponding
// one-character string. For an empty character literal
// Unquote returns the empty string.)
func Unquote(s string) (string, error) {
	out, rem, err := unquote(s, true)
	if len(rem) > 0 {
		return "", ErrSyntax
	}
	return out, err
}

// QuotedPrefix gives the quoted text at the start of s, with its quotes.
func QuotedPrefix(s string) (string, error) {
	out, _, err := unquote(s, false)
	return out, err
}

// unquote parses a quoted string at the start of the input,
// returning the parsed prefix, the remaining suffix, and any parse errors.
// If unescape is true, the parsed prefix is unescaped,
// otherwise the input prefix is provided verbatim.
func unquote(in string, unescape bool) (out, rem string, err error) {
	// Determine the quote form and optimistically find the terminating quote.
	if len(in) < 2 {
		return "", in, ErrSyntax
	}
	quote := in[0]
	end := index(in[1:], quote)
	if end < 0 {
		return "", in, ErrSyntax
	}
	end += 2 // position after terminating quote; may be wrong if escape sequences are present

	switch quote {
	case '`':
		switch {
		case !unescape:
			out = in[:end] // include quotes
		case !contains(in[:end], '\r'):
			out = in[len("`") : end-len("`")] // exclude quotes
		default:
			// Carriage return characters ('\r') inside raw string literals
			// are discarded from the raw string value.
			buf := make([]byte, 0, end-len("`")-len("\r")-len("`"))
			for i := len("`"); i < end-len("`"); i++ {
				if in[i] != '\r' {
					buf = append(buf, in[i])
				}
			}
			out = string(buf)
		}
		// NOTE: Prior implementations did not verify that raw strings consist
		// of valid UTF-8 characters and we continue to not verify it as such.
		// The Go specification does not explicitly require valid UTF-8,
		// but only mention that it is implicitly valid for Go source code
		// (which must be valid UTF-8).
		return out, in[end:], nil
	case '"', '\'':
		// Handle quoted strings without any escape sequences.
		if !contains(in[:end], '\\') && !contains(in[:end], '\n') {
			var valid bool
			switch quote {
			case '"':
				valid = utf8.ValidString(in[len(`"`) : end-len(`"`)])
			case '\'':
				r, n := utf8.DecodeRuneInString(in[len("'") : end-len("'")])
				valid = len("'")+n+len("'") == end && (r != utf8.RuneError || n != 1)
			}
			if valid {
				out = in[:end]
				if unescape {
					out = out[1 : end-1] // exclude quotes
				}
				return out, in[end:], nil
			}
		}

		// Handle quoted strings with escape sequences.
		var buf []byte
		in0 := in
		in = in[1:] // skip starting quote
		if unescape {
			buf = make([]byte, 0, 3*end/2) // try to avoid more allocations
		}
		for len(in) > 0 && in[0] != quote {
			// Process the next character,
			// rejecting any unescaped newline characters which are invalid.
			r, multibyte, rem, err := UnquoteChar(in, quote)
			if in[0] == '\n' || err != nil {
				return "", in0, ErrSyntax
			}
			in = rem

			// Append the character if unescaping the input.
			if unescape {
				if r < utf8.RuneSelf || !multibyte {
					buf = append(buf, byte(r))
				} else {
					buf = utf8.AppendRune(buf, r)
				}
			}

			// Single quoted strings must be a single character.
			if quote == '\'' {
				break
			}
		}

		// Verify that the string ends with a terminating quote.
		if !(len(in) > 0 && in[0] == quote) {
			return "", in0, ErrSyntax
		}
		in = in[1:] // skip terminating quote

		if unescape {
			return string(buf), in, nil
		}
		return in0[:len(in0)-len(in)], in, nil
	default:
		return "", in, ErrSyntax
	}
}

// contains reports whether the string holds the byte.
func contains(s string, c byte) bool {
	return index(s, c) != -1
}

// index gives the position of the byte in the string, and it gives -1 for a
// string that holds no such byte.
func index(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// unhex gives the value of a hexadecimal digit.
func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
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
