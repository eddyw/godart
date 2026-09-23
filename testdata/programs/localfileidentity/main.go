package main

import (
	"fmt"
	"reflect"
)

func main() {
	a, x := first()
	b, y := second()
	fmt.Println(reflect.TypeOf(a) == reflect.TypeOf(b), a == b)
	fmt.Println(reflect.TypeOf(x) == reflect.TypeOf(y), x == y)
}
