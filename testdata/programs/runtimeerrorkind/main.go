package main

import (
	"fmt"
	"reflect"
	"runtime"
)

func observe(name string, fail func()) {
	defer func() {
		e := recover()
		v := reflect.ValueOf(e)
		t := v.Type()
		_, rt := e.(runtime.Error)
		fmt.Println(name, t, t.Name(), t.PkgPath(), t.Kind(), t.Size(), t.Align(), t.Comparable(), rt)
		fmt.Println("error", e)
		if t.Kind() == reflect.String {
			fmt.Println("underlying", v.String(), v.Len(), v.Convert(reflect.TypeOf("")).Interface())
		} else {
			for i := 0; i < t.NumField(); i++ {
				f, value := t.Field(i), v.Field(i)
				fmt.Println("field", f.Name, f.Type, f.PkgPath, f.Offset)
				switch value.Kind() {
				case reflect.Int, reflect.Int64:
					fmt.Println("value", value.Int())
				case reflect.Uint8:
					fmt.Println("value", value.Uint())
				case reflect.Bool:
					// Go can encode a proven nonnegative index as unsigned.
					if v.Field(0).Int() >= 0 {
						fmt.Println("value", "nonnegative index")
					} else {
						fmt.Println("value", value.Bool())
					}
				}
			}
		}
		z := reflect.Zero(t).Interface().(error)
		fmt.Println("zero", z.Error(), e == e, e == z)
		copy := reflect.New(t).Elem()
		copy.Set(v)
		fmt.Println("copy", copy.Interface().(error).Error(), copy.Interface() == e)
		ptr := reflect.New(t)
		ptr.Elem().Set(v)
		fmt.Println("pointer", ptr.Interface().(error).Error(), ptr.Type().NumMethod())
		method := v.MethodByName("Error").Interface().(func() string)
		fmt.Println("direct method", method())
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			fmt.Println("method", m.Name, m.Type, m.PkgPath)
			result := v.Method(i).Call(nil)
			fmt.Println("result count", len(result))
			for _, result := range result {
				fmt.Println("result", result.Interface())
			}
		}
	}()
	fail()
	panic("operation did not panic")
}

func main() {
	observe("index", func() { v := []int{1}; n := 2; fmt.Println(v[n]) })
	observe("negative index", func() { v := []int{1}; n := -1; fmt.Println(v[n]) })
	observe("unsigned index", func() { v := []int{1}; n := ^uint(0); fmt.Println(v[n]) })
	observe("slice length", func() { n := 3; fmt.Println("hi"[:n]) })
	observe("slice capacity", func() { v := make([]int, 2, 3); n := 4; fmt.Println(v[:n]) })
	observe("slice order", func() { v := make([]int, 2, 3); n := 2; fmt.Println(v[n:1]) })
	observe("triple array", func() { var v [3]int; n := 4; fmt.Println(v[:1:n]) })
	observe("triple capacity", func() { v := make([]int, 2, 3); n := 4; fmt.Println(v[:1:n]) })
	observe("triple high", func() { v := make([]int, 2, 3); n := 2; fmt.Println(v[:n:1]) })
	observe("triple low", func() { v := make([]int, 2, 3); n := 2; fmt.Println(v[n:1:3]) })
	observe("divide", func() { var n int; fmt.Println(1 / n) })
	observe("shift", func() { n := -1; fmt.Println(1 << n) })
	observe("nil pointer", func() { var p *int; fmt.Println(*p) })
	observe("compare", func() { var a any = []int{}; fmt.Println(a == a) })
	observe("map write", func() { m := map[any]int{}; m[[]int{}] = 1 })
	observe("nil map", func() { var m map[int]int; m[1] = 1 })
	observe("closed send", func() { c := make(chan int); close(c); c <- 1 })
	observe("nil close", func() { var c chan int; close(c) })
	observe("closed close", func() { c := make(chan int); close(c); close(c) })
	observe("negative channel", func() { n := -1; _ = make(chan int, n) })
	observe("slice conversion", func() { v := []int{}; fmt.Println([1]int(v)) })
}
