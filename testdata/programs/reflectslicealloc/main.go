package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type hiddenKind struct{ reflect.Type }

func (hiddenKind) Kind() reflect.Kind { return reflect.Int }

func attempt(t reflect.Type, n, c int) {
	defer func() {
		value := recover()
		_, runtimeError := value.(runtime.Error)
		fmt.Printf("%T: %v, runtime=%v\n", value, value, runtimeError)
	}()
	v := reflect.MakeSlice(t, n, c)
	fmt.Println(v.Type(), v.Len(), v.Cap(), v.IsNil())
}

func main() {
	t := reflect.TypeOf([]int{})
	attempt(nil, 0, 0)
	attempt(t, 2, 3)
	attempt(t, 0, 0)
	attempt(t, -1, -1)
	attempt(t, 0, -1)
	attempt(t, 3, 2)
	attempt(reflect.TypeOf(0), -1, -1)
	attempt(t, 0, 1<<49)
	attempt(reflect.SliceOf(reflect.ArrayOf(1<<30, reflect.TypeOf(byte(0)))), 0, 1<<35)
	attempt(reflect.SliceOf(reflect.ArrayOf(1<<62, reflect.TypeOf(uint16(0)))), 0, 2)
	attempt(reflect.TypeOf([]struct{}{}), 1<<53+1, 1<<53+1)
	attempt(hiddenKind{t}, 1, 2)
}
