// The program proves the package fmt against real Go: each verb on each kind
// of value, each flag, a width and a precision from an argument, an argument
// index, each report of a wrong call, the methods Stringer, GoStringer,
// Formatter, and Error, the scan family, and the space rule of Sprint.
//
// The program prints no address. Go writes an address for a pointer that is
// not a pointer to a struct, and for each pointer below the top level.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-28s %v\n", label, value)
}

// sprintf and errorf hold the function in a variable, thus vet reads p as no
// wrapper of Printf and it accepts each wrong format on purpose.
var sprintf = fmt.Sprintf

var errorf = fmt.Errorf

// p prints the format and its result on 1 line, thus a diff names the format.
func p(format string, args ...any) {
	fmt.Printf("%-16q %q\n", format, sprintf(format, args...))
}

type pt struct {
	X, Y int
}

type nested struct {
	Name  string
	At    pt
	Tags  []string
	Score map[string]int
	Any   any
	Ptr   *pt
	err   error
}

type celsius float64

func (c celsius) String() string { return fmt.Sprintf("%.1f°C", float64(c)) }

type token string

func (t token) GoString() string { return "token(" + string(t) + ")" }

type loud struct{ Text string }

func (l loud) String() string { return strings.ToUpper(l.Text) }

func (l loud) GoString() string { return "LOUD{" + l.Text + "}" }

type spec struct{}

// Format writes each part of the verb, thus the program proves that fmt hands
// the flags, the width, and the precision to the method.
func (spec) Format(st fmt.State, verb rune) {
	w, hasW := st.Width()
	pr, hasP := st.Precision()
	fmt.Fprintf(st, "<%c plus=%v minus=%v sharp=%v space=%v zero=%v w=%d/%v p=%d/%v %s>",
		verb, st.Flag('+'), st.Flag('-'), st.Flag('#'), st.Flag(' '), st.Flag('0'), w, hasW, pr, hasP, fmt.FormatString(st, verb))
}

type nilSafe struct{ v int }

func (n *nilSafe) String() string {
	if n == nil {
		return "nil receiver"
	}
	return fmt.Sprint("value ", n.v)
}

type myErr struct{ code int }

func (e *myErr) Error() string { return fmt.Sprintf("myErr %d", e.code) }

type bits uint8

func (b bits) String() string { return fmt.Sprintf("%08b", uint8(b)) }

