// The program proves that a type parameter with an unnamed struct core takes an
// unnamed struct type.
package main

import "fmt"

func first[T ~struct{ a, b int }](v T) T {
	return v
}

func main() {
	v := struct{ a, b int }{1, 2}
	fmt.Println(first(v))
}
