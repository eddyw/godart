//go:build dart

package fmt

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"syscall/dart"
	"unicode/utf8"
)

// A ScanState holds the state of 1 scan. A Scanner of the caller reads it.
type ScanState interface {
	// ReadRune gives the next rune of the input.
	ReadRune() (r rune, size int, err error)
	// UnreadRune gives the last rune back to the input.
	UnreadRune() error
	// SkipSpace drops each space of the input.
	SkipSpace()
	// Token gives the next run of runes that f accepts. A nil f accepts each
	// rune that is not a space.
	Token(skipSpace bool, f func(rune) bool) (token []byte, err error)
	// Width gives the width of the verb, and the flag is false for a verb with
	// no width.
	Width() (wid int, ok bool)
	// Read is present for the interface io.Reader, and it always reports an
	// error.
	Read(buf []byte) (n int, err error)
}

// A Scanner reads its own form from a scan. Fscan and each function of its
// family call the method for an argument that holds it.
type Scanner interface {
	Scan(state ScanState, verb rune) error
}

// The errors of a scan. Each text matches the text of Go.
var (
	errEOF     = io.EOF
	errNewline = errors.New("unexpected newline")
	errWantNL  = errors.New("expected newline")
	// errNoNewline names a newline of the format that the input holds not, and
	// errNoInputNL names a newline of the input that the format holds not.
	errNoNewline  = errors.New("newline in format does not match input")
	errNoInputNL  = errors.New("newline in input does not match format")
	errNoMatch    = errors.New("input does not match format")
	errNoSpace    = errors.New("expected space in input to match format")
	errTooMany    = errors.New("too many operands")
	errInteger    = errors.New("expected integer")
	errBool       = errors.New("syntax error scanning boolean")
	errNumber     = errors.New("syntax error scanning number")
	errCharacter  = errors.New("expected character")
	errBadVerbNil = errors.New("bad verb")
	errComplex    = errors.New("syntax error scanning complex number")
)

// eof marks the end of the input inside the scanner.
const eof = -1

// ss holds the state of 1 scan.
type ss struct {
	rs    io.RuneScanner
	buf   []byte
	off   int
	atEOF bool

	// count holds the bytes that the scan took.
	count int

	// nlIsSpace marks a scan where a newline counts as a space. Scan and Fscan
	// set it, and Scanln and Scanf do not.
	nlIsSpace bool
	// nlIsEnd marks a scan where a newline ends the input. Scanln sets it.
	nlIsEnd bool

	// maxWid holds the width of the verb, and widPresent marks a verb that
	// names one.
	maxWid     int
	widPresent bool
	// taken counts the runes of the current verb.
	taken int
}

// newScanner gives a scanner of a reader.
//
// A reader that holds no ReadRune takes the wrapper below, which reads 1 byte
// at a time. A read of a block would take the bytes of the next call of the
// caller, thus a second Fscan over 1 reader would find no input.
func newScanner(r io.Reader, nlIsSpace, nlIsEnd bool) *ss {
	if rs, ok := r.(io.RuneScanner); ok {
		return &ss{rs: rs, nlIsSpace: nlIsSpace, nlIsEnd: nlIsEnd}
	}
	return &ss{rs: &readRune{reader: r, peekRune: -1}, nlIsSpace: nlIsSpace, nlIsEnd: nlIsEnd}
}

// readRune reads 1 rune of UTF-8 from a reader that holds no ReadRune. It
// takes 1 byte at a time, exactly as Go does, thus it holds no byte that the
// caller of the next scan needs.
type readRune struct {
	reader   io.Reader
	buf      [utf8.UTFMax]byte
	pending  int
	pendBuf  [utf8.UTFMax]byte
	peekRune rune
}

// readByte gives the next byte of the input.
func (r *readRune) readByte() (byte, error) {
	if r.pending > 0 {
		b := r.pendBuf[0]
		copy(r.pendBuf[0:], r.pendBuf[1:])
		r.pending--
		return b, nil
	}
	n, err := io.ReadFull(r.reader, r.pendBuf[:1])
	if n != 1 {
		return 0, err
	}
	return r.pendBuf[0], err
}

