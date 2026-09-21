package main

import (
	"example.com/linkfunc/api"
	"example.com/linkfunc/impl"
	"fmt"
	"reflect"
)

func main() {
	fmt.Println(api.Call(5), api.Wait(7), api.Saved(9), api.Local(10), impl.Initialized)
	fmt.Println(reflect.ValueOf(api.Saved).Pointer() == reflect.ValueOf(impl.Function()).Pointer())
}
