package main

import "fmt"

func two() (int, string) { return 1, "a" }

func three() (int, int, int) { return 7, 8, 9 }

func swap(v int) (int, int) { return v * 2, v }

func three2() (int, string) { return 3, "b" }

func main() {
	// n exists, and text is new.
	n := 0
	n, text := two()
	fmt.Println(n, text)

	// The first name is new, and the second name exists.
	count, text := three2()
	fmt.Println(count, text)

	// A blank stands between an existing name and a new name.
	a, _, b := three()
	fmt.Println(a, b)

	c := 0
	c, _, d := three()
	fmt.Println(c, d)

	// The right side reads the old value of an existing name.
	x := 5
	x, y := swap(x)
	fmt.Println(x, y)

	// A shadow inside a block keeps the outer name.
	{
		n, inner := two()
		fmt.Println(n, inner)
	}
	fmt.Println(n)
}
