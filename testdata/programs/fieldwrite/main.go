// The program proves a field write whose value blocks. The base of the write
// is a struct location, and the write must land on that location, and not on
// a copy in the frame. The parser of the Go repository writes the fields of a
// named result this way, and each parameter list came back empty.
package main

import "fmt"

type rec struct {
	a, b string
}

type outer struct {
	inner rec
}

func fill(ch chan string) (r rec) {
	r.a = <-ch
	r.b = <-ch
	return
}

func main() {
	ch := make(chan string)
	go func() {
		ch <- "one"
		ch <- "two"
		ch <- "three"
		ch <- "four"
	}()
	fmt.Println(fill(ch))

	var o outer
	o.inner.a = <-ch
	list := []rec{{}, {}}
	list[1].b = <-ch
	fmt.Println(o, list)
}