func verbs() {
	fmt.Println("-- bool --")
	for _, f := range []string{"%t", "%v", "%5t|", "%-5t|", "%d", "%s", "%q", "%08t"} {
		p(f, true)
	}
	fmt.Println("-- int --")
	for _, v := range []any{0, 42, -42, 255, -255, int8(-128), int16(32767), int32(-1), int64(math.MinInt64), int64(math.MaxInt64), uint8(255), uint16(65535), uint32(math.MaxUint32), uint64(math.MaxUint64), uintptr(0)} {
		for _, f := range []string{"%d", "%b", "%o", "%O", "%x", "%X", "%c", "%v", "%8d|", "%-8d|", "%08d", "%+d", "% d", "%#x", "%#o", "%#b", "%#X", "%.3d", "%8.3d|", "%+08d", "%-08d|", "% 08d", "%x", "% x", "%#8x|", "%#08x", "%-#8x|", "%+x", "%.0d", "%5.0d|"} {
			p(f, v)
		}
	}
	fmt.Println("-- rune --")
	for _, r := range []rune{'a', 'é', '日', 0x1F600, '\n', 0, 0x7f, 0xfffd, -1, 0x110000, 0xd800} {
		for _, f := range []string{"%c", "%q", "%d", "%x", "%v", "%+q", "%5c|", "%-5q|", "%#q"} {
			p(f, r)
		}
	}
	fmt.Println("-- float --")
	for _, v := range []any{0.0, math.Copysign(0, -1), 1.0, -1.5, 0.5, 2.5, 1.005, 123456.789, 1e21, 1e20, 1e-5, 1e-4, 100000.0, 1000000.0, math.Pi, 1e100, 5e-324, math.MaxFloat64, math.Inf(1), math.Inf(-1), math.NaN(), float32(0.1), float32(1.0 / 3), float32(16777216), float32(math.Pi), float32(math.Inf(-1)), float32(1e-45)} {
		for _, f := range []string{"%v", "%e", "%E", "%f", "%F", "%g", "%G", "%x", "%X", "%b", "%.2f", "%.0f", "%.3e", "%.3g", "%10.2f|", "%-10.2f|", "%#v", "%8.3v|", "%.10g", "%.20g", "%6.1e|", "%+v", "%-8v|", "%.0e", "%.1g"} {
			p(f, v)
		}
	}
	fmt.Println("-- complex --")
	for _, v := range []any{complex(1, 2), complex(-1.5, -2.5), complex(0, 0), complex(math.Inf(1), math.NaN()), complex64(complex(0.1, 0.2))} {
		for _, f := range []string{"%v", "%f", "%.2f", "%g", "%e", "%+v", "%#v", "%d"} {
			p(f, v)
		}
	}
	fmt.Println("-- string --")
	for _, s := range []string{"", "abc", "日本語", "a\tb\n", "quote\"back\\", "\xff\xfe", "with space", "`back`", "\x00\x7f", "é\u00a0\u2028"} {
		for _, f := range []string{"%s", "%q", "%v", "%x", "%X", "% x", "% X", "%# x", "%d", "%t", "%c", "%8s|", "%-8s|", "%.2s", "%8.2s|", "%-8.2s|", "%08s", "%+q", "%8q|", "%-8q|", "%.1q", "%T", "%5T|", "%e"} {
			p(f, s)
		}
	}
	fmt.Println("-- byte slice --")
	for _, b := range [][]byte{[]byte("abc"), []byte("日本"), []byte{0, 1, 255}, {}, nil, []byte("a\"b\n")} {
		for _, f := range []string{"%s", "%q", "%x", "% x", "%X", "% X", "%# x", "%v", "%d", "%c", "%08b", "%o", "%#v", "%8s|", "%-8q|", "%.2s", "%6x|", "%-6x|", "%06x", "%T", "%5d|", "%03d", "%x" + "%%"} {
			p(f, b)
		}
	}
	fmt.Println("-- the other slices and arrays --")
	for _, v := range []any{[]int{1, 2, 3}, []int{}, []int(nil), []string{"a", "b c", ""}, [2]bool{true, false}, [3]float64{1.5, 2, 1e21}, []any{1, "a", nil, 2.5, []int{1}, true, pt{1, 2}}, [][]int{{1}, {2, 3}, nil}, []rune("a日"), []uint8{1, 2}, [0]int{}, []celsius{1, 2.5}} {
		for _, f := range []string{"%v", "%q", "%+v", "%#v", "%T", "%b"} {
			p(f, v)
		}
	}
	fmt.Println("-- map --")
	for _, v := range []any{map[string]int{"b": 2, "a": 1, "c": 3}, map[int]string{3: "c", -1: "a", 0: "b"}, map[string][]int{"x": {1}, "y": nil}, map[bool]int{true: 1, false: 0}, map[float64]string{2.5: "b", -1: "a", 0: "z"}, map[string]int{}, map[string]int(nil), map[pt]string{{2, 1}: "b", {1, 2}: "a", {1, 1}: "z"}, map[string]any{"n": nil, "i": 1, "s": "x"}, map[rune]int{'b': 1, 'a': 2}} {
		for _, f := range []string{"%v", "%d", "%x", "%+v", "%#v", "%T", "%-5v|", "%c"} {
			p(f, v)
		}
	}
	fmt.Println("-- struct --")
	n := nested{Name: "n", At: pt{1, 2}, Tags: []string{"t1", "t2"}, Score: map[string]int{"b": 2, "a": 1}, Any: 3.5}
	for _, v := range []any{pt{1, 2}, pt{}, n, nested{}, struct{ A, B string }{"x", "y"}, struct{}{}, struct {
		In  pt
		Str celsius
		L   loud
	}{pt{3, 4}, 21.5, loud{"quiet"}}, struct{ F func() }{}, struct{ C chan int }{}, struct{ M map[string]int }{map[string]int{"k": 1}}} {
		for _, f := range []string{"%v", "%+v", "%#v", "%T"} {
			p(f, v)
		}
	}
	// The verb %#v of an error writes an address, thus the 2 values with an
	// error take each other verb alone.
	for _, v := range []any{struct{ E error }{errors.New("field")}, []error{errors.New("e"), nil}} {
		for _, f := range []string{"%v", "%+v", "%T"} {
			p(f, v)
		}
	}
	fmt.Println("-- pointer to a struct --")
	ptr := &pt{7, 8}
	var nilPtr *pt
	for _, f := range []string{"%v", "%+v", "%#v", "%T"} {
		p(f, ptr)
		p(f, nilPtr)
		p(f, &n)
	}
	fmt.Println("-- nil of each kind --")
	var nilErr error
	var nilAny any
	var nilSlice []int
	var nilMap map[string]int
	var nilFunc func()
	var nilChan chan int
	var nilIface fmt.Stringer
	var nilStr *nilSafe
	var nilBytes []byte
	for _, v := range []any{nil, nilErr, nilAny, nilSlice, nilMap, nilFunc, nilChan, nilIface, nilPtr, nilStr, nilBytes, []any{nil}, struct{ A any }{}, struct{ P *pt }{}} {
		for _, f := range []string{"%v", "%+v", "%#v", "%T", "%5v|"} {
			p(f, v)
		}
	}
	// The verb %p writes an address for each value that is not nil, thus the
	// nil values alone take it.
	fmt.Println("-- the type verb --")
	for _, v := range []any{1, int8(1), uint(1), uintptr(1), "s", 1.5, float32(1), complex(1, 2), true, 'r', byte(1), []int{}, [2]string{}, map[string]bool{}, pt{}, &pt{}, nested{}, celsius(1), token(""), errors.New("e"), &myErr{}, fmt.Errorf("w: %w", errors.New("x")), fmt.Errorf("p: %v", 1), func() {}, func(int) string { return "" }, make(chan int), make(<-chan string), struct{ A int }{}, []any{}, [][]pt{}, map[pt][]celsius{}, nilPtr, nilSlice, loud{}, &loud{}, new(int), []*pt{}} {
		p("%T", v)
	}
}

