package main

import (
	"example.com/reflectinterfaceidentity/foreign"
	"fmt"
	"reflect"
)

func first() []reflect.Type {
	type T int
	type Embedded interface{ F(T) T }
	return []reflect.Type{reflect.TypeFor[interface{ F(T) T }](), reflect.TypeFor[interface{ Embedded }](), reflect.TypeFor[interface{ G() interface{ F(T) T } }](), reflect.TypeFor[struct{ I interface{ F(T) T } }]()}
}
func second() []reflect.Type {
	type T int
	return []reflect.Type{reflect.TypeFor[interface{ F(T) T }](), reflect.TypeFor[interface{ F(T) T }](), reflect.TypeFor[interface{ G() interface{ F(T) T } }](), reflect.TypeFor[struct{ I interface{ F(T) T } }]()}
}

type Alias = byte
type Box[T any] struct{ X T }
type Input = struct {
	X []byte
	F func(byte) byte
}
type Output = Box[Alias]
type Worker struct{}

func (Worker) Apply(in Input) Output { return Output{in.F(in.X[0])} }

type Transformer interface {
	Apply(struct {
		X []uint8
		F func(uint8) uint8
	}) Box[uint8]
}
type Promoted struct{ Transformer }
type Generic[T any] interface{ Echo(func(T) T, T) T }
type Carrier[T any] struct{}

func (Carrier[T]) Echo(f func(T) T, value T) T { return f(value) }
func genericCall[T any](value T) {
	var x Generic[T] = Carrier[T]{}
	f := Generic[T].Echo
	fmt.Println(x.Echo(func(x T) T { return x }, value), f(x, func(x T) T { return x }, value))
}
func genericLocal[T any]() []reflect.Type {
	type Local struct{ X T }
	type Embedded interface{ F(Local) Local }
	return []reflect.Type{
		reflect.TypeFor[interface{ F(Local) Local }](),
		reflect.TypeFor[interface{ Embedded }](),
		reflect.TypeFor[func(interface{ F(Local) Local }) interface{ F(Local) Local }](),
	}
}
func interfaceNames[T any]() {
	typ := reflect.TypeFor[interface {
		hidden(T) T
		Exported(T) T
	}]()
	fmt.Println(typ, reflect.SliceOf(typ), reflect.FuncOf([]reflect.Type{typ}, []reflect.Type{typ}, false))
}

func main() {
	var value any = Worker{}
	x, ok := value.(Transformer)
	fmt.Println("alias assertion", ok)
	in := Input{[]byte{4}, func(b byte) byte { return b + 1 }}
	fmt.Println(x.Apply(in).X, Transformer.Apply(x, in).X, Promoted.Apply(Promoted{x}, in).X)
	genericCall(17)
	genericCall("text")
	g1, g2 := genericLocal[int](), genericLocal[string]()
	for i := range g1 {
		fmt.Println("generic local", g1[i] == g2[i], g1[i].AssignableTo(g2[i]), g1[i] == genericLocal[int]()[i])
	}
	fmt.Println("generic embedded", g1[0] == g1[1])

	a, b := first(), second()
	local, other := reflect.TypeFor[interface{ hidden() }](), foreign.Type()
	fmt.Println(local == other, local.AssignableTo(other), local.Implements(other))
	fmt.Println(local, other, reflect.TypeFor[interface{ foreign.Mixed }]())
	fmt.Println(reflect.TypeFor[interface {
		foreign.Mixed
		hidden()
	}]())
	interfaceNames[int]()
	interfaceNames[string]()
	fmt.Println(a[0] == a[1], a[0] == first()[0])
	for i := range a {
		fmt.Println(a[i] == b[i], a[i].AssignableTo(b[i]), a[i].ConvertibleTo(b[i]))
	}
	fmt.Println(a[0].Implements(b[0]), a[0].Method(0).Type.In(0) == b[0].Method(0).Type.In(0))
}
