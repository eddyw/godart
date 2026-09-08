// The witness of the 3 guards of fmt: the report of an argument that no verb
// reads, the recovery of a panic inside a method of the value, and the report
// of a width or of a precision above the bound of Go.
package main

import (
	"fmt"
	"os"
	"text/template"
)

type failing struct{}

func (failing) String() string { panic("boom") }

type failErr struct{}

func (failErr) Error() string { panic("bang") }

type node struct{ next *node }

func (n *node) String() string { return fmt.Sprint(n.next.next) }

type badFormat struct{}

func (badFormat) Format(f fmt.State, verb rune) { panic("crash") }

func main() {
	fmt.Println("-- the extra argument --")
	fmt.Println(fmt.Sprintf("%d", 1, 2))
	args := []any{1, 2, "x"}
	fmt.Println(fmt.Sprintf("%d", args...))
	fmt.Println(fmt.Sprintf("no verb", 1))
	fmt.Println(fmt.Sprintf("%d %d", 1, 2, 3, nil))
	fmt.Println(fmt.Sprintf("%[1]d", 1, 2))
	t := template.Must(template.New("t").Parse(`{{printf "%d" 1 2}}`))
	_ = t.Execute(os.Stdout, nil)
	fmt.Println()

	fmt.Println("-- the panic inside a method --")
	fmt.Printf("%v|\n", failing{})
	fmt.Printf("%s|\n", failing{})
	fmt.Printf("%d|\n", failErr{})
	fmt.Printf("%v|\n", []any{failing{}})
	fmt.Printf("%v|\n", badFormat{})
	var n *node
	fmt.Printf("%v|\n", n)

	fmt.Println("-- the width above the bound --")
	fmt.Printf("%*d|\n", 1<<40, 1)
	fmt.Printf("%.*f|\n", 1<<40, 1.5)
	fmt.Printf("%*.*d|\n", 1<<40, 1<<40, 1)
	fmt.Printf("%*d|\n", 1000001, 1)
	fmt.Printf("%*d|\n", -1000001, 1)
}