// ReadRune gives the next rune of the input, with its size in bytes.
func (r *readRune) ReadRune() (rr rune, size int, err error) {
	if r.peekRune >= 0 {
		rr = r.peekRune
		r.peekRune = ^r.peekRune
		size = utf8.RuneLen(rr)
		return
	}
	r.buf[0], err = r.readByte()
	if err != nil {
		return
	}
	if r.buf[0] < utf8.RuneSelf {
		rr = rune(r.buf[0])
		size = 1
		r.peekRune = ^rr
		return
	}
	var n int
	for n = 1; !utf8.FullRune(r.buf[:n]); n++ {
		r.buf[n], err = r.readByte()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
	}
	rr, size = utf8.DecodeRune(r.buf[:n])
	if size < n {
		copy(r.pendBuf[r.pending:], r.buf[size:n])
		r.pending += n - size
	}
	r.peekRune = ^rr
	return
}

// UnreadRune gives the last rune back. The reader holds 1 rune, exactly as Go
// holds it.
func (r *readRune) UnreadRune() error {
	if r.peekRune >= 0 {
		return errors.New("fmt: scanning called UnreadRune with no rune available")
	}
	r.peekRune = ^r.peekRune
	return nil
}

// fill reads 1 rune into the buffer. It reports the end of the input.
func (s *ss) fill() bool {
	if s.atEOF {
		return false
	}
	r, size, err := s.rs.ReadRune()
	if err != nil {
		s.atEOF = true
		return false
	}
	var chunk [utf8.UTFMax]byte
	n := utf8.EncodeRune(chunk[:], r)
	if r == utf8.RuneError && size == 1 {
		// A byte that holds no rune keeps its own value.
		n = 1
		chunk[0] = 0xFF
	}
	s.buf = append(s.buf, chunk[:n]...)
	return true
}

// ReadRune gives the next rune of the input.
func (s *ss) ReadRune() (rune, int, error) {
	if s.widPresent && s.taken >= s.maxWid {
		return 0, 0, errEOF
	}
	for {
		if s.off < len(s.buf) {
			r, size := utf8.DecodeRune(s.buf[s.off:])
			// A rune that the buffer holds in part reads again after a fill.
			if r == utf8.RuneError && size == 1 && !s.atEOF &&
				len(s.buf)-s.off < utf8.UTFMax {
				if s.fill() {
					continue
				}
			}
			s.off += size
			s.count += size
			s.taken++
			return r, size, nil
		}
		if !s.fill() {
			return 0, 0, errEOF
		}
	}
}

// UnreadRune gives the last rune back to the input.
func (s *ss) UnreadRune() error {
	if s.off <= 0 {
		return errors.New("fmt: UnreadRune with no read")
	}
	_, size := utf8.DecodeLastRune(s.buf[:s.off])
	s.off -= size
	s.count -= size
	s.taken--
	return nil
}

// peek gives the next rune and leaves it in the input.
func (s *ss) peek() rune {
	r, _, err := s.ReadRune()
	if err != nil {
		return eof
	}
	s.UnreadRune()
	return r
}

// Read reports an error. Go holds the same rule, because a scan reads runes.
func (s *ss) Read(buf []byte) (int, error) {
	return 0, errors.New("ScanState's Read should not be called")
}

// Width gives the width of the verb.
func (s *ss) Width() (int, bool) { return s.maxWid, s.widPresent }

// isSpace reports a rune that separates 2 tokens. Go holds the same set.
func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0:
		return true
	}
	return r > 0x1000 && (r == 0x1680 || (0x2000 <= r && r <= 0x200a) ||
		r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000)
}

