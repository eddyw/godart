// The program proves the package strconv against real Go: the integer forms
// with each base and each bit size, the error text of each failure, the float
// forms with each letter and each precision, the bool form, the quote forms
// with each escape, and the fields of a NumError.
package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

func integers() {
	row("Itoa", strconv.Itoa(-42))
	row("Itoa 0", strconv.Itoa(0))
	row("Itoa max", strconv.Itoa(math.MaxInt64))
	row("Itoa min", strconv.Itoa(math.MinInt64))
	for _, in := range []string{"123", "-123", "+123", "0", "", "abc", "12a", " 12",
		"9223372036854775807", "9223372036854775808", "-9223372036854775808",
		"-9223372036854775809", "0x10", "1_000", "010", "٣", "-", "+", "--1"} {
		n, err := strconv.Atoi(in)
		fmt.Printf("Atoi %-24q %d %v\n", in, n, err)
	}
	fmt.Println("-- ParseInt with each base --")
	for _, base := range []int{0, 2, 8, 10, 16, 36} {
		for _, in := range []string{"101", "-0x1f", "0b101", "0o17", "017", "0X1F", "zz", "1_0", "0x_1f", "1__0", "_1", "0b", "0x", "-0"} {
			n, err := strconv.ParseInt(in, base, 64)
			fmt.Printf("ParseInt %-8q base %-2d %d %v\n", in, base, n, err)
		}
	}
	fmt.Println("-- ParseInt with each bit size --")
	for _, bits := range []int{0, 8, 16, 32, 64} {
		for _, in := range []string{"127", "128", "-128", "-129", "32767", "32768", "2147483647", "2147483648", "-2147483649", "9223372036854775807", "9223372036854775808", "-9223372036854775808", "-9223372036854775809", "99999999999999999999999"} {
			n, err := strconv.ParseInt(in, 10, bits)
			fmt.Printf("ParseInt %-25q bits %-2d %d %v\n", in, bits, n, err)
		}
	}
	_, err := strconv.ParseInt("1", 1, 64)
	row("ParseInt base 1", err)
	_, err = strconv.ParseInt("1", 37, 64)
	row("ParseInt base 37", err)
	_, err = strconv.ParseInt("1", 10, 65)
	row("ParseInt bits 65", err)
	_, err = strconv.ParseInt("1", 10, -1)
	row("ParseInt bits -1", err)
	fmt.Println("-- ParseUint --")
	for _, in := range []string{"0", "255", "256", "65536", "4294967296", "18446744073709551615", "18446744073709551616", "-1", "+1", "", "0xff", "0b11111111", "0o377", "0377", "99999999999999999999999"} {
		for _, bits := range []int{8, 64} {
			n, err := strconv.ParseUint(in, 0, bits)
			fmt.Printf("ParseUint %-24q bits %-2d %d %v\n", in, bits, n, err)
		}
	}
	n16, err := strconv.ParseUint("65535", 10, 16)
	row("ParseUint 16", fmt.Sprint(n16, " ", err))
	n32, err := strconv.ParseUint("4294967295", 10, 32)
	row("ParseUint 32", fmt.Sprint(n32, " ", err))
	n32, err = strconv.ParseUint("ff", 16, 32)
	row("ParseUint base 16", fmt.Sprint(n32, " ", err))
	fmt.Println("-- FormatInt and FormatUint --")
	for _, v := range []int64{0, 1, -1, 255, -255, 1 << 40, math.MaxInt64, math.MinInt64} {
		for _, base := range []int{2, 8, 10, 16, 36} {
			fmt.Printf("FormatInt %-21d base %-2d %s\n", v, base, strconv.FormatInt(v, base))
		}
	}
	for _, v := range []uint64{0, 255, 1 << 63, math.MaxUint64} {
		for _, base := range []int{2, 10, 16, 36} {
			fmt.Printf("FormatUint %-20d base %-2d %s\n", v, base, strconv.FormatUint(v, base))
		}
	}
	row("AppendInt", string(strconv.AppendInt([]byte("n="), -99, 10)))
	row("AppendInt base 2", string(strconv.AppendInt(nil, 5, 2)))
	row("AppendUint", string(strconv.AppendUint([]byte("u="), 255, 16)))
	row("AppendBool", string(strconv.AppendBool([]byte("b="), true)))
	row("AppendFloat", string(strconv.AppendFloat([]byte("f="), 1.5, 'g', -1, 64)))
	row("AppendQuote", string(strconv.AppendQuote([]byte("q="), "a\"b\n")))
	row("AppendQuoteRune", string(strconv.AppendQuoteRune([]byte("r="), '☺')))
	row("AppendQuoteToASCII", string(strconv.AppendQuoteToASCII(nil, "日本")))
	row("AppendQuoteRuneToASCII", string(strconv.AppendQuoteRuneToASCII(nil, '日')))
	row("AppendQuoteToGraphic", string(strconv.AppendQuoteToGraphic(nil, "a b")))
	row("AppendQuoteRuneToGraphic", string(strconv.AppendQuoteRuneToGraphic(nil, ' ')))
}

