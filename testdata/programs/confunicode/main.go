// The program proves unicode, unicode/utf8, and unicode/utf16 against real Go:
// each predicate on a table of runes, In and Is with a range table, To with
// each case, SimpleFold, each function of utf8 on a valid and on an invalid
// input, and Encode, Decode, and the surrogate functions of utf16.
package main

import (
	"fmt"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

var runes = []rune{
	'a', 'Z', '0', '9', ' ', '\t', '\n', '\x00', 0x7f, '!', '-', '_', '$', '+', '<', '(', '"', '\'', '`',
	'é', 'É', 'ß', 'ǅ', 'ǆ', 'Ǆ', 'ſ', 'K', 'İ', 'ı', 'Σ', 'σ', 'ς', 'Ω', 'ω', 'µ', 'ª',
	'日', '本', 'ア', 'ㄱ', '한', 'א', 'ب', 'ก', 'Ⅷ', '½', '٣', '৩', '²', '№', '€', '©', '→', '∑', '😀', '🇩',
	0x00a0, 0x00ad, 0x0300, 0x0301, 0x034f, 0x0378, 0x0600, 0x1680, 0x2000, 0x2028, 0x2029, 0x200b, 0x200d, 0x202f, 0x2060, 0x3000,
	0xd800, 0xdfff, 0xe000, 0xfeff, 0xfffd, 0xfffe, 0xffff, 0x10000, 0x1d400, 0x1d7ce, 0xe0001, 0xf0000, 0x10ffff, 0x110000, -1, 0x85, 0x9f, 0x1c, 0x1f,
}

func predicates() {
	fmt.Printf("%-8s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s\n", "rune", "letter", "digit", "number", "space", "upper", "lower", "title", "punct", "ctrl", "graph", "print", "symbol", "mark")
	for _, r := range runes {
		fmt.Printf("%-8s %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v\n", fmt.Sprintf("%#x", r),
			unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsNumber(r), unicode.IsSpace(r), unicode.IsUpper(r), unicode.IsLower(r), unicode.IsTitle(r),
			unicode.IsPunct(r), unicode.IsControl(r), unicode.IsGraphic(r), unicode.IsPrint(r), unicode.IsSymbol(r), unicode.IsMark(r))
	}
}

func tables() {
	fmt.Printf("%-8s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s %-6s\n", "rune", "Han", "Latin", "Greek", "Cyril", "Lu", "Nd", "Zs", "Sc", "InL/N", "OneOf")
	for _, r := range runes {
		fmt.Printf("%-8s %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v %-6v\n", fmt.Sprintf("%#x", r),
			unicode.Is(unicode.Han, r), unicode.Is(unicode.Latin, r), unicode.Is(unicode.Greek, r), unicode.Is(unicode.Cyrillic, r),
			unicode.Is(unicode.Lu, r), unicode.Is(unicode.Nd, r), unicode.Is(unicode.Zs, r), unicode.Is(unicode.Sc, r),
			unicode.In(r, unicode.Letter, unicode.Number), unicode.IsOneOf([]*unicode.RangeTable{unicode.Punct, unicode.Symbol}, r))
	}
	custom := &unicode.RangeTable{
		R16:         []unicode.Range16{{Lo: 'a', Hi: 'f', Stride: 1}, {Lo: '0', Hi: '8', Stride: 2}, {Lo: 0x100, Hi: 0x104, Stride: 4}},
		R32:         []unicode.Range32{{Lo: 0x10000, Hi: 0x10004, Stride: 2}},
		LatinOffset: 2,
	}
	for _, r := range []rune{'a', 'f', 'g', '0', '1', '2', '8', '9', 0x100, 0x102, 0x104, 0x105, 0x10000, 0x10001, 0x10002, 0x10004, 0x10005} {
		fmt.Printf("custom Is %#x %v\n", r, unicode.Is(custom, r))
	}
	fmt.Println("In none", unicode.In('a'))
	fmt.Println("Is empty table", unicode.Is(&unicode.RangeTable{}, 'a'))
	fmt.Println("Categories has Lu", unicode.Categories["Lu"] == unicode.Lu)
	fmt.Println("Scripts has Han", unicode.Scripts["Han"] == unicode.Han)
	fmt.Println("Properties has Dash", unicode.Properties["Dash"] != nil)
	fmt.Println("Dash", unicode.Is(unicode.Dash, '-'), unicode.Is(unicode.Dash, 'a'))
	fmt.Println("White_Space", unicode.Is(unicode.White_Space, 0x85), unicode.Is(unicode.White_Space, 0x200b))
	fmt.Println("ASCII_Hex_Digit", unicode.Is(unicode.ASCII_Hex_Digit, 'f'), unicode.Is(unicode.ASCII_Hex_Digit, 'g'))
	fmt.Println("FoldCategory L", unicode.FoldCategory["L"] != nil, "Lu", unicode.FoldCategory["Lu"] != nil)
	fmt.Println("FoldScript Common", unicode.FoldScript["Common"] != nil)
	fmt.Println("MaxRune", unicode.MaxRune, "ReplacementChar", unicode.ReplacementChar, "MaxASCII", unicode.MaxASCII, "MaxLatin1", unicode.MaxLatin1)
	fmt.Println("Version", unicode.Version)
	fmt.Println("UpperCase LowerCase TitleCase MaxCase", unicode.UpperCase, unicode.LowerCase, unicode.TitleCase, unicode.MaxCase)
	fmt.Println("UpperLower", unicode.UpperLower)
}

func cases() {
	fmt.Printf("%-8s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n", "rune", "upper", "lower", "title", "ToUpper", "ToLower", "ToTitle", "fold")
	for _, r := range runes {
		fmt.Printf("%-8s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n", fmt.Sprintf("%#x", r),
			fmt.Sprintf("%#x", unicode.To(unicode.UpperCase, r)), fmt.Sprintf("%#x", unicode.To(unicode.LowerCase, r)), fmt.Sprintf("%#x", unicode.To(unicode.TitleCase, r)),
			fmt.Sprintf("%#x", unicode.ToUpper(r)), fmt.Sprintf("%#x", unicode.ToLower(r)), fmt.Sprintf("%#x", unicode.ToTitle(r)), fmt.Sprintf("%#x", unicode.SimpleFold(r)))
	}
	fmt.Println("To bad case", fmt.Sprintf("%#x", unicode.To(5, 'a')), fmt.Sprintf("%#x", unicode.To(-1, 'a')))
	for _, r := range []rune{'k', 'K', 'K', 's', 'S', 'ſ', 'σ', 'Σ', 'ς', 'ǅ', 'ǆ', 'Ǆ', 'µ', 'μ', 'Μ', 'θ', 'ϑ', 'Θ', 'ϴ', 'ß', 'ẞ', 'a', '1', 0x10ffff, 0x110000, -1, 0x1e9e} {
		fold := r
		var orbit []string
		for i := 0; i < 8; i++ {
			orbit = append(orbit, fmt.Sprintf("%#x", fold))
			fold = unicode.SimpleFold(fold)
			if fold == r {
				break
			}
		}
		fmt.Printf("SimpleFold orbit %#x %v\n", r, orbit)
	}
	for _, r := range []rune{'i', 'I', 'ı', 'İ', 'a', 'j'} {
		fmt.Printf("TurkishCase %#x upper=%#x lower=%#x title=%#x\n", r, unicode.TurkishCase.ToUpper(r), unicode.TurkishCase.ToLower(r), unicode.TurkishCase.ToTitle(r))
		fmt.Printf("AzeriCase   %#x upper=%#x lower=%#x title=%#x\n", r, unicode.AzeriCase.ToUpper(r), unicode.AzeriCase.ToLower(r), unicode.AzeriCase.ToTitle(r))
	}
	special := unicode.SpecialCase{
		unicode.CaseRange{Lo: 'a', Hi: 'a', Delta: [unicode.MaxCase]rune{'A' - 'a', 0, 'A' - 'a'}},
		unicode.CaseRange{Lo: 'x', Hi: 'z', Delta: [unicode.MaxCase]rune{0, 0, 0}},
	}
	for _, r := range []rune{'a', 'b', 'x', 'X', 'z'} {
		fmt.Printf("SpecialCase %c upper=%c lower=%c title=%c\n", r, special.ToUpper(r), special.ToLower(r), special.ToTitle(r))
	}
	fmt.Println("CaseRanges count > 300", len(unicode.CaseRanges) > 300)
}

func utf8s() {
	inputs := []string{"", "a", "é", "日", "😀", "a日😀", "\xff", "\xc3", "\xc3\x28", "\xe6\x97", "\xe6\x97\xa5x", "\xed\xa0\x80", "\xf4\x90\x80\x80", "\xc0\x80", "\xe0\x80\x80", "\xf0\x80\x80\x80", "\x80", "\xbf", "\xfe", "abc\xffdef", "\xef\xbf\xbd", "\xf8\x88\x80\x80\x80"}
	fmt.Printf("%-26s %-5s %-5s %-6s %-6s %-14s %-14s\n", "input", "len", "runes", "valid", "full", "first", "last")
	for _, s := range inputs {
		r1, n1 := utf8.DecodeRuneInString(s)
		r2, n2 := utf8.DecodeLastRuneInString(s)
		fmt.Printf("%-26q %-5d %-5d %-6v %-6v %-14s %-14s\n", s, len(s), utf8.RuneCountInString(s), utf8.ValidString(s), utf8.FullRuneInString(s),
			fmt.Sprintf("%#x/%d", r1, n1), fmt.Sprintf("%#x/%d", r2, n2))
	}
	fmt.Println("-- the byte forms --")
	for _, s := range inputs {
		b := []byte(s)
		r1, n1 := utf8.DecodeRune(b)
		r2, n2 := utf8.DecodeLastRune(b)
		fmt.Printf("%-26q %-5d %-6v %-6v %-14s %-14s\n", b, utf8.RuneCount(b), utf8.Valid(b), utf8.FullRune(b), fmt.Sprintf("%#x/%d", r1, n1), fmt.Sprintf("%#x/%d", r2, n2))
	}
	fmt.Println("-- the encode forms --")
	for _, r := range []rune{0, 'a', 0x7f, 0x80, 0x7ff, 0x800, 0xffff, 0x10000, 0x10ffff, 0xd800, 0xdfff, 0xfffd, -1, 0x110000, 0x1f600} {
		buf := make([]byte, utf8.UTFMax)
		n := utf8.EncodeRune(buf, r)
		app := utf8.AppendRune([]byte("x"), r)
		fmt.Printf("rune %-9s len=%-3d valid=%-6v enc=%-20q append=%q\n", fmt.Sprintf("%#x", r), utf8.RuneLen(r), utf8.ValidRune(r), string(buf[:n]), string(app))
	}
	fmt.Println("-- RuneStart --")
	for _, b := range []byte{0x00, 0x41, 0x7f, 0x80, 0xbf, 0xc0, 0xc2, 0xdf, 0xe0, 0xef, 0xf0, 0xf4, 0xf5, 0xff} {
		fmt.Printf("RuneStart %#x %v\n", b, utf8.RuneStart(b))
	}
	fmt.Println("constants", utf8.RuneError, utf8.RuneSelf, utf8.MaxRune, utf8.UTFMax)
	fmt.Printf("RuneError text %q %#x\n", string(utf8.RuneError), utf8.RuneError)
	s := "a日\xffb"
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		fmt.Printf("walk at %d rune=%#x size=%d\n", i, r, size)
		i += size
	}
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		fmt.Printf("walk back to %d rune=%#x size=%d\n", i, r, size)
		i -= size
	}
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-24s %v\n", label, recover())
	}()
	f()
}