// SkipSpace drops each space of the input. It reports a newline through a
// panic, exactly as Go does, and the recover of the caller reads it.
func (s *ss) SkipSpace() {
	for {
		r, _, err := s.ReadRune()
		if err != nil {
			return
		}
		if r == '\r' && s.peek() == '\n' {
			continue
		}
		if r == '\n' {
			if s.nlIsSpace {
				continue
			}
			// A scan that ends at a newline reports the newline itself, and
			// each other scan reports the input against the format.
			if s.nlIsEnd {
				panic(scanFail{errNewline})
			}
			panic(scanFail{errNoInputNL})
		}
		if !isSpace(r) {
			s.UnreadRune()
			return
		}
	}
}

// Token gives the next run of runes that f accepts.
func (s *ss) Token(skipSpace bool, f func(rune) bool) ([]byte, error) {
	if skipSpace {
		s.SkipSpace()
	}
	if f == nil {
		f = notSpace
	}
	var out []byte
	for {
		r, size, err := s.ReadRune()
		if err != nil {
			break
		}
		if !f(r) {
			s.UnreadRune()
			break
		}
		_ = size
		out = utf8.AppendRune(out, r)
	}
	return out, nil
}

func notSpace(r rune) bool { return !isSpace(r) }

// scanFail carries the cause of a failed scan through a panic. Go holds the
// same shape, thus each read reports its cause at 1 point.
type scanFail struct{ err error }

// token gives the next token of the input.
func (s *ss) token(skipSpace bool, f func(rune) bool) []byte {
	out, _ := s.Token(skipSpace, f)
	return out
}

// scanOne reads 1 value of a verb into the argument.
func (s *ss) scanOne(verb rune, arg any) {
	s.taken = 0
	if v, ok := arg.(Scanner); ok {
		if err := v.Scan(s, verb); err != nil {
			panic(scanFail{err})
		}
		return
	}
	switch v := arg.(type) {
	case *bool:
		*v = s.scanBool(verb)
	case *string:
		*v = string(s.convertString(verb))
	case *[]byte:
		*v = s.convertString(verb)
	case *int:
		*v = int(s.scanInt(verb, 64))
	case *int8:
		*v = int8(s.scanInt(verb, 8))
	case *int16:
		*v = int16(s.scanInt(verb, 16))
	// A rune is an int32, thus the 2 types hold 1 case.
	case *int32:
		*v = int32(s.scanInt(verb, 32))
	case *int64:
		*v = s.scanInt(verb, 64)
	case *uint:
		*v = uint(s.scanUint(verb, 64))
	case *uint8:
		*v = uint8(s.scanUint(verb, 8))
	case *uint16:
		*v = uint16(s.scanUint(verb, 16))
	case *uint32:
		*v = uint32(s.scanUint(verb, 32))
	case *uint64:
		*v = s.scanUint(verb, 64)
	case *uintptr:
		*v = uintptr(s.scanUint(verb, 64))
	case *float32:
		*v = float32(s.scanFloat(verb, 32))
	case *float64:
		*v = s.scanFloat(verb, 64)
	case *complex64:
		*v = complex64(s.scanComplex(verb, 64))
	case *complex128:
		*v = s.scanComplex(verb, 128)
	default:
		if !isPointer(arg) {
			panic(scanFail{errors.New("type not a pointer: " + typeText(arg))})
		}
		panic(scanFail{errors.New("can't scan type: " + typeText(arg))})
	}
}

// typeText names the type of an argument. The Dart side holds the descriptor.
func typeText(arg any) string { return Sprintf("%T", arg) }

// isPointer reports whether the argument holds a pointer. Go writes another
// text for a target that is no pointer.
func isPointer(arg any) bool { return dart.Kind(arg) == "ptr" }

