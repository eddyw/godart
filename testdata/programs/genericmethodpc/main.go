package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

type G[T any] struct{ V T }

func (g G[T]) M() T  { return g.V }
func (g *G[T]) P() T { return g.V }

type Outer[T any] struct{ G[T] }
type Alias = int

func show(f reflect.Value) {
	pc := f.Pointer()
	r := runtime.FuncForPC(pc)
	if r == nil {
		panic("missing method metadata")
	}
	file, line := r.FileLine(pc)
	fmt.Println(r.Name(), filepath.Base(file), line, r.Entry() == pc)
}

func methods[T any]() (func(G[T]) T, func(*G[T]) T) {
	return G[T].M, (*G[T]).M
}

func nested[T any]() func() func(G[T]) T {
	return func() func(G[T]) T { return G[T].M }
}

func mixed[T any]() (func() T, func(G[T]) T) {
	G[T].M(G[T]{})
	before := func() T { var zero T; return zero }
	return before, G[T].M
}

type Constraint interface{ M() int }
type Fixed int

func (v Fixed) M() int                       { return int(v) }
func constrained[T Constraint]() func(T) int { return T.M }

func main() {
	show(reflect.ValueOf(G[int].M))
	show(reflect.ValueOf((*G[int]).M))
	show(reflect.ValueOf((*G[int]).P))
	t := reflect.TypeOf(G[int]{})
	m, _ := t.MethodByName("M")
	show(m.Func)
	fmt.Println(m.Func.Pointer() == reflect.ValueOf(G[int].M).Pointer())
	p := reflect.PointerTo(t)
	for i := 0; i < p.NumMethod(); i++ {
		show(p.Method(i).Func)
	}
	m, _ = p.MethodByName("M")
	fmt.Println(m.Func.Pointer() == reflect.ValueOf((*G[int]).M).Pointer())
	m, _ = p.MethodByName("P")
	fmt.Println(m.Func.Pointer() == reflect.ValueOf((*G[int]).P).Pointer())
	show(reflect.ValueOf(G[string].M))
	fmt.Println(reflect.ValueOf(G[int].M).Pointer() != reflect.ValueOf(G[string].M).Pointer())
	fmt.Println(reflect.ValueOf(G[int].M).Pointer() == reflect.ValueOf(G[Alias].M).Pointer())
	value, pointer := methods[int]()
	show(reflect.ValueOf(value))
	show(reflect.ValueOf(pointer))
	fmt.Println(reflect.ValueOf(value).Pointer() == reflect.ValueOf(G[int].M).Pointer())
	fmt.Println(reflect.ValueOf(pointer).Pointer() == reflect.ValueOf((*G[int]).M).Pointer())
	outer := nested[int]()
	show(reflect.ValueOf(outer))
	show(reflect.ValueOf(outer()))
	before, after := mixed[int]()
	show(reflect.ValueOf(before))
	show(reflect.ValueOf(after))
	fmt.Println(before(), after(G[int]{9}))
	constraint := constrained[Fixed]()
	show(reflect.ValueOf(constraint))
	fmt.Println(constraint(Fixed(8)))
	v := G[int]{17}
	fmt.Println(value(v), pointer(&v))
	other, _ := methods[string]()
	fmt.Println(other(G[string]{"text"}))
	fmt.Println(reflect.ValueOf(value).Pointer() != reflect.ValueOf(other).Pointer())
	for _, item := range []any{G[uint8]{23}, Outer[bool]{G[bool]{true}}} {
		t := reflect.PointerTo(reflect.TypeOf(item))
		m, _ := t.MethodByName("M")
		show(m.Func)
		p := reflect.New(t.Elem())
		p.Elem().Set(reflect.ValueOf(item))
		fmt.Println(m.Func.Call([]reflect.Value{p})[0].Interface())
		again, _ := t.MethodByName("M")
		fmt.Println(m.Func.Pointer() == again.Func.Pointer())
	}
}
