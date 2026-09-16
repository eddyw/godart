package main

import (
	"example.com/structconv/types"
	"fmt"
	"reflect"
)

type A struct {
	X [2]struct {
		N int `a`
	}
	Values []int
}
type B struct {
	X [2]struct {
		N int `b`
	}
	Values []int
}
type Named struct{ X int }

func (n Named) Number() int { return n.X }
func generic[T interface {
	~struct{ X int }
	Number() int
}](n int) T {
	v := T{X: n}
	fmt.Println(v.Number())
	return v
}

type RefA struct {
	P *struct {
		X int `a`
	}
	S []struct {
		X int `a`
	}
}
type RefB struct {
	P *struct {
		X int `b`
	}
	S []struct {
		X int `b`
	}
}

func references() {
	a := RefA{P: &struct {
		X int `a`
	}{1}, S: []struct {
		X int `a`
	}{{2}}}
	b := RefB(a)
	b.P.X = 3
	b.S[0].X = 4
	fmt.Println(a.P.X, a.S[0].X)
	fmt.Println(reflect.TypeOf(*a.P) == reflect.TypeOf(*b.P))
	fmt.Println(reflect.TypeOf(*a.P).Field(0).Tag, reflect.TypeOf(*b.P).Field(0).Tag)
	c := reflect.ValueOf(a).Convert(reflect.TypeOf(RefB{})).Interface().(RefB)
	c.P.X = 5
	c.S[0].X = 6
	fmt.Println(a.P.X, a.S[0].X)
}
func main() {
	references()
	a := A{Values: []int{3}}
	a.X[0].N = 2
	b := B(a)
	b.X[0].N = 9
	b.Values[0] = 4
	fmt.Println(a.X[0].N, b.X[0].N, a.Values[0])
	c := reflect.ValueOf(a).Convert(reflect.TypeOf(B{})).Interface().(B)
	c.X[0].N = 11
	fmt.Println(a.X[0].N, c.X[0].N)
	src := types.New()
	dst := types.Target(src)
	fmt.Println(src.Number(), dst.Number())
	fmt.Println(generic[Named](12))
}
