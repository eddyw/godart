// The program proves text/tabwriter and text/scanner against real Go. The
// tabwriter part writes 1 table under each flag and under 2 padding
// characters, with a cell that holds a tab and a cell with a rune of 2 bytes.
// The scanner part reads a Go like source and prints each token kind with its
// position, and the count of the errors.
package main

import (
	"fmt"
	"strings"
	"text/scanner"
	"text/tabwriter"
	"unicode"
)

const table = "name\tqty\tprice\tnote\n" +
	"apple\t3\t1.25\tred\n" +
	"kiwi\t12\t0.5\t\n" +
	"pomegranate\t1\t12.75\tlarge <b>fruit</b>\n" +
	"\t\t\t\n" +
	"fig\t100\t3\téé\n" +
	"\xff\tin\xff\tx\ty\n" +
	"ünïcödé\t7\t2\tz\n"

func show(label string, minwidth, tabwidth, padding int, padchar byte, flags uint, text string) {
	fmt.Printf("-- %s: minwidth=%d tabwidth=%d padding=%d padchar=%q flags=%d --\n", label, minwidth, tabwidth, padding, padchar, flags)
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, minwidth, tabwidth, padding, padchar, flags)
	n, err := fmt.Fprint(w, text)
	if err != nil {
		fmt.Println("write error", n, err)
	}
	if err := w.Flush(); err != nil {
		fmt.Println("flush error", err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(sb.String(), "\n"), "\n") {
		fmt.Printf("|%s|\n", line)
	}
}

func tabs() {
	show("default", 0, 8, 1, ' ', 0, table)
	show("minwidth 5 padding 2", 5, 8, 2, ' ', 0, table)
	show("dots", 0, 8, 1, '.', 0, table)
	show("dots AlignRight", 0, 8, 1, '.', tabwriter.AlignRight, table)
	show("tab padchar", 0, 4, 1, '\t', 0, table)
	show("tab padchar TabIndent", 0, 4, 1, '\t', tabwriter.TabIndent, "\tindented\tcell\n\t\tdeeper\tcell\nplain\tx\ty\n")
	show("AlignRight", 0, 8, 1, ' ', tabwriter.AlignRight, table)
	show("AlignRight dots padding 3", 0, 8, 3, '.', tabwriter.AlignRight, table)
	show("FilterHTML", 0, 8, 1, '.', tabwriter.FilterHTML, table)
	show("FilterHTML entity", 0, 8, 1, '.', tabwriter.FilterHTML, "a&amp;b\tc\n&lt;\td\n<span class=\"x\">e</span>\tf\n")
	show("StripEscape", 0, 8, 1, '.', tabwriter.StripEscape, table)
	show("no StripEscape", 0, 8, 1, '.', 0, "\xffa\tb\xff\tc\nd\te\n")
	show("DiscardEmptyColumns", 0, 8, 1, '.', tabwriter.DiscardEmptyColumns, "a\t\tc\nd\t\tf\n\t\t\n")
	show("keep empty columns", 0, 8, 1, '.', 0, "a\t\tc\nd\t\tf\n\t\t\n")
	show("Debug", 0, 8, 1, ' ', tabwriter.Debug, table)
	show("Debug AlignRight dots", 0, 8, 1, '.', tabwriter.Debug|tabwriter.AlignRight, table)
	show("Debug DiscardEmptyColumns", 0, 8, 1, ' ', tabwriter.Debug|tabwriter.DiscardEmptyColumns, "a\t\tc\nd\t\tf\n")
	show("each flag", 4, 4, 2, '-', tabwriter.FilterHTML|tabwriter.StripEscape|tabwriter.AlignRight|tabwriter.DiscardEmptyColumns|tabwriter.Debug, table)
	show("no trailing tab", 0, 8, 1, '.', 0, "a\tb\nlonger\tc\nx\n")
	show("blank lines", 0, 8, 1, '.', 0, "a\tb\n\nc\td\n\n\n")
	show("form feed", 0, 8, 1, '.', 0, "a\tb\nccc\td\fe\tf\nggggg\th\n")
	show("vertical tab", 0, 8, 1, '.', 0, "a\vb\vc\nlonger\vd\ve\n")
	show("html tag with tab", 0, 8, 1, '.', tabwriter.FilterHTML, "<a\thref=\"x\">\tb\nc\td\n")
	show("wide runes", 0, 8, 1, '.', 0, "日本語\tx\nab\ty\n😀\tz\n")
	show("zero padding", 0, 8, 0, '.', 0, "a\tb\nccc\td\n")
	show("large minwidth", 12, 8, 1, '.', 0, "a\tb\nccc\td\n")
	show("tabwidth 2 with tab padchar", 0, 2, 1, '\t', 0, "a\tb\nccccc\td\n")
	show("cells with spaces", 0, 8, 1, '.', 0, "a b c\td\ne\tf g\n")
	show("one column", 0, 8, 1, '.', 0, "a\nbb\nccc\n")
	show("only tabs", 0, 8, 1, '.', 0, "\t\t\n\t\t\n")
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 8, 1, '.', 0)
	w.Write([]byte("split\tacross"))
	w.Write([]byte(" writes\tx\n"))
	w.Write([]byte("a\tb\tc\n"))
	w.Flush()
	fmt.Printf("%-24s %q\n", "split writes", sb.String())
	sb.Reset()
	w.Init(&sb, 0, 8, 1, '-', tabwriter.AlignRight)
	fmt.Fprint(w, "a\tb\nccc\td\n")
	w.Flush()
	fmt.Printf("%-24s %q\n", "Init", sb.String())
	sb.Reset()
	w = tabwriter.NewWriter(&sb, 0, 8, 1, '.', 0)
	fmt.Fprint(w, "first\tx\n")
	w.Flush()
	fmt.Fprint(w, "second longer\ty\n")
	w.Flush()
	fmt.Printf("%-24s %q\n", "2 flushes", sb.String())
	sb.Reset()
	w = tabwriter.NewWriter(&sb, 0, 8, 1, '.', 0)
	fmt.Fprint(w, "no flush\tx\n")
	fmt.Printf("%-24s %q\n", "before Flush", sb.String())
	catch("bad padchar", func() { tabwriter.NewWriter(&sb, 0, 8, -1, '.', 0) })
	fmt.Printf("%-24s %d\n", "Escape", tabwriter.Escape)
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-24s %v\n", label, recover())
	}()
	f()
}

