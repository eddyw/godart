// The witness of the report of a verb that a kind rejects, of the width of
// such a report, of the back quote of %#q, of %q of a byte slice, of %x of a
// value with a String method, of a Formatter inside a compound value, and of
// the flag 0 beside a precision.
//
// Go drops the flag 0 of a whole number when a precision stands beside it, and
// it pads with spaces there.
package main

import (
	"errors"
	"fmt"
)

type celsius float64

type spec struct{}

func (spec) Format(f fmt.State, verb rune) {
	fmt.Fprintf(f, "<%c plus=%v sharp=%v>", verb, f.Flag('+'), f.Flag('#'))
}

type point struct {
	X int
	Y string
}

func main() {
	fmt.Println("-- the verb that the kind rejects --")
	fmt.Printf("%e|\n", 42)
	fmt.Printf("%f|\n", 42)
	fmt.Printf("%.1f|\n", []int{1, 2, 3})
	fmt.Printf("%e|\n", point{1, "a"})
	fmt.Printf("%s|\n", int8(-128))
	fmt.Printf("%d|\n", float32(0.1))
	fmt.Printf("%c|\n", celsius(1))
	fmt.Printf("%q|\n", 42)
	fmt.Printf("%t|\n", 42)
	fmt.Printf("%U|\n", "a")
	fmt.Printf("%3.1f|\n", []int{1, 2})
	fmt.Printf("%08.2f|\n", []int{1, 2})

	fmt.Println("-- the width inside the report --")
	fmt.Printf("%5d|\n", []string{"a"})
	fmt.Printf("%05d|\n", []string{"a"})
	fmt.Printf("%-5d|\n", []string{"a"})

	fmt.Println("-- the back quote --")
	fmt.Printf("%#q|\n", "abc")
	fmt.Printf("%+#q|\n", "abc")
	fmt.Printf("%#q|\n", "a`b")

	fmt.Println("-- the quote of a byte slice --")
	fmt.Printf("%q|\n", []byte("￿"))
	fmt.Printf("%+q|\n", []byte("é"))
	fmt.Printf("%q|\n", []byte("ab\x00"))

	fmt.Println("-- the hexadecimal of a String method --")
	fmt.Printf("%x|\n", errors.New("plain"))
	fmt.Printf("%X|\n", errors.New("plain"))
	fmt.Printf("%.2s|\n", errors.New("plain"))

	fmt.Println("-- the flag 0 beside a precision --")
	fmt.Printf("[%08.2d]\n", 1)
	fmt.Printf("[%08.4d]\n", -1)
	fmt.Printf("[%08.2x]\n", 255)
	fmt.Printf("[%08.2o]\n", 8)
	fmt.Printf("[%08.2b]\n", 3)
	fmt.Printf("[%08.2v]\n", 1)
	fmt.Printf("[%08d]\n", 1)
	fmt.Printf("[%08.2f]\n", 1.5)
	fmt.Printf("[%-08.2d]\n", 1)
	fmt.Printf("[%08.2s]\n", "abcd")
	fmt.Printf("[%.2d]\n", 1)
	fmt.Printf("[%.0d]\n", 0)
	fmt.Printf("[%5.0d]\n", 0)
	fmt.Printf("[%05.0d]\n", 0)
	fmt.Printf("[%#08.2x]\n", 255)
	fmt.Printf("[%08.3d]\n", uint8(7))

	fmt.Println("-- the Formatter inside a list --")
	fmt.Printf("%q|\n", []any{spec{}})
	fmt.Printf("%v|\n", []any{spec{}})
	fmt.Printf("%+v|\n", map[string]any{"k": spec{}})
	fmt.Printf("%#v|\n", struct{ S any }{spec{}})
}
