package main

import (
	"errors"
	"fmt"
)

type shape interface{ area() int }

type box struct{ n int }

func (b box) area() int { return b.n }

func pair() (float64, error) { return 2.5, errors.New("x") }

func shaped() (box, int) { return box{3}, 7 }

func boxed() (any, error) { return pair() }

func viaShape() (shape, int) { return shaped() }

func take(v any, err error) string { return fmt.Sprint(v, "/", err) }

func main() {
	v, err := boxed()
	fmt.Println(v, err)

	s, n := viaShape()
	fmt.Println(s.area(), n)

	var a any
	var e error
	a, e = pair()
	fmt.Println(a, e)

	fmt.Println(take(pair()))

	list := []any{}
	x, _ := pair()
	list = append(list, x)
	fmt.Println(list)
}
