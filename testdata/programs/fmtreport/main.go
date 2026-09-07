// The program proves each report of the package fmt: an argument that no verb
// reads, a width and a precision above the limit, the back quote of the flag #
// with the verb q, and a panic inside a String method or an Error method.
package main

import "fmt"

type named string

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

func main() {
	reports()
	panics()
}