func floats() {
	inputs := []string{"1e", ".5", "5.", "-.5e1", "1.5e+3", "1.5E3", " 1", "1 ", "0x1.8p1", "0x1p", "0x.8p1",
		"0x1.8", "1e23", "9007199254740993", "infinity", "-Infinity", "+inf", "INF", "1_0e1", "1e1_0", "0X1P-2",
		"1e-", "+", "-", "", ".", "e5", "1e+", "0b1", "0o7", "1e309", "-1e309", "2.4703282292062327e-324",
		"2.4703282292062328e-324", "1e-324", "0.1", "0.3", "1.7976931348623157e308", "1.7976931348623159e308",
		"123456789012345678901234567890", "0.000000000000000000000000000001", "1e0", "00001", "1.e5", "١", "1e-7", "1E+05",
		"1e-400", "0x1p-1074", "0x1p-1075", "0x1p1024", "0x1.fffffffffffffp1023", "nan", "NAN", "1.5e", "1e5.5", "1,5", "0x", "0xg", "1e+308", "0.000001", "1e-6"}
	for _, in := range inputs {
		v, err := strconv.ParseFloat(in, 64)
		fmt.Printf("ParseFloat 64 %-32q %-24v %v\n", in, v, err)
	}
	for _, in := range []string{"3.4e39", "-3.4e39", "1e-46", "0.1", "16777217", "3.4028235e38", "3.40282357e38", "1e-45", "7e-46", "0x1p-149", "0x1p-150", "1.00000006", "1.00000005"} {
		v, err := strconv.ParseFloat(in, 32)
		fmt.Printf("ParseFloat 32 %-32q %-24v %v bits=%#x\n", in, v, err, math.Float32bits(float32(v)))
	}
	fmt.Println("-- FormatFloat with each letter --")
	values := []float64{0, math.Copysign(0, -1), 1, -1.5, 0.5, 2.5, 1.005, 123456.789, 1e21, 1e20, 1e-7, 1e-4, 1e-5,
		math.Pi, 1e100, 5e-324, math.MaxFloat64, math.Inf(1), math.Inf(-1), math.NaN(), 100, 0.1, 1234567, 12345678, 0.000123456, 2.2250738585072014e-308}
	for _, v := range values {
		for _, letter := range []byte{'e', 'E', 'f', 'g', 'G', 'x', 'X', 'b'} {
			fmt.Printf("FormatFloat %-23v %c %s\n", v, letter, strconv.FormatFloat(v, letter, -1, 64))
		}
	}
	fmt.Println("-- FormatFloat with each precision --")
	for _, v := range []float64{0, 0.5, 1.5, 2.5, 1.005, 123456.789, 1e21, 1e-7, math.Pi, 99.999, 0.000123456, 1234567, 0.05, 0.15, 0.25, 0.35, 2.675, 1e15, 123456789012345678} {
		for _, prec := range []int{0, 1, 2, 3, 6, 10, 17, 25} {
			fmt.Printf("FormatFloat %-12v prec %-2d e=%s f=%s g=%s\n", v, prec,
				strconv.FormatFloat(v, 'e', prec, 64), strconv.FormatFloat(v, 'f', prec, 64), strconv.FormatFloat(v, 'g', prec, 64))
		}
	}
	for _, v := range []float64{math.Pi, 1e21, 5e-324, 1.5, 0.1, 1.0} {
		for _, prec := range []int{0, 1, 3, 13} {
			fmt.Printf("FormatFloat %-23v x prec %-2d %s X %s\n", v, prec, strconv.FormatFloat(v, 'x', prec, 64), strconv.FormatFloat(v, 'X', prec, 64))
		}
	}
	for _, v := range []float64{math.Inf(1), math.NaN(), 1e6, 1e-7} {
		fmt.Printf("FormatFloat %-23v G prec 3 %s E prec 3 %s\n", v, strconv.FormatFloat(v, 'G', 3, 64), strconv.FormatFloat(v, 'E', 3, 64))
	}
	fmt.Println("-- FormatFloat with 32 bits --")
	for _, v := range []float64{0.1, 1.0 / 3.0, 16777217, 3.4028234663852886e38, 1e-45, 1.5, math.Pi, 1e10, 123456.789} {
		for _, letter := range []byte{'g', 'e', 'f', 'x', 'b'} {
			fmt.Printf("FormatFloat32 %-23v %c %s\n", v, letter, strconv.FormatFloat(float64(float32(v)), letter, -1, 32))
		}
	}
	row("FormatFloat 32 prec 3", strconv.FormatFloat(float64(float32(math.Pi)), 'g', 3, 32))
	row("FormatFloat 32 f prec 10", strconv.FormatFloat(float64(float32(0.1)), 'f', 10, 32))
	row("FormatFloat bad letter", strconv.FormatFloat(1.5, 'z', -1, 64))
	row("FormatComplex", strconv.FormatComplex(complex(1.5, -2), 'g', -1, 128))
	row("FormatComplex e", strconv.FormatComplex(complex(0, 1), 'e', 2, 64))
	row("FormatComplex inf", strconv.FormatComplex(complex(math.Inf(1), math.NaN()), 'g', -1, 128))
	c, err := strconv.ParseComplex("1+2i", 128)
	row("ParseComplex", fmt.Sprintf("%v %v", c, err))
	c, err = strconv.ParseComplex("(3-4i)", 128)
	row("ParseComplex parens", fmt.Sprintf("%v %v", c, err))
	c, err = strconv.ParseComplex("2i", 128)
	row("ParseComplex imaginary", fmt.Sprintf("%v %v", c, err))
	c, err = strconv.ParseComplex("1+", 128)
	row("ParseComplex bad", fmt.Sprintf("%v %v", c, err))
	c, err = strconv.ParseComplex("1e400+1i", 128)
	row("ParseComplex range", fmt.Sprintf("%v %v", c, err))
}