// scanBool reads a boolean.
func (s *ss) scanBool(verb rune) bool {
	if verb != 'v' && verb != 't' {
		panic(scanFail{badVerb(verb, "bool")})
	}
	// Go reads 1 rune and it takes each letter that follows it, thus a text
	// that starts with no digit and with no t or f gives false and no error.
	s.SkipSpace()
	if s.peek() == eof {
		panic(scanFail{errEOF})
	}
	switch s.scanRune() {
	case '0':
		return false
	case '1':
		return true
	case 't', 'T':
		if s.acceptSet("rR") && (!s.acceptSet("uU") || !s.acceptSet("eE")) {
			panic(scanFail{errBool})
		}
		return true
	case 'f', 'F':
		if s.acceptSet("aA") &&
			(!s.acceptSet("lL") || !s.acceptSet("sS") || !s.acceptSet("eE")) {
			panic(scanFail{errBool})
		}
		return false
	}
	return false
}

// digitsOf gives the base of a verb and the set of its digits.
func digitsOf(verb rune) (int, string) {
	switch verb {
	case 'b':
		return 2, "01"
	case 'o', 'O':
		return 8, "01234567"
	case 'x', 'X':
		return 16, "0123456789abcdefABCDEF"
	case 'd', 'v', 'c', 'q', 'U':
		return 10, "0123456789"
	}
	return 0, ""
}

// scanNumber reads the text of a number of the base of the verb.
func (s *ss) scanNumber(verb rune, signed bool) string {
	base, digits := digitsOf(verb)
	if base == 0 {
		panic(scanFail{badVerb(verb, "number")})
	}
	s.SkipSpace()
	var b strings.Builder
	if signed {
		if r := s.peek(); r == '+' || r == '-' {
			s.ReadRune()
			b.WriteRune(r)
		}
	}
	// The verb v takes the prefix of the base, exactly as a Go literal does.
	if verb == 'v' {
		if s.peek() == '0' {
			s.ReadRune()
			b.WriteByte('0')
			switch r := s.peek(); r {
			case 'x', 'X', 'b', 'B', 'o', 'O':
				s.ReadRune()
				b.WriteRune(r)
				_, digits = digitsOf(lower(r))
			default:
				digits = "01234567"
			}
		}
	}
	for {
		r := s.peek()
		// A Go literal accepts an underscore between 2 digits, and the verb v
		// reads a Go literal. strconv holds the rule of the position.
		if r == '_' && verb == 'v' {
			s.ReadRune()
			b.WriteRune(r)
			continue
		}
		if r == eof || !strings.ContainsRune(digits, r) {
			break
		}
		s.ReadRune()
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" || out == "+" || out == "-" {
		if s.peek() == eof {
			panic(scanFail{errEOF})
		}
		panic(scanFail{errInteger})
	}
	return out
}

func lower(r rune) rune {
	if 'A' <= r && r <= 'Z' {
		return r + 'a' - 'A'
	}
	return r
}

// scanInt reads a signed number of the width of bits.
func (s *ss) scanInt(verb rune, bits int) int64 {
	if verb == 'c' {
		return int64(s.scanRune())
	}
	okVerb(verb, "bdoUxXv", "integer")
	base, _ := digitsOf(verb)
	text := s.scanNumber(verb, true)
	if verb == 'v' {
		base = 0
	}
	out, err := strconv.ParseInt(text, base, bits)
	if err != nil {
		// Go writes the value that a wider parse gave, thus a value outside
		// the width names the overflow and not the syntax.
		if wide, second := strconv.ParseInt(text, base, 64); second == nil {
			_ = wide
			panic(scanFail{errors.New("integer overflow on token " + text)})
		}
		panic(scanFail{errNumber})
	}
	return out
}

// scanUint reads a number with no sign of the width of bits.
func (s *ss) scanUint(verb rune, bits int) uint64 {
	if verb == 'c' {
		return uint64(s.scanRune())
	}
	okVerb(verb, "bdoUxXv", "integer")
	base, _ := digitsOf(verb)
	text := s.scanNumber(verb, false)
	if verb == 'v' {
		base = 0
	}
	out, err := strconv.ParseUint(text, base, bits)
	if err != nil {
		if wide, second := strconv.ParseUint(text, base, 64); second == nil {
			_ = wide
			panic(scanFail{errors.New("unsigned integer overflow on token " + text)})
		}
		panic(scanFail{errNumber})
	}
	return out
}

