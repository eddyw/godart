package main

import (
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
	"time"
)

type Reader struct{ C chan int }

func (r Reader) Read() int { return <-r.C }
func main() {
	synctest.Test(new(testing.T), func(t *testing.T) {
		c := make(chan int)
		conditionalDefer(true)
		f := func(a int) (int, string) { time.Sleep(time.Nanosecond); return a + <-c, "done" }
		go func() { c <- 3 }()
		out := reflect.ValueOf(f).Call([]reflect.Value{reflect.ValueOf(2)})
		fmt.Println(out[0].Int(), out[1].String())
		variadic := func(a int, rest ...int) int { time.Sleep(time.Nanosecond); return a + rest[0] }
		out = reflect.ValueOf(variadic).CallSlice([]reflect.Value{reflect.ValueOf(4), reflect.ValueOf([]int{5})})
		fmt.Println(out[0].Int())
		out = reflect.ValueOf(variadic).Call([]reflect.Value{reflect.ValueOf(6), reflect.ValueOf(7)})
		fmt.Println(out[0].Int())
		typ := reflect.TypeOf((func(int) (int, string))(nil))
		made := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
			time.Sleep(time.Nanosecond)
			return []reflect.Value{reflect.ValueOf(int(args[0].Int()) + 1), reflect.ValueOf("made")}
		})
		g := made.Interface().(func(int) (int, string))
		fmt.Println(g(8))
		out = made.Call([]reflect.Value{reflect.ValueOf(9)})
		fmt.Println(out[0].Int(), out[1].String())
		go func() { c <- 11 }()
		method := reflect.ValueOf(Reader{c}).MethodByName("Read").Interface().(func() int)
		fmt.Println(method())
		func() {
			defer func() { fmt.Println(recover()) }()
			reflect.ValueOf(func() { time.Sleep(time.Nanosecond); panic("callback") }).Call(nil)
		}()
	})
}

func conditionalDefer(on bool) {
	value := 1
	if on {
		original := value
		defer func() { fmt.Println("deferred", value, original) }()
		value = 2
	}
	time.Sleep(time.Nanosecond)
}