func bools() {
	for _, in := range []string{"1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False", "yes", "", "tRuE", "2", " true"} {
		b, err := strconv.ParseBool(in)
		fmt.Printf("ParseBool %-8q %-5v %v\n", in, b, err)
	}
	row("FormatBool true", strconv.FormatBool(true))
	row("FormatBool false", strconv.FormatBool(false))
}

func quotes() {
	inputs := []string{"", "plain", "a\"b", "a'b", "a\\b", "\a\b\f\n\r\t\v", "\x00\x01\x1f\x7f", "é", "\u00a0", "\u00ad",
		"\U0001F600", "日本語", "a\xffb", "\xed\xa0\x80", "`back`", "\u200b", "\ufeff", "\u3000", "�", "\U000E0001", "͸", "\x1b[0m", "\u0085", "\u2028", "tab\tin", "\xc3\x28"}
	for _, in := range inputs {
		fmt.Printf("Quote          %s\n", strconv.Quote(in))
		fmt.Printf("QuoteToASCII   %s\n", strconv.QuoteToASCII(in))
		fmt.Printf("QuoteToGraphic %s\n", strconv.QuoteToGraphic(in))
		fmt.Printf("CanBackquote   %v\n", strconv.CanBackquote(in))
	}
	for _, r := range []rune{'a', '\'', '"', '\\', '\n', 0, 0x7f, 'é', '☺', '日', 0x1F600, 0xa0, 0xfffd, 0xd800, -1, 0x110000, 0x378, 0xe0001, 0x0085, 0x200b, 0x1b, '\t', 0x3000} {
		fmt.Printf("QuoteRune %-8d %s %s %s\n", r, strconv.QuoteRune(r), strconv.QuoteRuneToASCII(r), strconv.QuoteRuneToGraphic(r))
	}
	fmt.Println("-- Unquote --")
	unquotes := []string{`"plain"`, `"a\"b"`, `"\a\b\f\n\r\t\v\\"`, `"\x41é\U0001F600"`, `"\101"`, `"\377"`, `"\400"`,
		`"日本"`, `'a'`, `'\n'`, `'ab'`, `''`, `'é'`, "`raw\\n`", "`multi\nline`", "`with\rcr`", `"unterminated`, `plain`, `""`,
		`"\z"`, `"\x4"`, `"\u12"`, `"\U0011FFFF"`, `"\xff"`, `"\ud800"`, "\"new\nline\"", "\"tab\tinside\"", `"'"`, `'"'`, `'\''`, `'\"'`, `"\'"`,
		`"a" "b"`, `"\U0001F600\x00"`, "'\xff'", "\"\xff\"", `'\xff'`, `'\377'`, `'é'`, "'日'", `"é"`, "``", "`a`b`"}
	for _, in := range unquotes {
		s, err := strconv.Unquote(in)
		fmt.Printf("Unquote %-26q %q %v\n", in, s, err)
	}
	for _, in := range []string{`"abc" rest`, `'x' rest`, "`raw` rest", `"bad`, `nothing`, `"\n"more`, `''`, `"\z"x`} {
		s, err := strconv.QuotedPrefix(in)
		fmt.Printf("QuotedPrefix %-16q %q %v\n", in, s, err)
	}
	for _, in := range []string{`a rest`, `\n rest`, `\x41 rest`, "é", `\U0001F600`, `\'`, `\"`, `"`, `'`, `\z`, ``, `日本`, `\377`, `\xff`, `éx`, "\xff", `\`, `\x`, "\n"} {
		for _, q := range []byte{'"', '\'', 0} {
			v, mb, tail, err := strconv.UnquoteChar(in, q)
			fmt.Printf("UnquoteChar %-14q %-3q %q %v %q %v\n", in, q, v, mb, tail, err)
		}
	}
	fmt.Println("-- IsPrint and IsGraphic --")
	for _, r := range []rune{' ', 'a', '~', 0x7f, 0x80, 0xa0, 0xad, 0xe9, 0x300, 0x378, 0x2028, 0x200b, 0x3000, 0xfeff, 0xfffd, 0xfffe, 0x1f600, 0xe0001, 0xf0000, 0x10ffff, -1, 0x110000, 0x0085, 0x1680, 0x2000, 0x202f, 0x205f, 0xd800, 0x10fffe, 0x1f, 0x0009, 0xe000, 0xffff, 0x2ffff} {
		fmt.Printf("IsPrint %-8d %-5v IsGraphic %v\n", r, strconv.IsPrint(r), strconv.IsGraphic(r))
	}
}

func numErrors() {
	_, err := strconv.Atoi("12x")
	var ne *strconv.NumError
	if errors.As(err, &ne) {
		fmt.Printf("Atoi NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	row("Is ErrSyntax", errors.Is(err, strconv.ErrSyntax))
	row("Is ErrRange", errors.Is(err, strconv.ErrRange))
	row("Unwrap", errors.Unwrap(err) == strconv.ErrSyntax)
	_, err = strconv.ParseInt("999999999999999999999", 10, 64)
	if errors.As(err, &ne) {
		fmt.Printf("ParseInt NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	row("Is ErrRange", errors.Is(err, strconv.ErrRange))
	_, err = strconv.ParseUint("-1", 10, 64)
	if errors.As(err, &ne) {
		fmt.Printf("ParseUint NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	_, err = strconv.ParseFloat("1e400", 64)
	if errors.As(err, &ne) {
		fmt.Printf("ParseFloat NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	_, err = strconv.ParseBool("maybe")
	if errors.As(err, &ne) {
		fmt.Printf("ParseBool NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	_, err = strconv.ParseInt("1", 99, 64)
	if errors.As(err, &ne) {
		fmt.Printf("ParseInt base NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	_, err = strconv.ParseComplex("bad", 128)
	if errors.As(err, &ne) {
		fmt.Printf("ParseComplex NumError Func=%q Num=%q Err=%v\n", ne.Func, ne.Num, ne.Err)
	}
	_, err = strconv.Unquote("bad")
	row("Unquote error", err)
	row("Unquote Is ErrSyntax", errors.Is(err, strconv.ErrSyntax))
	made := &strconv.NumError{Func: "Custom", Num: "x", Err: strconv.ErrRange}
	row("made NumError", made)
	row("made Unwrap", made.Unwrap())
	row("ErrSyntax text", strconv.ErrSyntax)
	row("ErrRange text", strconv.ErrRange)
	row("IntSize", strconv.IntSize)
	_, err = strconv.Atoi("\xff")
	row("Atoi invalid utf8", err)
}

func main() {
	fmt.Println("== the integers ==")
	integers()
	fmt.Println("== the floats ==")
	floats()
	fmt.Println("== the bools ==")
	bools()
	fmt.Println("== the quotes ==")
	quotes()
	fmt.Println("== NumError ==")
	numErrors()
}