// scanRune reads 1 rune of the input.
func (s *ss) scanRune() rune {
	r, _, err := s.ReadRune()
	if err != nil {
		panic(scanFail{errEOF})
	}
	return r
}

// scanFloat reads a float of the width of bits.
func (s *ss) scanFloat(verb rune, bits int) float64 {
	switch verb {
	case 'v', 'b', 'e', 'E', 'f', 'F', 'g', 'G':
	default:
		panic(scanFail{badVerb(verb, "float")})
	}
	s.SkipSpace()
	if s.peek() == eof {
		panic(scanFail{errEOF})
	}
	text := s.floatToken()
	out, err := strconv.ParseFloat(text, bits)
	if err != nil {
		panic(scanFail{errNumber})
	}
	return out
}

// acceptSet reads the next rune when the set holds it. It reads nothing and
// gives false otherwise.
func (s *ss) acceptSet(set string) bool {
	r := s.peek()
	if r == eof || !strings.ContainsRune(set, r) {
		return false
	}
	s.ReadRune()
	return true
}

// scanComplex reads a complex value of the width of bits. Go writes such a
// value as 2 float tokens with a sign between them and the letter i behind
// them, and a pair of parentheses can hold the whole form.
func (s *ss) scanComplex(verb rune, bits int) complex128 {
	switch verb {
	case 'v', 'b', 'e', 'E', 'f', 'F', 'g', 'G':
	default:
		panic(scanFail{badVerb(verb, "complex")})
	}
	s.SkipSpace()
	if s.peek() == eof {
		panic(scanFail{errEOF})
	}
	parens := s.acceptSet("(")
	realText := s.floatToken()
	sign := s.peek()
	if sign != '+' && sign != '-' {
		panic(scanFail{errComplex})
	}
	s.ReadRune()
	imagText := string(sign) + s.floatToken()
	if !s.acceptSet("i") {
		panic(scanFail{errComplex})
	}
	if parens && !s.acceptSet(")") {
		panic(scanFail{errComplex})
	}
	realPart, err := strconv.ParseFloat(realText, bits/2)
	if err != nil {
		panic(scanFail{errNumber})
	}
	imagPart, err := strconv.ParseFloat(imagText, bits/2)
	if err != nil {
		panic(scanFail{errNumber})
	}
	return complex(realPart, imagPart)
}

// accept reads the next rune when the set holds it, and it writes that rune
// into b. It reads nothing and gives false otherwise.
func (s *ss) accept(b *strings.Builder, set string) bool {
	r := s.peek()
	if r == eof || !strings.ContainsRune(set, r) {
		return false
	}
	s.ReadRune()
	b.WriteRune(r)
	return true
}

// floatToken reads the text of 1 float literal. The token ends at the first
// rune that no float literal holds, thus a format such as "%fC" leaves the C
// for the literal text of the format. A set of accepted runes cannot give that
// rule, because a hexadecimal float holds each letter from a to f.
func (s *ss) floatToken() string {
	var b strings.Builder
	if s.accept(&b, "nN") && s.accept(&b, "aA") && s.accept(&b, "nN") {
		return b.String()
	}
	s.accept(&b, "+-")
	if s.accept(&b, "iI") && s.accept(&b, "nN") && s.accept(&b, "fF") {
		return b.String()
	}
	digits := "0123456789_"
	exp := "eE"
	if s.accept(&b, "0") && s.accept(&b, "xX") {
		digits = "0123456789aAbBcCdDeEfF_"
		exp = "pP"
	}
	for s.accept(&b, digits) {
	}
	if s.accept(&b, ".") {
		for s.accept(&b, digits) {
		}
	}
	if s.accept(&b, exp) {
		s.accept(&b, "+-")
		for s.accept(&b, "0123456789_") {
		}
	}
	return b.String()
}

