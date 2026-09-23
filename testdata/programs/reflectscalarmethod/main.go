package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type number float64

var calls int

func (n number) Zero() {
	runtime.Gosched()
	calls++
}

func (n number) One() int {
	runtime.Gosched()
	return int(n)
}

func (n number) Pair(prefix string) ([]byte, error) {
	runtime.Gosched()
	return []byte(fmt.Sprintf("%s%d", prefix, int(n))), nil
}

func (n number) Triple() (int, string, bool) {
	runtime.Gosched()
	return int(n), "ready", true
}

func main() {
	value := reflect.New(reflect.TypeFor[number]())
	value.Elem().SetFloat(42)
	methods := value.Interface().(interface {
		Zero()
		One() int
		Pair(string) ([]byte, error)
		Triple() (int, string, bool)
	})
	methods.Zero()
	fmt.Println(calls)
	fmt.Println(methods.One())
	bytes, err := methods.Pair("value:")
	fmt.Println(string(bytes), err)
	fmt.Println(methods.Triple())
	results := value.MethodByName("Pair").Call([]reflect.Value{reflect.ValueOf("reflect:")})
	fmt.Println(string(results[0].Bytes()), results[1].IsNil())
}
