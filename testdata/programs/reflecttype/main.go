package main

import (
	"fmt"
	"reflect"
	"strings"
)

type Leaf struct{ Match int }
type Left struct{ Leaf }
type Right struct{ Leaf }
type Deeper struct{ Other int }
type Deep struct{ Deeper }
type Ambiguous struct {
	Left
	Right
	Deep
}
type Cycle struct {
	*Cycle
	Value int
}
type Methods struct{}

func (Methods) First()  {}
func (Methods) Second() {}

func panicText(fn func()) {
	defer func() { fmt.Println("panic:", recover()) }()
	fn()
}

func main() {
	for _, value := range []any{Leaf{7}, Left{}, Ambiguous{}, Cycle{}, struct{}{}, struct{ Match, Other int }{}} {
		typ := reflect.TypeOf(value)
		var calls []string
		match := func(name string) bool {
			calls = append(calls, name)
			return name == "Match" || name == "Other" || name == "Value"
		}
		field, ok := typ.FieldByNameFunc(match)
		fmt.Println(ok, field.Name, field.Index, strings.Join(calls, ","))
		fmt.Println(reflect.ValueOf(value).FieldByNameFunc(func(name string) bool { return name == "Match" || name == "Other" || name == "Value" }).IsValid())
		for f := range typ.Fields() {
			fmt.Println(f.Name)
			break
		}
	}
	for method := range reflect.TypeFor[Methods]().Methods() {
		fmt.Println(method.Name)
		break
	}
	for method := range reflect.TypeFor[interface {
		First()
		Second()
	}]().Methods() {
		fmt.Println(method.Name)
	}
	fn := reflect.TypeOf(func(int, string) (bool, error) { return false, nil })
	for in := range fn.Ins() {
		fmt.Println(in)
	}
	for out := range fn.Outs() {
		fmt.Println(out)
		break
	}
	for _, fn := range []func(){
		func() { reflect.TypeFor[int]().Fields() },
		func() { reflect.TypeFor[int]().Ins() },
		func() { reflect.TypeFor[int]().Outs() },
		func() { reflect.TypeFor[int]().FieldByNameFunc(func(string) bool { return true }) },
		func() { reflect.TypeFor[string]().OverflowInt(0) },
		func() { reflect.TypeFor[string]().OverflowUint(0) },
		func() { reflect.TypeFor[string]().OverflowFloat(0) },
		func() { reflect.TypeFor[string]().OverflowComplex(0) },
	} {
		panicText(fn)
	}
	fmt.Println(reflect.TypeFor[complex64]().OverflowComplex(1e100i), reflect.TypeFor[complex128]().OverflowComplex(1e100i))
}
