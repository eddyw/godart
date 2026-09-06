// The witness of a width on a compound value. Go gives the width to each
// element of a slice, of an array, of a map, and of a struct.
//
// Go:   [1        2       ]
// gart: [1 2]
package main

import "fmt"

func main() {
	fmt.Printf("[%-8v]\n", []int{1, 2})
	fmt.Printf("[%8v]\n", []int{1, 2})
	fmt.Printf("[%-8v]\n", [2]int{1, 2})
	fmt.Printf("[%-8v]\n", struct{ A, B int }{1, 2})
	fmt.Printf("[%-8s]\n", []string{"x", "y"})
}
