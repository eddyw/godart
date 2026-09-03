// The program asserts a method value and a MakeFunc value to a function type
// with a result. The witness of the method value is fixedbugs/issue27695 of the
// test suite of Go.
package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type stt struct {
	Data any
}

type my struct{ b byte }

func (m *my) Run(raw []byte) (stt, error) {
	return stt{Data: "hello " + string(raw)}, nil
}

func (m *my) Count(a, b int) int { return a + b }

func (m *my) Log(text string) { fmt.Println("log", text) }

func main() {
	// issue27695: a method value fits its own function type.
	f := reflect.ValueOf(&my{}).MethodByName("Run")
	method, ok := f.Interface().(func([]byte) (stt, error))
	fmt.Println(ok)
	s, e := method([]byte("world"))
	fmt.Println(s.Data, e)

	// A method value of 1 result and of no result.
	add, ok := reflect.ValueOf(&my{}).MethodByName("Count").
		Interface().(func(int, int) int)
	fmt.Println(ok, add(2, 3))
	log, ok := reflect.ValueOf(&my{}).MethodByName("Log").
		Interface().(func(string))
	fmt.Println(ok)
	log("one")

	// Value.Call still reads the same method value.
	out := f.Call([]reflect.Value{reflect.ValueOf([]byte("call"))})
	fmt.Println(out[0].Interface().(stt).Data, out[1].IsNil())

	// MakeFunc builds a value of a function type that the program names.
	var upper func(string) string
	ft := reflect.TypeOf(upper)
	mf := reflect.MakeFunc(ft, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(strings.ToUpper(in[0].String()))}
	})
	upper = mf.Interface().(func(string) string)
	fmt.Println(upper("abc"))

	// MakeFunc of a type with 2 results and of a type with none.
	var pair func(int) (int, error)
	pf := reflect.MakeFunc(reflect.TypeOf(pair), func(in []reflect.Value) []reflect.Value {
		n := int(in[0].Int())
		if n < 0 {
			return []reflect.Value{reflect.ValueOf(0), reflect.ValueOf(errors.New("negative"))}
		}
		return []reflect.Value{reflect.ValueOf(n * 2), reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
	})
	pair = pf.Interface().(func(int) (int, error))
	fmt.Println(pair(5))
	fmt.Println(pair(-1))

	var shout func(string)
	sf := reflect.MakeFunc(reflect.TypeOf(shout), func(in []reflect.Value) []reflect.Value {
		fmt.Println("shout", in[0].String())
		return nil
	})
	shout = sf.Interface().(func(string))
	shout("hey")

	// Call of a MakeFunc value reads the same body.
	res := pf.Call([]reflect.Value{reflect.ValueOf(7)})
	fmt.Println(res[0].Int(), res[1].IsNil())

	// MakeFunc of a variadic type.
	var join func(string, ...string) string
	jf := reflect.MakeFunc(reflect.TypeOf(join), func(in []reflect.Value) []reflect.Value {
		parts := make([]string, in[1].Len())
		for i := range parts {
			parts[i] = in[1].Index(i).String()
		}
		return []reflect.Value{reflect.ValueOf(strings.Join(parts, in[0].String()))}
	})
	join = jf.Interface().(func(string, ...string) string)
	fmt.Println(join("-", "a", "b", "c"))

	// CallSlice assigns the last argument to the variadic parameter.
	items := []string{"x", "y"}
	cs := jf.CallSlice([]reflect.Value{reflect.ValueOf("+"), reflect.ValueOf(items)})
	fmt.Println(cs[0].String())
}
