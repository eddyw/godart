// The program proves each report of the package fmt: an argument that no verb
// reads, a width and a precision above the limit, the back quote of the flag #
// with the verb q, and a panic inside a String method or an Error method.
package main

import (
	"errors"
	"fmt"
)

type named string

type spec struct{}

func (spec) Format(st fmt.State, verb rune) {
	fmt.Fprintf(st, "<%c w=%v p=%v>", verb, flagText(st), fmt.FormatString(st, verb))
}

func flagText(st fmt.State) string {
	w, ok := st.Width()
	if !ok {
		return "none"
	}
	return fmt.Sprint(w)
}

type failing struct{}

func (failing) String() string { panic("boom") }

type failErr struct{}

func (failErr) Error() string { panic(42) }

type hexed struct{ n int }

func (h hexed) String() string { return fmt.Sprintf("h%d", h.n) }

type nilStr struct{ v int }

func (n *nilStr) String() string { return fmt.Sprint(n.v) }

func reports() {
	fmt.Printf("%d|\n", 1, 2)
	fmt.Printf("%d|\n", 1, "a", 2.5, []int{1}, nil)
	fmt.Printf("no verb|\n", 1)
	fmt.Println(fmt.Sprintf("%d", 1, 2))
	fmt.Printf("%*d|\n", 1<<40, 1)
	fmt.Printf("%.*d|\n", 1<<40, 1)
	fmt.Printf("%99999999999d|\n", 1)
	fmt.Printf("%.99999999999d|\n", 1)
	fmt.Printf("%#q|%#q|%#q|\n", "plain", "with\ttab", named("named"))
	fmt.Printf("%#q|%+q|%.3q|\n", "a`b", "héllo", "abcdef")
	fmt.Printf("%q|%#q|\n", named("n"), 'x')
}

func panics() {
	fmt.Printf("%v|%s|%q|\n", failing{}, failing{}, failing{})
	fmt.Printf("%v|%s|\n", failErr{}, failErr{})
	fmt.Printf("%x|%X|%v|%s|%q|\n", hexed{3}, hexed{3}, hexed{3}, hexed{3}, hexed{3})
	var p *nilStr
	fmt.Printf("%v|%s|\n", p, p)
	fmt.Printf("%v|\n", []any{failing{}, hexed{1}})
}

// leaves proves that a Format method reaches each leaf of a compound value,
// that the precision cuts the text of a method, and that the quote of a text
// reads the print table of Go.
func leaves() {
	e := errors.New("plain")
	fmt.Printf("%x|%.2s|%.3v|%8.2s|%.1q|\n", e, e, e, e, e)
	fmt.Printf("%q|%v|%d|%8v|\n", []any{spec{}}, []any{spec{}}, map[string]spec{"a": {}}, []any{spec{}})
	fmt.Printf("%v|%+v|%#v|\n", struct{ S spec }{}, [2]spec{}, map[int]spec{1: {}})
	for _, v := range []any{"\uffff", "é", "a`b", "tab\there", "\x00", "日本", []byte("\uffff"), []byte("é"), []byte("abc")} {
		fmt.Printf("%q|%+q|%#q|%v|\n", v, v, v, v)
	}
	fmt.Printf("%q|%q|\n", []string{"\uffff", "é"}, map[string]string{"a": "\uffff"})
}

func main() {
	reports()
	panics()
	leaves()
}