func methods() {
	c := celsius(36.6)
	for _, f := range []string{"%v", "%s", "%f", "%.2f", "%q", "%+v", "%#v", "%10s|", "%-10v|", "%T", "%e", "%g", "%08.2f"} {
		p(f, c)
	}
	fmt.Println("-- GoStringer --")
	t := token("abc")
	for _, f := range []string{"%v", "%s", "%#v", "%q", "%+v", "%x", "%T"} {
		p(f, t)
	}
	fmt.Println("-- both methods --")
	l := loud{"hey"}
	for _, f := range []string{"%v", "%s", "%#v", "%+v", "%q", "%d", "%T", "%8v|", "%-8s|", "%+q", "%08v", "%c"} {
		p(f, l)
		p(f, &l)
	}
	fmt.Println("-- Formatter --")
	for _, f := range []string{"%s", "%d", "%x", "%+v", "%-v", "%#v", "% v", "%0v", "%8v", "%.3v", "%8.3v", "%*v", "%.*d", "%[1]*[2]d", "%%%v%%", "%z", "%!"} {
		switch f {
		case "%*v":
			p(f, 7, spec{})
		case "%.*d":
			p(f, 4, spec{})
		case "%[1]*[2]d":
			p(f, 9, spec{})
		default:
			p(f, spec{})
		}
	}
	fmt.Println("-- error --")
	e := &myErr{code: 4}
	for _, f := range []string{"%v", "%+v", "%T", "%10s|", "%-10v|", "%t", "%c"} {
		p(f, e)
	}
	wrapped := fmt.Errorf("wrap %d: %w", 1, e)
	for _, f := range []string{"%v", "%s", "%q", "%+v", "%T"} {
		p(f, wrapped)
	}
	joined := errors.Join(errors.New("a"), errors.New("b"))
	for _, f := range []string{"%v", "%s", "%q", "%T"} {
		p(f, joined)
	}
	plain := errors.New("plain")
	for _, f := range []string{"%v", "%s", "%q", "%T", "%+v", "%10v|"} {
		p(f, plain)
	}
	var np *nilSafe
	for _, f := range []string{"%v", "%s", "%q", "%#v", "%+v", "%T"} {
		p(f, np)
		p(f, &nilSafe{v: 3})
	}
	b := bits(5)
	for _, f := range []string{"%v", "%s", "%b", "%08b", "%q", "%c", "%#v", "%T", "%5v|", "%-12s|"} {
		p(f, b)
	}
	p("%v %s", []bits{1, 2}, []bits{3})
	p("%q", []any{celsius(1), token("t"), loud{"l"}, 1, "s", 'r'})
	p("%#v", []any{celsius(1), token("t"), loud{"l"}, 1, "s"})
}

