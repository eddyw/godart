// fmtindex reads the star width, the star precision, and the explicit argument
// index of a format.
package main

import "fmt"

type point struct{ X, Y int }

func main() {
	fmt.Printf("[%*d]\n", 5, 42)
	fmt.Printf("[%-*d]\n", 5, 42)
	fmt.Printf("[%.*f]\n", 2, 3.14159)
	fmt.Printf("[%*.*f]\n", 8, 2, 3.14159)
	fmt.Printf("[%*s]\n", 6, "ab")
	fmt.Printf("[%*d]\n", -5, 42)
	fmt.Printf("[%.*s]\n", 2, "abcdef")
	fmt.Printf("[%.*f]\n", -1, 3.5)

	fmt.Printf("%d: %d %[2]s %d\n", 1, []byte{1, 2}, 3)
	fmt.Printf("%[2]d %[1]d\n", 11, 22)
	fmt.Printf("%[1]d %d %d\n", 1, 2)
	fmt.Printf("%[3]d\n", 1, 2)
	fmt.Printf("%[1]*d\n", 5, 42)
	fmt.Printf("%*[2]d\n", 4, 7)
	fmt.Printf("%.[2]d\n", 1, 3)
	fmt.Printf("%[x]d\n", 1)
	fmt.Printf("%[0]d\n", 1)
	fmt.Printf("%[2v\n", 1, 2)
	fmt.Println(fmt.Sprintf("%[2]v-%[1]v", "a", "b"))
	fmt.Printf("%[2]v %[1]v %v\n", point{1, 2}, "x")
	fmt.Printf("%+[1]v %[1]v\n", point{3, 4})
	fmt.Printf("%[1]s%[1]s\n", "ab")
	fmt.Printf("%08.3[1]f\n", 3.14159)
	for _, r := range []rune{'a', 'A', 'k', 0x212A, 'K', '1', 0x130, '\n', 0, 0x10FFFF, 0x7F, '世'} {
		fmt.Printf("%U %#U\n", r, r)
	}
	fmt.Printf("%U %#U\n", 65, 65)
	fmt.Printf("%8U|%-8U|%08U|\n", 'a', 'a', 'a')
	fmt.Printf("%U\n", "x")
	fmt.Printf("[%.2d] [%.5d] [%.5d] [%.0d] [%.0d]\n", 5, -42, 42, 0, 7)
	fmt.Printf("[%#.5x] [%.5x] [%#.5o] [%.3b]\n", 255, 255, 8, 5)
	fmt.Printf("[%8.5d] [%-8.5d] [%+.5d]\n", 42, 42, 42)
	for _, r := range []rune{'a', '\b', '\n', '\t', 0, 0x7F, '世', '\'', '\\', 0x10FFFF, 0xD800} {
		fmt.Printf("%q %+q ", r, r)
	}
	fmt.Println()
	fmt.Printf("%q %+q\n", "a\nb\x00", "a\nb世")
	fmt.Printf("%q\n", []rune{'a', '\n'})
	fmt.Printf("%\n")
	fmt.Printf("%!\n", 1)
}
