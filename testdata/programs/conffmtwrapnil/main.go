// The witness of the verb %w with a nil error. Go writes the bad verb form.
//
// Go:   nothing: %!w(<nil>)
// gart: nothing: <nil>
package main

import "fmt"

func main() {
	fmt.Println(fmt.Errorf("nothing: %w", error(nil)))
	fmt.Println(fmt.Errorf("a text: %w", "not an error"))
}
