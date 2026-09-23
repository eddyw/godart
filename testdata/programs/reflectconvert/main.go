package main

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

type NumberFunc func(int) int

func main() {
	fn := reflect.ValueOf(func(x int) int { return x + 3 }).Convert(reflect.TypeFor[NumberFunc]()).Interface().(NumberFunc)
	fmt.Println(fn(5))
	slice := []int{1, 2, 3, 4}
	window := slice[1:]
	array := reflect.ValueOf(window).Convert(reflect.TypeFor[*[2]int]()).Interface().(*[2]int)
	array[0] = 7
	fmt.Println(slice, array == (*[2]int)(window), &array[1] == &slice[2])
	var nilSlice []int
	fmt.Println(reflect.ValueOf(nilSlice).Convert(reflect.TypeFor[*[0]int]()).IsNil(), reflect.ValueOf([]int{}).Convert(reflect.TypeFor[*[0]int]()).IsNil())
	p := new([0]int)
	*p = [0]int{}
	fmt.Println(len(*p))
	value := struct{ X int }{17}
	converted := reflect.ValueOf(&value).Elem().Convert(reflect.TypeOf(value))
	value.X = 23
	fmt.Println(converted.Field(0).Int(), value.X)
	boxed := reflect.ValueOf(&value).Elem().Convert(reflect.TypeFor[any]())
	value.X = 31
	fmt.Println(reflect.ValueOf(boxed.Interface()).Field(0).Int(), value.X)
	var reader io.Reader = new(bytes.Buffer)
	func() {
		defer func() { fmt.Println("interface conversion rejected", recover() != nil) }()
		_ = reflect.ValueOf(&reader).Elem().Convert(reflect.TypeFor[io.Writer]())
	}()
}
