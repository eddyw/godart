// Command strconvquote exercises the quote family, the graphic tests, and the
// complex conversions of the package strconv.
package main

import (
	"fmt"
	"strconv"
)

func main() {
	quoted()
	runes()
	graphic()
	complexes()
}

// texts holds a plain text, an escape, and a rune outside ASCII. It also holds
// a space of Unicode that is graphic but not printable, a separator that is
// neither, an emoji, and a byte outside any rune of UTF-8.
var texts = []string{
	"plain",
	"a\tb\nc",
	"café",
	"gap here",
	"line break",
	"grin\U0001F600",
	"bad\xffbyte",
	"quote\"back\\slash",
	"\a\b\f\v\x00\x7f",
}

func quoted() {
	for _, s := range texts {
		fmt.Printf("Q   %s\n", strconv.Quote(s))
		fmt.Printf("QA  %s\n", strconv.QuoteToASCII(s))
		fmt.Printf("QG  %s\n", strconv.QuoteToGraphic(s))
		fmt.Printf("AQA %s\n", string(strconv.AppendQuoteToASCII([]byte("dst="), s)))
		fmt.Printf("AQG %s\n", string(strconv.AppendQuoteToGraphic([]byte("dst="), s)))
	}
}

// marks holds a rune of each class that the escape rules separate.
var marks = []rune{
	'a', ' ', '\t', '\n', '\'', '"', '\\', 0x7f, 0x00,
	0x00e9, 0x00a0, 0x2028, 0x3000, 0xfffd, 0x1f600, 0xffff, 0x10ffff,
	-1, 0x110000, 0xd800,
}

func runes() {
	for _, r := range marks {
		fmt.Printf("QR  %d %s\n", r, strconv.QuoteRune(r))
		fmt.Printf("QRA %d %s\n", r, strconv.QuoteRuneToASCII(r))
		fmt.Printf("QRG %d %s\n", r, strconv.QuoteRuneToGraphic(r))
		fmt.Printf("AQR %d %s\n", r, string(strconv.AppendQuoteRune([]byte("dst="), r)))
		fmt.Printf("ARA %d %s\n", r, string(strconv.AppendQuoteRuneToASCII([]byte("dst="), r)))
		fmt.Printf("ARG %d %s\n", r, string(strconv.AppendQuoteRuneToGraphic([]byte("dst="), r)))
	}
}

func graphic() {
	for _, r := range marks {
		fmt.Printf("PG %d %v %v\n", r, strconv.IsPrint(r), strconv.IsGraphic(r))
	}
	sweep()
}

// sweep counts the printable runes and the graphic runes of the whole range,
// and it folds the quoted form of a sample into a hash. A table that differs
// from the table of Go moves a count or the hash.
func sweep() {
	printCount, graphicCount := 0, 0
	var hash uint64 = 14695981039346656037
	for r := rune(0); r <= 0x10ffff; r++ {
		if strconv.IsPrint(r) {
			printCount++
		}
		if strconv.IsGraphic(r) {
			graphicCount++
		}
		if r%313 != 0 {
			continue
		}
		for _, s := range []string{
			strconv.QuoteRune(r),
			strconv.QuoteRuneToASCII(r),
			strconv.QuoteRuneToGraphic(r),
		} {
			for i := 0; i < len(s); i++ {
				hash ^= uint64(s[i])
				hash *= 1099511628211
			}
		}
	}
	fmt.Printf("sweep print=%d graphic=%d hash=%d\n", printCount, graphicCount, hash)
}

// numbers holds a valid form, a form with parentheses, a lone imaginary part, a
// special value, an underscore, a hexadecimal part, and several bad forms.
var numbers = []string{
	"1+2i", "(3-4i)", "5i", "3", "-2.5e3-1.5e-2i",
	"NaN", "NaNi", "+Infi", "1+NaNi", "Inf-Infi",
	"1e400+1i", "1-1e400i", "0x1p-2+0x1p-1i", "1_0.5+2i",
	"(1+2i", "1++2i", "1+2", "i", "+i", "-i", "1e", "", " 1+2i", "1+2i ",
}

func complexes() {
	for _, s := range numbers {
		c, err := strconv.ParseComplex(s, 128)
		fmt.Printf("PC128 %s -> %v %v\n", strconv.Quote(s), c, err)
		c, err = strconv.ParseComplex(s, 64)
		fmt.Printf("PC64  %s -> %v %v\n", strconv.Quote(s), c, err)
	}

	values := []complex128{
		complex(1.5, -2.5),
		complex(1, 2),
		complex(0, 0),
		complex(-3, 0),
		complex(1.0/3.0, 2.0/3.0),
	}
	for _, c := range values {
		fmt.Printf("FC f2  %s\n", strconv.FormatComplex(c, 'f', 2, 128))
		fmt.Printf("FC e3  %s\n", strconv.FormatComplex(c, 'e', 3, 128))
		fmt.Printf("FC g   %s\n", strconv.FormatComplex(c, 'g', -1, 128))
		fmt.Printf("FC f4  %s\n", strconv.FormatComplex(c, 'f', 4, 64))
		fmt.Printf("FC g64 %s\n", strconv.FormatComplex(c, 'g', -1, 64))
		fmt.Printf("FC e64 %s\n", strconv.FormatComplex(c, 'e', 2, 64))
	}
}
