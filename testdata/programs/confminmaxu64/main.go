// The witness of the min and the max builtin on a uint64. Go compares the 2
// values as unsigned, and the Dart output compares them as signed.
//
// Go:   min 0 / max 18446744073709551615
// gart: min 18446744073709551615 / max 0
package main

import "fmt"

func main() {
	var u, v uint64 = 0, 1<<64 - 1
	fmt.Println("min", min(u, v))
	fmt.Println("max", max(u, v))
}