// convertString reads a text of the verb.
func (s *ss) convertString(verb rune) []byte {
	switch verb {
	case 'v', 's', 'x', 'X', 'q':
	default:
		panic(scanFail{badVerb(verb, "string")})
	}
	if verb == 'q' {
		return s.quotedString()
	}
	if verb == 'x' || verb == 'X' {
		return s.hexString()
	}
	out := s.token(true, notSpace)
	if len(out) == 0 && s.peek() == eof {
		panic(scanFail{errEOF})
	}
	return out
}

// quotedString reads a text between 2 quotation marks or between 2 back
// quotation marks.
func (s *ss) quotedString() []byte {
	s.SkipSpace()
	switch quote := s.scanRune(); quote {
	case '`':
		var b strings.Builder
		for {
			r := s.scanRune()
			if r == '`' {
				break
			}
			b.WriteRune(r)
		}
		return []byte(b.String())
	case '"':
		var b strings.Builder
		b.WriteByte('"')
		for {
			r := s.scanRune()
			b.WriteRune(r)
			if r == '\\' {
				b.WriteRune(s.scanRune())
				continue
			}
			if r == '"' {
				break
			}
		}
		out, err := strconv.Unquote(b.String())
		if err != nil {
			panic(scanFail{errors.New("invalid syntax for quoted string")})
		}
		return []byte(out)
	}
	panic(scanFail{errors.New("expected quoted string")})
}

// hexString reads a text of pairs of hexadecimal digits.
func (s *ss) hexString() []byte {
	s.SkipSpace()
	var out []byte
	for {
		high, ok := s.hexDigit()
		if !ok {
			break
		}
		low, ok := s.hexDigit()
		if !ok {
			panic(scanFail{errors.New("odd count of hexadecimal digits")})
		}
		out = append(out, byte(high<<4|low))
	}
	if len(out) == 0 {
		panic(scanFail{errEOF})
	}
	return out
}

func (s *ss) hexDigit() (int, bool) {
	r := s.peek()
	switch {
	case '0' <= r && r <= '9':
		s.ReadRune()
		return int(r - '0'), true
	case 'a' <= r && r <= 'f':
		s.ReadRune()
		return int(r-'a') + 10, true
	case 'A' <= r && r <= 'F':
		s.ReadRune()
		return int(r-'A') + 10, true
	}
	return 0, false
}

// okVerb stops the scan when the kind of the target rejects the verb. Go
// writes the kind of the target, thus an integer names the word integer.
func okVerb(verb rune, okVerbs, kind string) {
	if strings.ContainsRune(okVerbs, verb) {
		return
	}
	panic(scanFail{badVerb(verb, kind)})
}

func badVerb(verb rune, kind string) error {
	return errors.New("bad verb '%" + string(verb) + "' for " + kind)
}

// doScan reads each argument as 1 token, and it separates the tokens with
// spaces.
func (s *ss) doScan(a []any) (int, error) {
	count := 0
	var cause error
	func() {
		defer func() { cause = recoverScan(recover()) }()
		for _, arg := range a {
			s.scanOne('v', arg)
			count++
		}
		// Scanln needs a newline or the end of the input after the last value.
		if s.nlIsEnd {
			for {
				r, _, err := s.ReadRune()
				if err != nil {
					break
				}
				if r == '\n' {
					break
				}
				if !isSpace(r) {
					s.UnreadRune()
					panic(scanFail{errWantNL})
				}
			}
		}
	}()
	return count, cause
}

