package main

import "fmt"

func main() {
	constants()
	arithmetic()
	division()
	narrow()
	compare()
	verbs()
}

func constants() {
	c := 1 + 2i
	fmt.Println(c)
	fmt.Println(-c)
	fmt.Println(3i)
	fmt.Println(complex(1.5, -2.5))
	var zero complex128
	fmt.Println(zero)
}

func arithmetic() {
	a := complex(3.0, 4.0)
	b := complex(-1.0, 2.5)
	fmt.Println(a+b, a-b, a*b, a/b)
	fmt.Println(real(a), imag(a), real(b), imag(b))
}

// division proves the algorithm of Smith. A naive division overflows on a
// large part, and the corrections of C99 give an infinity and a zero.
func division() {
	x := 1e308
	y := 1e-308
	z := 0.0
	big := complex(x, x)
	fmt.Println(big / complex(1.5*x, 1.5*x))
	fmt.Println(complex(1, 1) / complex(z, z))
	fmt.Println(big / complex(y, y))
	fmt.Println(complex(1, 2) / complex(3, -4))
}

// narrow proves the rounding of complex64. Each operation rounds the 2 parts
// to 32 bits.
func narrow() {
	var a complex64 = complex(1.1, 2.2)
	var b complex64 = complex(0.3, -0.7)
	fmt.Println(a + b)
	fmt.Println(a * b)
	fmt.Println(complex128(a))
	c := complex128(3.3 + 4.4i)
	fmt.Println(complex64(c))
	fmt.Println(real(a), imag(b))
}

func compare() {
	a := 1 + 2i
	b := complex(1, 2)
	fmt.Println(a == b, a == 1+3i)
	counts := map[complex128]int{}
	counts[a] = 1
	counts[b]++
	counts[5i] = 3
	fmt.Println(counts[1+2i], counts[5i], len(counts))
}

func verbs() {
	c := complex(1.0, -2.0)
	fmt.Printf("%v %f %.2f %e %g\n", c, c, c, c, c)
	var s complex64 = complex(0.5, 1.25)
	fmt.Printf("%v %f\n", s, s)
	fmt.Printf("%v\n", []complex128{1 + 1i, 2 - 2i})
}
