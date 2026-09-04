// The program proves text/scanner. The Scan method reads the whitespace set
// with `1 << uint(ch)`, and ch holds -1 at the end of the input. Go gives 0 for
// a shift count above the width, thus the read needs the unsigned form of the
// count. Read decision 328.
package main

import (
	"fmt"
	"strings"
	"text/scanner"
)

func main() {
	var s scanner.Scanner
	s.Init(strings.NewReader("if x >= 42 { name := \"go\"; f(1.5) } // note\n"))
	s.Filename = "input"
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		fmt.Printf("%s: %s %q\n", s.Position, scanner.TokenString(tok), s.TokenText())
	}

	var c scanner.Scanner
	c.Init(strings.NewReader("a\tb\nc"))
	c.Whitespace = 0
	c.Mode = scanner.ScanIdents
	for tok := c.Scan(); tok != scanner.EOF; tok = c.Scan() {
		fmt.Printf("raw %s %q\n", scanner.TokenString(tok), c.TokenText())
	}
	fmt.Println("errors", c.ErrorCount)
}
