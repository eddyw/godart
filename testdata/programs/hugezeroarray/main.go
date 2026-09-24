package main

import (
	"fmt"
	"reflect"
)

func invalid(v reflect.Value, i int) {
	defer func() { e := recover(); fmt.Printf("%T %v\n", e, e) }()
	v.Index(i)
	panic("index did not panic")
}

func main() {
	for _, n := range []int{0, 1, 1<<53 + 1, int(^uint(0) >> 1)} {
		t := reflect.ArrayOf(n, reflect.TypeOf(struct{}{}))
		v := reflect.Zero(t)
		fmt.Println(t.Len(), v.Len(), v.Cap(), t.Size())
		if n > 0 {
			fmt.Println(v.Index(n - 1).Interface())
			w := reflect.New(t).Elem()
			f := w.Index(n - 1)
			f.Set(reflect.ValueOf(struct{}{}))
			fmt.Println(f.Interface(), f.Addr().Elem().Interface(), f.CanSet())
		}
		invalid(v, -1)
		invalid(v, n)
	}
	for _, v := range []reflect.Value{reflect.ValueOf([2]int{1, 2}), reflect.ValueOf([]int{1, 2}), reflect.ValueOf("hi")} {
		fmt.Println(v.Index(1).Interface())
		invalid(v, -1)
		invalid(v, 2)
	}
}
