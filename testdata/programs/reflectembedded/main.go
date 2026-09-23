package main

import (
	"example.com/reflectembedded/foreign"
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
	"time"
)

type Number int

func (n Number) Get() int { return int(n) }
func (Number) hidden()    {}

type Getter interface{ Get() int }
type Secret interface{ hidden() }
type ForeignSecret interface{ foreign.Hidden }
type Outer struct{ foreign.Base }
type Async int

func (n Async) Pair(x int) (int, int) { time.Sleep(time.Nanosecond); return int(n) + x, int(n) + x + 1 }

type NilAware struct{ N int }

func (n *NilAware) Get() int {
	if n == nil {
		return -1
	}
	return n.N
}

type Direct struct {
	Z [0]byte
	P *int
}

func makeValue(name string, t reflect.Type, x reflect.Value) reflect.Value {
	st := reflect.StructOf([]reflect.StructField{{Name: name, Type: t, Anonymous: true}})
	v := reflect.New(st).Elem()
	if x.IsValid() {
		v.Field(0).Set(x)
	}
	return v
}
func panicText(f func()) { defer func() { fmt.Println(recover()) }(); f() }
func main() {
	n := Number(42)
	for _, x := range []any{n, &n} {
		v := makeValue("Number", reflect.TypeOf(x), reflect.ValueOf(x))
		t := v.Type()
		fmt.Println(t.NumMethod(), reflect.PointerTo(t).NumMethod(), v.Interface().(Getter).Get())
		_, secret := v.Interface().(Secret)
		fmt.Println(secret)
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			fmt.Println(m.Name, m.PkgPath, m.Type, m.Func.IsValid())
		}
		fmt.Println(v.MethodByName("Get").Call(nil)[0].Int())
		m, _ := t.MethodByName("Get")
		fmt.Println(m.Func.Call([]reflect.Value{v})[0].Int())
	}
	nilv := makeValue("NilAware", reflect.TypeOf((*NilAware)(nil)), reflect.Value{})
	fmt.Println(nilv.Interface().(Getter).Get())
	iv := makeValue("Getter", reflect.TypeFor[Getter](), reflect.ValueOf(n))
	fmt.Println(iv.Type().NumMethod(), iv.Type().Implements(reflect.TypeFor[Getter]()))
	panicText(func() { iv.Interface().(Getter).Get() })
	panicText(func() { makeValue("Secret", reflect.TypeFor[Secret](), reflect.Value{}) })
	fmt.Println(reflect.TypeFor[Secret]().Method(0).PkgPath)
	panicText(func() { makeValue("Hidden", reflect.TypeFor[foreign.Hidden](), reflect.Value{}) })
	panicText(func() { makeValue("ForeignSecret", reflect.TypeFor[ForeignSecret](), reflect.Value{}) })
	panicText(func() { makeValue("Hidden", reflect.TypeFor[interface{ foreign.Hidden }](), reflect.Value{}) })
	panicText(func() { makeValue("Outer", reflect.TypeOf(Outer{}), reflect.Value{}) })

	panicText(func() { makeValue("Number", reflect.TypeOf((**Number)(nil)), reflect.Value{}) })
	panicText(func() { makeValue("Getter", reflect.TypeOf((*Getter)(nil)), reflect.Value{}) })
	panicText(func() {
		reflect.StructOf([]reflect.StructField{{Name: "X", Type: reflect.TypeOf(0)}, {Name: "Number", Type: reflect.TypeOf(n), Anonymous: true}})
	})
	panicText(func() {
		reflect.StructOf([]reflect.StructField{{Name: "Direct", Type: reflect.TypeOf(Direct{}), Anonymous: true}, {Name: "Empty", Type: reflect.TypeOf(struct{}{})}})
	})
	synctest.Test(new(testing.T), func(*testing.T) {
		v := makeValue("Async", reflect.TypeOf(Async(40)), reflect.ValueOf(Async(40)))
		x, y := v.Interface().(interface{ Pair(int) (int, int) }).Pair(1)
		fmt.Println(x, y)
		bound := v.MethodByName("Pair")
		values := bound.Call([]reflect.Value{reflect.ValueOf(2)})
		fmt.Println(values[0].Int(), values[1].Int())
		f := bound.Interface().(func(int) (int, int))
		x, y = f(3)
		fmt.Println(x, y)
		m, _ := v.Type().MethodByName("Pair")
		values = m.Func.Call([]reflect.Value{v, reflect.ValueOf(4)})
		fmt.Println(values[0].Int(), values[1].Int())
	})
}