const source = `// A comment line.
package main

/* a block
   comment */
import "fmt"

func main() {
	x := 42
	y := 0x1F + 0o17 + 0b101 + 1_000
	f := 3.14 + .5 + 1e3 + 0x1p-2 + 6.
	c := 'a' + '\n' + '\x41' + 'é'
	s := "text\t\"quoted\"" + ` + "`raw\nstring`" + `
	fmt.Println(x, y, f, c, s, ünïcödé, _under, a1)
	if x >= 10 && y != 0 || !true { return }
	z := x<<2 | y>>1 &^ 3
	arr[1:2]; m["k"] = &z; p.q.r
	@ $ 'unterminated
	"unterminated string
	0x 08 1e+ 'ab'
}
`

func scan() {
	var s scanner.Scanner
	s.Init(strings.NewReader(source))
	s.Filename = "source.go"
	errs := 0
	s.Error = func(s *scanner.Scanner, msg string) {
		errs++
		fmt.Printf("error at %s: %s\n", s.Pos(), msg)
	}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("error count", errs, s.ErrorCount)
	fmt.Println("end position", s.Pos())
	fmt.Println("-- with comments --")
	s.Init(strings.NewReader("a // line\nb /* block */ c\n/* unterminated"))
	s.Mode = scanner.GoTokens &^ scanner.SkipComments
	s.Error = func(s *scanner.Scanner, msg string) { fmt.Printf("error at %s: %s\n", s.Pos(), msg) }
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("error count", s.ErrorCount)
	fmt.Println("-- no comment scan --")
	s.Init(strings.NewReader("a // not a comment\nb"))
	s.Mode = scanner.ScanIdents
	s.Error = func(s *scanner.Scanner, msg string) { fmt.Printf("error at %s: %s\n", s.Pos(), msg) }
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("-- idents and ints only --")
	s.Init(strings.NewReader("abc 123 4.5 'c' \"s\" `r` // c\n+ x"))
	s.Mode = scanner.ScanIdents | scanner.ScanInts
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("-- whitespace kept --")
	s.Init(strings.NewReader("a b\tc\nd"))
	s.Whitespace = 1<<'\t' | 1<<'\n'
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("-- custom ident --")
	s.Init(strings.NewReader("a-b c.d $e 1x"))
	s.IsIdentRune = func(ch rune, i int) bool {
		return ch == '$' || ch == '-' || unicode.IsLetter(ch) || (unicode.IsDigit(ch) && i > 0)
	}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("-- Next and Peek --")
	s.Init(strings.NewReader("ab\ncd"))
	for {
		p := s.Peek()
		ch := s.Next()
		if ch == scanner.EOF {
			fmt.Println("EOF", p == scanner.EOF, s.Pos())
			break
		}
		fmt.Printf("Next %q Peek %q pos %s\n", ch, p, s.Pos())
	}
	fmt.Println("-- mixed Next and Scan --")
	s.Init(strings.NewReader("x = 5"))
	s.Scan()
	fmt.Printf("%q at %s then %q\n", s.TokenText(), s.Position, s.Next())
	s.Scan()
	fmt.Printf("%q at %s\n", s.TokenText(), s.Position)
	s.Scan()
	fmt.Printf("%q at %s\n", s.TokenText(), s.Position)
	fmt.Println("-- invalid input --")
	s.Init(strings.NewReader("a\xffb \x00 c"))
	s.Error = func(s *scanner.Scanner, msg string) { fmt.Printf("error at %s: %s\n", s.Pos(), msg) }
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%-16s %-12s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}
	fmt.Println("error count", s.ErrorCount)
	fmt.Println("-- positions --")
	var pos scanner.Position
	fmt.Println("zero IsValid", pos.IsValid(), fmt.Sprintf("%q", pos.String()))
	pos = scanner.Position{Filename: "f.go", Line: 3, Column: 7}
	fmt.Println("with file", pos.IsValid(), pos)
	pos = scanner.Position{Line: 3, Column: 7}
	fmt.Println("no file", pos)
	pos = scanner.Position{Filename: "f.go"}
	fmt.Println("only file", pos)
	pos = scanner.Position{Line: 1}
	fmt.Println("no column", pos)
	fmt.Println("-- token names --")
	for _, t := range []rune{scanner.EOF, scanner.Ident, scanner.Int, scanner.Float, scanner.Char, scanner.String, scanner.RawString, scanner.Comment, '+', 'a', 0x7f, 'é', '\n'} {
		fmt.Printf("%d %s\n", t, scanner.TokenString(t))
	}
	fmt.Println("GoTokens", scanner.GoTokens, "GoWhitespace", scanner.GoWhitespace)
}

func main() {
	fmt.Println("== tabwriter ==")
	tabs()
	fmt.Println("== scanner ==")
	scan()
}
