package main

import (
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
	"time"
)

type Number int
type Wide int
type DynamicWide Wide

func (w Wide) Many(a0 int, a1 int, a2 int, a3 int, a4 int, a5 int, a6 int, a7 int, a8 int, a9 int) (int, int, int, int, int, int, int, int, int) {
	time.Sleep(time.Nanosecond)
	v := int(w) + a0 + a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9
	return v + 0, v + 1, v + 2, v + 3, v + 4, v + 5, v + 6, v + 7, v + 8
}
func (w DynamicWide) Many(a0 int, a1 int, a2 int, a3 int, a4 int, a5 int, a6 int, a7 int, a8 int, a9 int) (int, int, int, int, int, int, int, int, int) {
	return Wide(w).Many(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9)
}
func report(name string, values []reflect.Value) {
	fmt.Print(name)
	for _, v := range values {
		fmt.Print(" ", v.Interface())
	}
	fmt.Println()
}
func panicText(f func()) { defer func() { fmt.Println(recover()) }(); f() }
func main() {
	synctest.Test(new(testing.T), func(*testing.T) {
		typ := reflect.TypeFor[Number]()
		for _, count := range []int{0, 7, 51, 128} {
			types := make([]reflect.Type, count)
			args := make([]reflect.Value, count)
			for i := range types {
				types[i] = typ
				args[i] = reflect.ValueOf(Number(i))
			}
			f := reflect.MakeFunc(reflect.FuncOf(types, nil, false), func(in []reflect.Value) []reflect.Value { fmt.Println("width", len(in)); return nil })
			f.Call(args)
		}
		maximumResults := make([]reflect.Type, 128)
		for i := range maximumResults {
			maximumResults[i] = typ
		}
		maximum := reflect.MakeFunc(reflect.FuncOf(nil, maximumResults, false), func([]reflect.Value) []reflect.Value {
			time.Sleep(time.Nanosecond)
			values := make([]reflect.Value, 128)
			for i := range values {
				values[i] = reflect.ValueOf(Number(i))
			}
			return values
		})
		report("maximum results", maximum.Call(nil))
		many := make([]reflect.Type, 129)
		for i := range many {
			many[i] = typ
		}
		panicText(func() { reflect.FuncOf(many, nil, false) })
		panicText(func() { reflect.FuncOf(many[:128], many[:1], false) })
		panicText(func() { reflect.FuncOf(nil, many, false) })
		panicText(func() { reflect.FuncOf(many, nil, true) })
		params := make([]reflect.Type, 10)
		args := make([]reflect.Value, 10)
		ints := make([]reflect.Value, 10)
		for i := range params {
			params[i] = typ
			args[i] = reflect.ValueOf(Number(i))
			ints[i] = reflect.ValueOf(i)
		}
		f := reflect.MakeFunc(reflect.FuncOf(params, []reflect.Type{typ}, false), func(in []reflect.Value) []reflect.Value {
			time.Sleep(time.Nanosecond)
			sum := Number(0)
			for _, v := range in {
				sum += Number(v.Int())
			}
			return []reflect.Value{reflect.ValueOf(sum)}
		})
		report("arguments", f.Call(args))
		box := reflect.New(f.Type()).Elem()
		box.Set(f)
		report("copy", reflect.ValueOf(box.Interface()).Call(args))
		results := make([]reflect.Type, 9)
		for i := range results {
			results[i] = typ
		}
		f = reflect.MakeFunc(reflect.FuncOf(nil, results, false), func([]reflect.Value) []reflect.Value {
			time.Sleep(time.Nanosecond)
			out := make([]reflect.Value, 9)
			for i := range out {
				out[i] = reflect.ValueOf(Number(i))
			}
			return out
		})
		report("results", f.Call(nil))
		g := reflect.ValueOf(func() (int, int, int, int, int, int, int, int, int) {
			time.Sleep(time.Nanosecond)
			v := 1
			return v + 0, v + 1, v + 2, v + 3, v + 4, v + 5, v + 6, v + 7, v + 8
		})
		report("source", g.Call(nil))
		params = append(params, reflect.SliceOf(typ))
		f = reflect.MakeFunc(reflect.FuncOf(params, []reflect.Type{typ}, true), func(in []reflect.Value) []reflect.Value {
			time.Sleep(time.Nanosecond)
			sum := Number(0)
			for _, v := range in[:10] {
				sum += Number(v.Int())
			}
			for i := 0; i < in[10].Len(); i++ {
				sum += Number(in[10].Index(i).Int())
			}
			return []reflect.Value{reflect.ValueOf(sum)}
		})
		report("variadic", f.Call(append(args, reflect.ValueOf(Number(10)), reflect.ValueOf(Number(11)))))
		report("slice", f.CallSlice(append(args, reflect.ValueOf([]Number{10, 11}))))
		w := reflect.ValueOf(Wide(40))
		report("method", w.MethodByName("Many").Call(ints))
		m, _ := w.Type().MethodByName("Many")
		report("method function", m.Func.Call(append([]reflect.Value{w}, ints...)))
		full := m.Func.Interface().(func(Wide, int, int, int, int, int, int, int, int, int, int) (int, int, int, int, int, int, int, int, int))
		report("typed method function", reflect.ValueOf(full).Call(append([]reflect.Value{w}, ints...)))
		dynamic := reflect.ValueOf(DynamicWide(40))
		dm, _ := dynamic.Type().MethodByName("Many")
		report("dynamic method function", dm.Func.Call(append([]reflect.Value{dynamic}, ints...)))
		typed := w.MethodByName("Many").Interface().(func(a0 int, a1 int, a2 int, a3 int, a4 int, a5 int, a6 int, a7 int, a8 int, a9 int) (int, int, int, int, int, int, int, int, int))
		report("typed method", reflect.ValueOf(typed).Call(ints))
	})
}
