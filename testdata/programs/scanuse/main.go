package main

import (
	"errors"
	"fmt"
	"strings"
)

// tag reads its own form from a scan.
type tag struct{ text string }

func (t *tag) Scan(state fmt.ScanState, verb rune) error {
	token, err := state.Token(true, nil)
	if err != nil {
		return err
	}
	if len(token) == 0 {
		return errors.New("tag: no token")
	}
	t.text = "<" + string(token) + ">"
	return nil
}

func main() {
	basic()
	formats()
	lines()
	failures()
	floats()
	custom()
}

func basic() {
	var a int
	var b string
	var c float64
	var d bool
	n, err := fmt.Sscan("12 hi 2.5 true", &a, &b, &c, &d)
	fmt.Println(n, a, b, c, d, err)

	var e, f int
	n, err = fmt.Sscan("  7\n8  ", &e, &f)
	fmt.Println(n, e, f, err)

	var u uint8
	var i8 int8
	var i16 int16
	var i64 int64
	var u64 uint64
	var f32 float32
	n, err = fmt.Sscan("255 -128 -32768 9007199254740993 18446744073709551615 1.5",
		&u, &i8, &i16, &i64, &u64, &f32)
	fmt.Println(n, u, i8, i16, i64, u64, f32, err)

	var r rune
	n, err = fmt.Sscan("65", &r)
	fmt.Println(n, r, err)
}

func formats() {
	var x, y int
	n, err := fmt.Sscanf("3-4", "%d-%d", &x, &y)
	fmt.Println(n, x, y, err)

	var name string
	var age int
	n, err = fmt.Sscanf("ada is 36", "%s is %d", &name, &age)
	fmt.Println(n, name, age, err)

	var hex, oct, bin int
	n, err = fmt.Sscanf("ff 17 101", "%x %o %b", &hex, &oct, &bin)
	fmt.Println(n, hex, oct, bin, err)

	var ch rune
	var rest string
	n, err = fmt.Sscanf("Xyz", "%c%s", &ch, &rest)
	fmt.Println(n, string(ch), rest, err)

	var quoted string
	n, err = fmt.Sscanf(`"a b"`, "%q", &quoted)
	fmt.Printf("%d %q %v\n", n, quoted, err)

	var raw []byte
	n, err = fmt.Sscanf("6869", "%x", &raw)
	fmt.Printf("%d %q %v\n", n, string(raw), err)

	var wide int
	var tail string
	n, err = fmt.Sscanf("12345", "%3d%s", &wide, &tail)
	fmt.Println(n, wide, tail, err)

	var pct int
	n, err = fmt.Sscanf("50%", "%d%%", &pct)
	fmt.Println(n, pct, err)

	var g float64
	n, err = fmt.Sscanf("1.5e3", "%g", &g)
	fmt.Println(n, g, err)

	var vhex int
	n, err = fmt.Sscanf("0x1f", "%v", &vhex)
	fmt.Println(n, vhex, err)
}

func lines() {
	var a, b int
	n, err := fmt.Sscanln("1 2", &a, &b)
	fmt.Println(n, a, b, err)

	var c int
	n, err = fmt.Sscanln("3\n4", &c)
	fmt.Println(n, c, err)

	var d, e int
	n, err = fmt.Sscanln("5\n6", &d, &e)
	fmt.Println(n, d, err != nil)

	var first, second int
	n, err = fmt.Fscan(strings.NewReader("10\n20"), &first, &second)
	fmt.Println(n, first, second, err)

	var only int
	n, err = fmt.Fscanln(strings.NewReader("30\n40"), &only)
	fmt.Println(n, only, err)

	var p, q int
	n, err = fmt.Fscanf(strings.NewReader("1 2"), "%d %d", &p, &q)
	fmt.Println(n, p, q, err)
}

func failures() {
	var a int
	n, err := fmt.Sscan("abc", &a)
	fmt.Println(n, err)

	n, err = fmt.Sscan("", &a)
	fmt.Println(n, err)

	var b, c int
	n, err = fmt.Sscanf("1", "%d %d", &b, &c)
	fmt.Println(n, err)

	n, err = fmt.Sscanf("1 2", "%d", &b)
	fmt.Println(n, err)

	n, err = fmt.Sscanf("x", "y", &b)
	fmt.Println(n, err != nil)

	var s string
	n, err = fmt.Sscanf("1", "%d", &s)
	fmt.Println(n, err != nil)

	var ptr *int
	n, err = fmt.Sscan("1", &ptr)
	fmt.Println(n, err != nil)
}

// floats reads a float that a literal of the format follows. The token of a
// float must end at the first rune that no float literal holds.
func floats() {
	var v float64
	n, err := fmt.Sscanf("21.5C", "%fC", &v)
	fmt.Println(n, err, v)

	var hex float64
	n, err = fmt.Sscanf("0x1p4 rest", "%g rest", &hex)
	fmt.Println(n, err, hex)

	var e float64
	n, err = fmt.Sscanf("1.5e-3ms", "%fms", &e)
	fmt.Println(n, err, e)

	var nan, inf float64
	n, err = fmt.Sscanf("NaN Inf", "%f %f", &nan, &inf)
	fmt.Println(n, err, nan != nan, inf)

	var bad float64
	n, err = fmt.Sscanf("abc", "%f", &bad)
	fmt.Println(n, err != nil)

	var sign float64
	n, err = fmt.Sscanf("-2.5x", "%fx", &sign)
	fmt.Println(n, err, sign)
}

func custom() {
	var t tag
	n, err := fmt.Sscan("word", &t)
	fmt.Println(n, t.text, err)

	var t2 tag
	n, err = fmt.Sscanf("  word rest", "%s", &t2)
	fmt.Println(n, t2.text, err)
}
