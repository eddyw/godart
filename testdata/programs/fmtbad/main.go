package main

import "fmt"

type celsius float64

type point struct{ X, Y int }

type named string

func main() {
	kinds()
	widths()
	compounds()
}

// kinds proves the report of a verb that the kind of a value rejects. Go
// writes the type of the value and the value in the format of the verb v.
func kinds() {
	values := []any{
		42, int8(-128), int16(1), int32(2), int64(3),
		uint(4), uint8(5), uint16(6), uint32(7), uint64(8), uintptr(9),
		float32(0.1), float64(2.5), celsius(1),
		"text", named("n"), true, 'r', complex64(1 + 2i), complex128(3 + 4i),
	}
	verbs := []string{"v", "d", "s", "q", "t", "c", "U", "e", "f", "g", "b", "o", "x", "X"}
	for _, v := range values {
		for _, verb := range verbs {
			fmt.Printf("%%%-2s %-14T %s\n", verb, v, fmt.Sprintf("%"+verb, v))
		}
	}
}

// widths proves that the width of a verb stands inside the report and not
// around it.
func widths() {
	forms := []string{"%5d", "%05d", "%-5d", "%5.2d", "%+5d", "% d", "%#d"}
	for _, f := range forms {
		fmt.Printf("%-6s %s|%s|\n", f, fmt.Sprintf(f, "a"), fmt.Sprintf(f, true))
	}
}

// compounds proves that Go gives the verb to each element of a compound value,
// and that each element writes its own report.
func compounds() {
	fmt.Printf("%.1f|%3.1f|%q|%d\n", []int{1, 2}, []int{1, 2}, []int{1, 2}, point{1, 2})
	fmt.Printf("%s|%d|%v|%q|%x\n", []byte("ab"), []byte("ab"), []byte("ab"), []byte("ab"), []byte("ab"))
	fmt.Printf("%c|%t|%s\n", point{1, 2}, map[string]int{"a": 1}, [2]float64{1, 2})
	fmt.Printf("%d|%q\n", struct{ A string }{"x"}, struct{ B bool }{true})
}
