// The program proves the space flag of the verb x. Go puts 1 space between
// each pair of digits of a byte array, of a byte slice, and of a string. The
// flag # then writes the prefix of the base on each byte.
package main

import "fmt"

func main() {
	b := []byte{0x4d, 0x5a, 0x00, 0x01}
	fmt.Printf("[% x]\n", b)
	fmt.Printf("[% X]\n", b)
	fmt.Printf("[%x]\n", b)
	var a [3]byte = [3]byte{1, 2, 255}
	fmt.Printf("[% x]\n", a)
	fmt.Printf("[% x]\n", "MZ\x00")
	fmt.Printf("[% x]\n", 255)
	fmt.Printf("[% d]\n", []int{1, 2, 3})
	fmt.Printf("[%#x]\n", b)
	fmt.Printf("[% #x]\n", b)
}
