// The program proves that a composite literal names a type parameter. The
// constraint of the parameter is an interface, and the core type holds the
// shape.
package main

import "fmt"

func zeroArray[T ~[4]byte](in []byte) (T, error) {
	if len(in) != 4 {
		return T{}, fmt.Errorf("want 4 bytes, got %d", len(in))
	}
	var out T
	for i, v := range in {
		out[i] = v
	}
	return out, nil
}

func zeroSlice[T ~[]string]() T {
	return T{"a", "b"}
}

func zeroMap[T ~map[string]int]() T {
	return T{"one": 1}
}

type quad [4]byte
type names []string
type counts map[string]int

func main() {
	q, err := zeroArray[quad]([]byte{1, 2, 3, 4})
	fmt.Println(q, err)

	bad, err := zeroArray[quad]([]byte{1})
	fmt.Println(bad, err)

	fmt.Println(zeroSlice[names]())
	fmt.Println(zeroMap[counts]())
}