func flagsAndWidths() {
	fmt.Println("-- star --")
	p("%*d|", 6, 42)
	p("%-*d|", 6, 42)
	p("%*d|", -6, 42)
	p("%.*f|", 2, 3.14159)
	p("%*.*f|", 9, 3, 3.14159)
	p("%-*.*f|", 9, 3, 3.14159)
	p("%*s|", 5, "ab")
	p("%.*s|", 1, "abc")
	p("%*d|", "x", 42)
	p("%.*f|", "x", 1.5)
	p("%.*f|", -1, 1.5)
	p("%*d %*d|", 3, 1, 4, 2)
	p("%*d|", 3)
	p("%*.*d|", 5, 3, 7)
	p("%0*d|", 5, 42)
	p("%0*d|", 5, -42)
	p("%*x|", 6, 255)
	p("%#*x|", 6, 255)
	p("%*q|", 8, "a")
	p("%*c|", 3, 'x')
	p("%*v|", 6, []int{1, 2})
	p("%*t|", 6, true)
	fmt.Println("-- argument index --")
	p("%[2]d %[1]d", 1, 2)
	p("%[1]d %d %d", 5, 6, 7)
	p("%[2]d %d", 1, 2)
	p("%[3]d", 1, 2)
	p("%[0]d", 1)
	p("%[-1]d", 1)
	p("%[1]d %[1]d %[1]d", 9)
	p("%[1]*d|", 5, 42)
	p("%[1]*[2]d|", 5, 42)
	p("%[2]*[1]d|", 42, 5)
	p("%[1]*.[2]*[3]f|", 8, 2, 3.14159)
	p("%.[1]*f|", 2, 3.14159)
	p("%[1]s %[2]d", "a")
	p("%[1]v %[3]v", 1, 2)
	p("%[2]v", 1, 2, 3)
	p("%d %[1]d %d", 1, 2)
	p("%[]d", 1)
	p("%[x]d", 1)
	p("%[1", 1)
	p("%[1]", 1)
	p("%[1]T", 1.5)
	p("%[2]q %[1]q", "a", "b")
	fmt.Println("-- the reports --")
	p("%d %d", 1)
	p("100%")
	p("%%")
	p("%%%%")
	p("%z", 1)
	p("%!", 1)
	p("%!d", 1)
	p("%y %d", 1, 2)
	p("%.", 1)
	p("%.d", 1)
	p("%  d", 1)
	p("%++d", 1)
	p("%--5d|", 1)
	p("%00d", 1)
	p("%+-d|", 1)
	p("%-+5d|", 1)
	p("% -5d|", 1)
	p("%+ d", 1)
	p("%s %s", []string{"a"}, nil)
	p("%s %d", nil, nil)
	p("%q", nil)
	p("%x", nil)
	p("%t", nil)
	p("%c", nil)
	p("%e", nil)
	p("%p", nil)
	p("%T", nil)
	p("%!v", nil)
	p("%.2s|", 1.5)
	p("%8v|", spec{})
	p("%e", "text")
	p("%f", true)
	p("%b", "s")
	p("%o", 1.5)
	p("%c", "ab")
	p("%q", true)
	p("%x", pt{1, 2})
	p("%t", pt{1, 2})
	p("%x", []bool{true})
	p("%c", []string{"a"})
	p("%+d|", "s")
	p("% d|", "s")
	p("%#d|", "s")
	p("%#s|", 1)
	p("%d %d %d", 1, 2)
	p("%d%%%d", 1)
	p("%v %v", 1)
	p("%.2f %.2f", 1.5)
	p("%*d|", 5)
	p("%.*d|", 5)
	p("%[2]d %[3]d", 1)
	p("%d %[5]d %d", 1, 2, 3)
	p("%[3]d %d", 1, 2, 3)
	p("%[3]d %d %d", 1, 2, 3)
	p("%d %[1]d %d %d", 1, 2)
	p("%q%q%q", "a")
}

