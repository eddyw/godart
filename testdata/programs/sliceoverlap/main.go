// The program proves the order of a copy and of an append when the source
// range and the destination range share 1 backing array. Go moves the bytes,
// thus each byte of the source reaches the destination with its old value.
package main

import "fmt"

// copyForward writes a range over a range that starts later.
func copyForward() []int {
	a := []int{0, 1, 2, 3, 4, 5, 6, 7}
	copy(a[2:], a[:6])
	return a
}

// copyBack writes a range over a range that starts earlier.
func copyBack() []int {
	a := []int{0, 1, 2, 3, 4, 5, 6, 7}
	copy(a[:6], a[2:])
	return a
}

// appendOver adds a range of the same array behind the length of the slice.
func appendOver() []int {
	a := []int{0, 1, 2, 3, 4, 5, 6, 7}
	head := a[0:2:8]
	return append(head, a[1:5]...)
}

// appendSelf adds a slice to itself.
func appendSelf() []byte {
	b := make([]byte, 0, 16)
	b = append(b, "abcd"...)
	b = append(b, b...)
	return b
}

// appendString adds the bytes of a string that a slice of the same text gave.
func appendString() string {
	s := "hello, world"
	var b []byte
	b = append(b, s[7:]...)
	b = append(b, ' ')
	b = append(b, s[:5]...)
	return string(b)
}

// copyString writes the bytes of a string into a byte slice.
func copyString() string {
	b := make([]byte, 5)
	n := copy(b, "abcdefgh")
	return fmt.Sprint(n, string(b))
}

func main() {
	fmt.Println(copyForward())
	fmt.Println(copyBack())
	fmt.Println(appendOver())
	fmt.Println(string(appendSelf()))
	fmt.Println(appendString())
	fmt.Println(copyString())
}
