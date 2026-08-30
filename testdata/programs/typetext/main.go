package main

import (
	"fmt"
	"reflect"
)

type named struct{ q int }

type reader interface{ Read() int }

func (n named) Read() int { return n.q }

func main() {
	vals := []any{
		struct{}{},
		struct{ a int }{},
		struct {
			a int
			b string
		}{},
		struct {
			A int `json:"a"`
			B string
		}{},
		struct{ named }{},
		[]struct{ x int }{},
		map[string]struct{ y int }{},
		&struct{ z int }{},
		[2]struct{ w int }{},
		struct{ f func(int) string }{},
		struct{ g func(...int) (string, error) }{},
		struct{ c chan struct{} }{},
		struct{ r <-chan int }{},
		struct{ s chan<- int }{},
		struct{ n named }{},
		struct{ i any }{},
		struct{ p reader }{},
		make(chan struct{}),
		named{},
		[]any{},
		map[string][]*named{},
	}
	for _, v := range vals {
		t := reflect.TypeOf(v)
		fmt.Printf("%-42q kind=%-9s name=%q\n", t.String(), t.Kind(), t.Name())
	}

	// The tag reaches the descriptor of the type.
	tagged := reflect.TypeOf(struct {
		A int    `json:"a" db:"col"`
		B string `json:"b,omitempty"`
	}{})
	for i := 0; i < tagged.NumField(); i++ {
		f := tagged.Field(i)
		fmt.Printf("%s %q %q %q\n", f.Name, f.Tag, f.Tag.Get("json"), f.Tag.Get("db"))
	}
}
