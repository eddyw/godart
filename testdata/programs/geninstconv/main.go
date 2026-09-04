package main

import "fmt"

type box[T any] struct {
	v []T
	n bool
}

type enc struct{ a int }
type dec struct{ b int }

type encBox = box[enc]

type opt encBox

func (*opt) mark() {}

type marker interface{ mark() }

func main() {
	b := &box[enc]{v: []enc{{a: 1}}, n: true}
	o := (*opt)(b)
	o.mark()
	var m marker = o
	fmt.Println(len(b.v), b.n, m != nil)
	back := (*encBox)(o)
	fmt.Println(back == b)
	var any1 any = b
	_, ok := any1.(marker)
	fmt.Println("box is marker:", ok)
}