func scans() {
	var (
		i  int
		u  uint
		f  float64
		s  string
		b  bool
		r  rune
		by []byte
		x  int
		u8 uint8
	)
	report := func(label string, n int, err error, values ...any) {
		fmt.Printf("%-28s n=%d err=%v %v\n", label, n, err, values)
	}
	n, err := fmt.Sscanf("10 abc 3.5 true x 1f", "%d %s %f %t %c %x", &i, &s, &f, &b, &r, &x)
	report("Sscanf each verb", n, err, i, s, f, b, r, x)
	n, err = fmt.Sscanf("age: 42, name: Ada", "age: %d, name: %s", &i, &s)
	report("Sscanf literal text", n, err, i, s)
	n, err = fmt.Sscanf("age: 42", "name: %d", &i)
	report("Sscanf literal mismatch", n, err, i)
	n, err = fmt.Sscanf("abc", "%d", &i)
	report("Sscanf bad integer", n, err, i)
	n, err = fmt.Sscanf("123456", "%3d%3d", &i, &x)
	report("Sscanf width", n, err, i, x)
	n, err = fmt.Sscanf("1234", "%d%d", &i, &x)
	report("Sscanf no space", n, err, i, x)
	n, err = fmt.Sscanf("12 34", "%d%d", &i, &x)
	report("Sscanf space between", n, err, i, x)
	n, err = fmt.Sscanf("12", "%d %d", &i, &x)
	report("Sscanf too short", n, err, i, x)
	n, err = fmt.Sscanf("", "%d", &i)
	report("Sscanf empty", n, err, i)
	n, err = fmt.Sscanf("12 34", "%d", &i)
	report("Sscanf extra input", n, err, i)
	n, err = fmt.Sscanf("12 34", "%d", &i, &x)
	report("Sscanf extra args", n, err, i, x)
	n, err = fmt.Sscanf("1\n2", "%d %d", &i, &x)
	report("Sscanf newline in space", n, err, i, x)
	n, err = fmt.Sscanf("1\n2", "%d\n%d", &i, &x)
	report("Sscanf newline in format", n, err, i, x)
	n, err = fmt.Sscanf("hello world", "%s %s", &s, &by)
	report("Sscanf into bytes", n, err, s, string(by))
	n, err = fmt.Sscanf("日本 語", "%s %c", &s, &r)
	report("Sscanf runes", n, err, s, r)
	n, err = fmt.Sscanf("T f 1 0 TRUE", "%t %t %t %t %t", &b, &b, &b, &b, &b)
	report("Sscanf bools", n, err, b)
	var b2, b3 bool
	n, err = fmt.Sscanf("t F", "%t %t", &b2, &b3)
	report("Sscanf 2 bools", n, err, b2, b3)
	n, err = fmt.Sscanf("-1", "%d", &u)
	report("Sscanf negative unsigned", n, err, u)
	n, err = fmt.Sscanf("1.5", "%d", &i)
	report("Sscanf float as int", n, err, i)
	n, err = fmt.Sscanf("%", "%%")
	report("Sscanf percent", n, err)
	n, err = fmt.Sscanf("a%b", "a%%b")
	report("Sscanf percent text", n, err)
	n, err = fmt.Sscanf("5", "%d", &s)
	report("Sscanf wrong type", n, err, s)
	n, err = fmt.Sscanf("  7  ", "%d", &i)
	report("Sscanf leading space", n, err, i)
	n, err = fmt.Sscanf("+7", "%d", &i)
	report("Sscanf plus", n, err, i)
	fmt.Println("-- Sscan --")
	var a1, a2, a3 int
	n, err = fmt.Sscan("1 2 3", &a1, &a2, &a3)
	report("Sscan", n, err, a1, a2, a3)
	n, err = fmt.Sscan("1\n2\n\n3", &a1, &a2, &a3)
	report("Sscan newlines", n, err, a1, a2, a3)
	n, err = fmt.Sscan("1 2", &a1, &a2, &a3)
	report("Sscan too few", n, err, a1, a2, a3)
	n, err = fmt.Sscan("1 x 3", &a1, &a2, &a3)
	report("Sscan bad", n, err, a1, a2, a3)
	n, err = fmt.Sscan("word 2.5 true 'c' 99", &s, &f, &b, &r, &u8)
	report("Sscan mixed", n, err, s, f, b, r, u8)
	n, err = fmt.Sscan(`"quoted" plain`, &s, &by)
	report("Sscan quoted string", n, err, s, string(by))
	n, err = fmt.Sscan("", &a1)
	report("Sscan empty", n, err)
	n, err = fmt.Sscan("   ", &a1)
	report("Sscan blank", n, err)
	n, err = fmt.Sscan("5")
	report("Sscan no args", n, err)
	fmt.Println("-- Sscanln --")
	n, err = fmt.Sscanln("1 2", &a1, &a2)
	report("Sscanln", n, err, a1, a2)
	n, err = fmt.Sscanln("1 2\n3", &a1, &a2, &a3)
	report("Sscanln stops at newline", n, err, a1, a2, a3)
	n, err = fmt.Sscanln("1\n2", &a1, &a2)
	report("Sscanln early newline", n, err, a1, a2)
	n, err = fmt.Sscanln("1 2 3", &a1, &a2)
	report("Sscanln extra", n, err, a1, a2)
	n, err = fmt.Sscanln("1 2\n", &a1, &a2)
	report("Sscanln trailing newline", n, err, a1, a2)
	n, err = fmt.Sscanln("", &a1)
	report("Sscanln empty", n, err)
	n, err = fmt.Sscanln("\n", &a1)
	report("Sscanln only newline", n, err)
	n, err = fmt.Sscanln("a b", &s, &by)
	report("Sscanln strings", n, err, s, string(by))
	fmt.Println("-- Fscan --")
	rd := strings.NewReader("10 20\n30 forty")
	n, err = fmt.Fscan(rd, &a1, &a2)
	report("Fscan", n, err, a1, a2)
	rd = strings.NewReader("a\nb\n")
	n, err = fmt.Fscanln(rd, &s)
	report("Fscanln line 1", n, err, s)
	n, err = fmt.Fscanf(strings.NewReader("k=v"), "k=%s", &s)
	report("Fscanf literal", n, err, s)
}

