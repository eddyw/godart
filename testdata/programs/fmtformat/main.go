package main

import (
	"fmt"
	"strings"
)

// tag writes its own text under each verb.
type tag struct {
	name  string
	count int
}

func (t tag) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "tag(name=%s count=%d)", t.name, t.count)
			return
		}
		fmt.Fprintf(f, "tag:%s", t.name)
	case 's':
		w, ok := f.Width()
		if ok {
			fmt.Fprintf(f, "%s", strings.Repeat(t.name, w))
			return
		}
		fmt.Fprintf(f, "%s", t.name)
	case 'd':
		p, ok := f.Precision()
		fmt.Fprintf(f, "%d/%v/%v", t.count, p, ok)
	default:
		fmt.Fprintf(f, "%%!%c(tag)", verb)
	}
}

// mirror writes the verb again.
type mirror struct{}

func (mirror) Format(f fmt.State, verb rune) {
	fmt.Fprint(f, fmt.FormatString(f, verb))
}

func main() {
	t := tag{name: "go", count: 3}
	fmt.Printf("%v\n", t)
	fmt.Printf("%+v\n", t)
	fmt.Printf("%3s\n", t)
	fmt.Printf("%.7d\n", t)
	fmt.Printf("%q\n", t)
	fmt.Println(t)
	fmt.Print(t, "\n")
	fmt.Println(fmt.Sprint(t))
	fmt.Println(fmt.Sprintf("%v|%s", t, t))

	m := mirror{}
	fmt.Printf("%v\n", m)
	fmt.Printf("%+#08.3d\n", m)
	fmt.Printf("%-5s\n", m)
	fmt.Println(fmt.Sprintf("%x", m))

	var s fmt.State
	fmt.Println(s == nil)
	var _ fmt.Formatter = t
}