// doScanf reads each verb of the format.
func (s *ss) doScanf(format string, a []any) (int, error) {
	count := 0
	var cause error
	func() {
		defer func() { cause = recoverScan(recover()) }()
		next := 0
		for i := 0; i < len(format); {
			c, size := utf8.DecodeRuneInString(format[i:])
			if c != '%' {
				i += size
				if isSpace(c) {
					s.matchSpace(c)
					continue
				}
				s.matchRune(c)
				continue
			}
			i++
			if i >= len(format) {
				panic(scanFail{errors.New("missing verb: % at end of format string")})
			}
			if format[i] == '%' {
				i++
				s.matchRune('%')
				continue
			}
			s.widPresent = false
			s.maxWid = 0
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				s.widPresent = true
				s.maxWid = s.maxWid*10 + int(format[i]-'0')
				i++
			}
			verb, size := utf8.DecodeRuneInString(format[i:])
			i += size
			if next >= len(a) {
				panic(scanFail{errors.New("too few operands for format '%" +
					string(verb) + "'")})
			}
			s.scanOne(verb, a[next])
			next++
			count++
		}
		s.widPresent = false
		if next < len(a) {
			panic(scanFail{errTooMany})
		}
	}()
	return count, cause
}

// matchSpace reads the spaces of the input that a space of the format names. A
// newline of the format must match a newline of the input.
func (s *ss) matchSpace(c rune) {
	if c == '\n' {
		for {
			r := s.peek()
			// Go reads each space of the input, and it takes the end of the
			// input for the newline of the format.
			if r == eof {
				return
			}
			if r == '\n' {
				s.ReadRune()
				return
			}
			if !isSpace(r) {
				panic(scanFail{errNoNewline})
			}
			s.ReadRune()
		}
	}
	for {
		r := s.peek()
		if r == eof || !isSpace(r) {
			return
		}
		if r == '\n' && !s.nlIsSpace {
			return
		}
		s.ReadRune()
	}
}

// matchRune reads 1 rune of the input that the format names.
func (s *ss) matchRune(c rune) {
	r, _, err := s.ReadRune()
	if err != nil {
		panic(scanFail{errEOF})
	}
	if r != c {
		s.UnreadRune()
		if r == '\n' {
			panic(scanFail{errNoInputNL})
		}
		panic(scanFail{errNoMatch})
	}
}

// recoverScan maps the value of a recover to an error. A panic of another kind
// runs again.
func recoverScan(v any) error {
	if v == nil {
		return nil
	}
	if fail, ok := v.(scanFail); ok {
		return fail.err
	}
	panic(v)
}

// Sscan reads each argument of str as 1 token. A newline counts as a space.
func Sscan(str string, a ...any) (int, error) {
	return Fscan(strings.NewReader(str), a...)
}

// Sscanln reads each argument of str as 1 token, and it stops at a newline.
func Sscanln(str string, a ...any) (int, error) {
	return Fscanln(strings.NewReader(str), a...)
}

// Sscanf reads each verb of the format from str.
func Sscanf(str string, format string, a ...any) (int, error) {
	return Fscanf(strings.NewReader(str), format, a...)
}

// Fscan reads each argument of r as 1 token. A newline counts as a space.
func Fscan(r io.Reader, a ...any) (int, error) {
	return newScanner(r, true, false).doScan(a)
}

// Fscanln reads each argument of r as 1 token, and it stops at a newline.
func Fscanln(r io.Reader, a ...any) (int, error) {
	return newScanner(r, false, true).doScan(a)
}

// Fscanf reads each verb of the format from r.
func Fscanf(r io.Reader, format string, a ...any) (int, error) {
	return newScanner(r, false, false).doScanf(format, a)
}

// stdin reads standard input.
//
// Go reads os.Stdin here. An import of os would put the whole file access and
// the clock of the time package into each program that reads fmt, and the web
// target holds neither. Thus the reader calls the runtime itself.
type stdin struct{}

func (stdin) Read(p []byte) (int, error) {
	n, cause := dart.StdinRead(p)
	if cause != "" {
		return n, errors.New(cause)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

// Scan reads each argument of standard input as 1 token.
func Scan(a ...any) (int, error) { return Fscan(stdin{}, a...) }

// Scanln reads each argument of standard input as 1 token, and it stops at a
// newline.
func Scanln(a ...any) (int, error) { return Fscanln(stdin{}, a...) }

// Scanf reads each verb of the format from standard input.
func Scanf(format string, a ...any) (int, error) {
	return Fscanf(stdin{}, format, a...)
}