func utf16s() {
	texts := []string{"", "a", "aé日", "😀", "a😀b", "\xff", "�", "\U0010ffff"}
	for _, s := range texts {
		enc := utf16.Encode([]rune(s))
		dec := utf16.Decode(enc)
		fmt.Printf("Encode %-14q %-32v Decode %q\n", s, fmt.Sprintf("%#x", enc), string(dec))
	}
	fmt.Println("Encode invalid rune", fmt.Sprintf("%#x", utf16.Encode([]rune{-1, 0x110000, 0xd800, 0xdfff, 0xe000})))
	fmt.Println("Decode lone surrogates", fmt.Sprintf("%#x", utf16.Decode([]uint16{0xd800, 0x41, 0xdc00, 0xd83d, 0xde00, 0xd83d})))
	fmt.Println("Decode reversed pair", fmt.Sprintf("%#x", utf16.Decode([]uint16{0xde00, 0xd83d})))
	fmt.Println("Decode nil", utf16.Decode(nil) == nil, len(utf16.Decode(nil)))
	fmt.Println("Encode nil", utf16.Encode(nil) == nil, len(utf16.Encode(nil)))
	for _, r := range []rune{0x41, 0xffff, 0x10000, 0x1f600, 0x10ffff, 0x110000, 0xd800, -1, 0xe000} {
		r1, r2 := utf16.EncodeRune(r)
		fmt.Printf("EncodeRune %-9s %#x %#x IsSurrogate=%v RuneLen=%d AppendRune=%#x\n", fmt.Sprintf("%#x", r), r1, r2, utf16.IsSurrogate(r), utf16.RuneLen(r), utf16.AppendRune([]uint16{0x20}, r))
	}
	for _, p := range [][2]rune{{0xd83d, 0xde00}, {0xd800, 0xdc00}, {0xdbff, 0xdfff}, {0xde00, 0xd83d}, {0x41, 0x42}, {0xd800, 0x41}, {0x41, 0xdc00}, {0xdbff, 0xe000}} {
		fmt.Printf("DecodeRune %#x %#x -> %#x\n", p[0], p[1], utf16.DecodeRune(p[0], p[1]))
	}
	for _, r := range []rune{0xd7ff, 0xd800, 0xdbff, 0xdc00, 0xdfff, 0xe000, 0x41, -1, 0x10000} {
		fmt.Printf("IsSurrogate %#x %v\n", r, utf16.IsSurrogate(r))
	}
}

// encodeRoom proves the panic of EncodeRune and of AppendRune when the slice
// holds too few bytes. Go reads the last byte of the room that it needs first,
// thus the index of the panic names that byte.
func encodeRoom() {
	for _, c := range []struct {
		size int
		r    rune
	}{{1, 'a'}, {0, 'a'}, {1, 'é'}, {1, '日'}, {2, '日'}, {1, '\U0001F600'}, {3, '\U0001F600'}} {
		func() {
			defer func() { fmt.Printf("%-22s %v\n", fmt.Sprintf("encode %d %q", c.size, c.r), recover()) }()
			n := utf8.EncodeRune(make([]byte, c.size), c.r)
			fmt.Printf("%-22s %v\n", fmt.Sprintf("encode %d %q", c.size, c.r), n)
		}()
	}
}

func main() {
	encodeRoom()
	fmt.Println("== the predicates ==")
	predicates()
	fmt.Println("== the tables ==")
	tables()
	fmt.Println("== the cases ==")
	cases()
	fmt.Println("== utf8 ==")
	utf8s()
	fmt.Println("== utf16 ==")
	utf16s()
}