func prints() {
	var sb strings.Builder
	n, err := fmt.Fprint(&sb, 1, 2, "a", "b", 3.5, true, nil, []int{1}, pt{1, 2}, 'r')
	row("Fprint", fmt.Sprintf("%d %v %q", n, err, sb.String()))
	sb.Reset()
	n, err = fmt.Fprintln(&sb, 1, 2, "a", "b", 3.5, true, nil)
	row("Fprintln", fmt.Sprintf("%d %v %q", n, err, sb.String()))
	sb.Reset()
	n, err = fmt.Fprintf(&sb, "%d-%s-%v", 1, "a", 2.5)
	row("Fprintf", fmt.Sprintf("%d %v %q", n, err, sb.String()))
	var buf bytes.Buffer
	fmt.Fprint(&buf, "into a buffer ", 42)
	row("Fprint Buffer", fmt.Sprintf("%q", buf.String()))
	n, err = fmt.Fprint(failWriter{}, "x")
	row("Fprint error", fmt.Sprint(n, " ", err))
	n, err = fmt.Fprintf(failWriter{}, "%d", 1)
	row("Fprintf error", fmt.Sprint(n, " ", err))
	fmt.Println("-- Sprint against Sprintln --")
	cases := [][]any{
		{"a", "b"}, {1, 2}, {"a", 1, "b"}, {1, "a", 2}, {1.5, 2}, {nil, nil}, {[]int{1}, []int{2}}, {true, false}, {"a"}, {}, {1},
		{"a", "b", "c"}, {1, 2, 3}, {"a", 1}, {1, "a"}, {errors.New("e"), errors.New("f")}, {celsius(1), celsius(2)}, {token("t"), token("u")},
		{'a', 'b'}, {"", ""}, {"", 1, ""}, {1, "", 2}, {[]byte("x"), []byte("y")}, {pt{1, 2}, pt{3, 4}}, {nil, "a", nil},
		{1.5, "s", 2.5, 3}, {loud{"a"}, "b"}, {"a", loud{"b"}}, {uint8(1), int8(2)}, {complex(1, 1), 2}, {"日", "本"}, {[]string{"a"}, "b"},
		{map[string]int{"k": 1}, map[string]int{}}, {struct{}{}, struct{}{}}, {[2]int{}, "x"}, {any(nil), 1},
	}
	for _, c := range cases {
		fmt.Printf("Sprint   %-56q %q\n", fmt.Sprintf("%v", c), fmt.Sprint(c...))
		fmt.Printf("Sprintln %-56q %q\n", fmt.Sprintf("%v", c), fmt.Sprintln(c...))
	}
	row("Append", string(fmt.Append([]byte("pre:"), 1, "a", 2)))
	row("Appendln", fmt.Sprintf("%q", fmt.Appendln([]byte("pre:"), 1, "a")))
	row("Appendf", string(fmt.Appendf(nil, "%03d|%s", 7, "x")))
	row("Sprint of Sprint", fmt.Sprint(fmt.Sprint(1, 2), fmt.Sprint("a", "b")))
	row("Sprintf no args", fmt.Sprintf("plain"))
	row("Sprintf empty", fmt.Sprintf(""))
	row("Sprintf only percent", fmt.Sprintf("%%"))
	row("Println return", func() string {
		n, err := fmt.Println("Println return               ok")
		return fmt.Sprint(n, " ", err)
	}())
	row("Print return", func() string {
		n, err := fmt.Print("Print return                 ok\n")
		return fmt.Sprint(n, " ", err)
	}())
	row("Printf return", func() string {
		n, err := fmt.Printf("Printf return                %s\n", "ok")
		return fmt.Sprint(n, " ", err)
	}())
	row("Errorf text", fmt.Errorf("code %d: %s", 5, "text"))
	row("Errorf %w", fmt.Errorf("outer: %w", errors.New("inner")))
	row("Errorf %w twice", fmt.Errorf("%w + %w", errors.New("a"), errors.New("b")))
	row("Errorf %w non error", errorf("bad %w", 5))
	row("Errorf %w nil", fmt.Errorf("nil %w", nil))
	row("Errorf %T", fmt.Sprintf("%T %T", fmt.Errorf("a"), fmt.Errorf("a %w", errors.New("b"))))
	row("Errorf %v of error", fmt.Errorf("v: %v", errors.New("inner")))
	row("Errorf missing", errorf("%d %d", 1))
}

type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("the writer failed") }

func main() {
	fmt.Println("== the verbs ==")
	verbs()
	fmt.Println("== the methods ==")
	methods()
	fmt.Println("== the flags, the widths, and the reports ==")
	flagsAndWidths()
	fmt.Println("== the scan family ==")
	scans()
	fmt.Println("== the print family ==")
	prints()
}
