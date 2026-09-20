package main

import (
	"fmt"
	"reflect"
)

func outcome(name string, f func()) {
	defer func() {
		if v := recover(); v != nil {
			fmt.Println(name, v)
		}
	}()
	f()
	fmt.Println(name, "returned")
}

type sealed interface{ hidden() }
type item struct{}

func (item) hidden() {}

func main() {
	var secret sealed = item{}
	method := reflect.ValueOf(&secret).Elem().Method(0)
	privateSlice := reflect.ValueOf(struct{ values []int }{[]int{1}}).Field(0)

	fixed := reflect.ValueOf(func(int) {})
	variadic := reflect.ValueOf(func(string, ...int) {})
	var absent func(int)
	var absentVariadic func(...int)
	hidden := reflect.ValueOf(struct{ value int }{1}).Field(0)
	integer := reflect.ValueOf(1)
	text := reflect.ValueOf("x")
	list := reflect.ValueOf([]int{1, 2})
	for _, c := range []struct {
		name  string
		fn    reflect.Value
		args  []reflect.Value
		slice bool
	}{
		{"nil", reflect.ValueOf(absent), nil, false},
		{"nil slice", reflect.ValueOf(absentVariadic), nil, true},
		{"few", fixed, nil, false},
		{"many", fixed, []reflect.Value{integer, integer}, false},
		{"zero", fixed, []reflect.Value{{}}, false},
		{"type", fixed, []reflect.Value{text}, false},
		{"private", fixed, []reflect.Value{hidden}, false},
		{"slice fixed", fixed, []reflect.Value{integer}, true},
		{"slice few", variadic, []reflect.Value{text}, true},
		{"slice many", variadic, []reflect.Value{text, list, list}, true},
		{"slice zero", variadic, []reflect.Value{text, {}}, true},
		{"slice type", variadic, []reflect.Value{text, integer}, true},
		{"variadic few", variadic, nil, false},
		{"variadic type", variadic, []reflect.Value{text, text}, false},
		{"variadic private", variadic, []reflect.Value{text, hidden}, false},
		{"private method", reflect.ValueOf(func(func()) {}), []reflect.Value{method}, false},
		{"variadic private method", reflect.ValueOf(func(...func()) {}), []reflect.Value{method}, false},
		{"slice private", variadic, []reflect.Value{text, privateSlice}, true},
		{"invalid before type", variadic, []reflect.Value{integer, {}}, false},
		{"variadic type before private", reflect.ValueOf(func(int, ...string) {}), []reflect.Value{hidden, integer}, false},
		{"valid fixed", fixed, []reflect.Value{integer}, false},
		{"valid variadic empty", variadic, []reflect.Value{text}, false},
		{"valid variadic", variadic, []reflect.Value{text, integer, integer}, false},
		{"valid slice", variadic, []reflect.Value{text, list}, true},
	} {
		outcome(c.name, func() {
			if c.slice {
				c.fn.CallSlice(c.args)
			} else {
				c.fn.Call(c.args)
			}
		})
	}
}
