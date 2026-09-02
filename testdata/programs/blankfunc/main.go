package main

import "fmt"

type color int

const (
	red color = iota
	green
)

type size int

const (
	small size = iota
	large
)

// The 2 functions below carry the blank name, and stringer writes such a
// function for each type that it generates. Go calls neither 1.
func _() {
	var x [1]struct{}
	_ = x[red-0]
	_ = x[green-1]
}

func _() {
	var x [1]struct{}
	_ = x[small-0]
	_ = x[large-1]
}

func main() {
	fmt.Println(red, green, small, large)
}
