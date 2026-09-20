package main

import (
	"fmt"
	"reflect"
	"strings"

	"Example.com/reflectcontracts/Upper"
)

type record struct {
	Number int
	Array  [2]int
	Ref    *int
}

type methods struct{}

func (methods) A() int { return 1 }
func (methods) B() int { return 2 }

type onlyB interface{ B() int }

func panicsWith(text string, f func()) (matched bool) {
	defer func() {
		if value := recover(); value != nil {
			matched = strings.Contains(fmt.Sprint(value), text)
		}
	}()
	f()
	return false
}

type recursive[T any] interface {
	Next() recursive[T]
}

func recursiveMetadata[T any]() {
	t := reflect.TypeFor[recursive[T]]()
	fmt.Println("recursive method type", t.Method(0).Type.Out(0) == t)
}

func main() {
	recursiveMetadata[int]()
	for _, input := range [][]int{nil, {}, {1, 2}} {
		v := reflect.ValueOf(input)
		a, b := v.Slice(0, 0), v.Slice3(0, 0, 0)
		fmt.Println("empty slice", a.IsNil(), a.Len(), a.Cap(), b.IsNil(), b.Len(), b.Cap())
	}
	array := [3]int{1, 2, 3}
	for _, v := range []reflect.Value{reflect.ValueOf(&array).Elem(), reflect.ValueOf(array[:]), reflect.ValueOf("abc")} {
		message := "reflect.Value.Slice: slice index out of bounds"
		if v.Kind() == reflect.String {
			message = "reflect.Value.Slice: string slice index out of bounds"
		}
		for _, bounds := range [][2]int{{-1, 0}, {2, 1}, {0, 4}} {
			fmt.Println("slice bounds", panicsWith(message, func() { v.Slice(bounds[0], bounds[1]) }))
		}
	}
	for _, v := range []reflect.Value{reflect.ValueOf(&array).Elem(), reflect.ValueOf(array[:])} {
		for _, bounds := range [][3]int{{-1, 0, 0}, {2, 1, 3}, {0, 2, 1}, {0, 1, 4}} {
			fmt.Println("slice3 bounds", panicsWith("reflect.Value.Slice3: slice index out of bounds", func() { v.Slice3(bounds[0], bounds[1], bounds[2]) }))
		}
	}

	var sealed Upper.Sealed = Upper.Value{}
	fmt.Println("private calls", Upper.Call(sealed))
	fmt.Println("exported method count", reflect.TypeOf(Upper.Value{}).NumMethod())
	sealedType := reflect.TypeFor[Upper.Sealed]()
	for i := 0; i < sealedType.NumMethod(); i++ {
		method := sealedType.Method(i)
		fmt.Println("interface method metadata", i, method.Name, method.PkgPath, method.IsExported(), method.Type, method.Func.IsValid())
	}
	privateMethod := reflect.ValueOf(&sealed).Elem().Method(1)
	fmt.Println("private method value", privateMethod.Kind(), privateMethod.Type(), privateMethod.CanInterface(), privateMethod.IsNil())
	fmt.Println("private method interface", panicsWith("unexported method", func() { privateMethod.Interface() }))
	fmt.Println("private method call", panicsWith("unexported method", func() { privateMethod.Call(nil) }))
	fmt.Println("exported method index", reflect.ValueOf(&sealed).Elem().Method(0).Call(nil)[0].Int())
	bytesOwner := struct{ data []byte }{[]byte{1, 2, 3}}
	bytesValue := reflect.ValueOf(&bytesOwner).Elem().Field(0)
	fmt.Println("private bytes", bytesValue.CanInterface(), bytesValue.Slice(0, 2).CanInterface(), bytesValue.Slice3(0, 2, 3).CanInterface())
	bytesValue.Bytes()[0] = 9
	fmt.Println("private byte alias", bytesOwner.data[0])
	for _, makeSlice := range []func(){
		func() { reflect.ValueOf([2]int{}).Slice(0, 1) },
		func() { reflect.ValueOf([2]int{}).Slice3(0, 1, 2) },
	} {
		fmt.Println("array address required", panicsWith("unaddressable array", makeSlice))
	}
	var limited onlyB = methods{}
	methodValue := reflect.ValueOf(&limited).Elem()
	fmt.Println("interface methods", methodValue.NumMethod(), methodValue.Method(0).Call(nil)[0].Int())
	fmt.Println("interface lookup", methodValue.MethodByName("A").IsValid(), methodValue.MethodByName("B").Call(nil)[0].Int())
	integerResult := reflect.TypeOf(func() int { return 0 })
	for _, result := range [][]reflect.Value{
		nil,
		{reflect.ValueOf(1), reflect.ValueOf(2)},
	} {
		fmt.Println("result count", panicsWith("wrong return count", func() {
			reflect.MakeFunc(integerResult, func([]reflect.Value) []reflect.Value {
				return result
			}).Interface().(func() int)()
		}))
	}
	for _, result := range []struct {
		value reflect.Value
		text  string
	}{
		{reflect.Value{}, "returned zero Value"},
		{reflect.ValueOf(struct{ hidden int }{3}).Field(0), "returned value obtained from unexported field"},
	} {
		fmt.Println("invalid result", panicsWith(result.text, func() {
			reflect.MakeFunc(integerResult, func([]reflect.Value) []reflect.Value {
				return []reflect.Value{result.value}
			}).Interface().(func() int)()
		}))
	}
	interfaceResult := reflect.TypeOf(func() any { return nil })
	nilResult := reflect.MakeFunc(interfaceResult, func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.Zero(reflect.TypeFor[any]())}
	}).Interface().(func() any)()
	fmt.Println("nil interface result", nilResult == nil)

	for _, source := range [][]int{nil, {}, {4}, {4, 5}} {
		value := reflect.ValueOf(source)
		for size := 0; size <= 3; size++ {
			target := reflect.ArrayOf(size, reflect.TypeFor[int]())
			fmt.Println("array conversion", len(source), size, value.Type().ConvertibleTo(target), value.CanConvert(target))
			if len(source) >= size {
				converted := value.Convert(target)
				fmt.Println(converted.CanAddr(), converted.Len(), converted.Interface())
			} else {
				fmt.Println("short slice", panicsWith("cannot convert slice with length", func() { value.Convert(target) }))
			}
		}
	}
	fmt.Println("different element", reflect.ValueOf([]int{1}).CanConvert(reflect.TypeFor[[1]int64]()))
	shared := 7
	source := []record{{Number: 1, Array: [2]int{2, 3}, Ref: &shared}}
	converted := reflect.ValueOf(source).Convert(reflect.TypeFor[[1]record]()).Interface().([1]record)
	source[0].Number = 99
	source[0].Array[0] = 88
	shared = 8
	fmt.Println("value copy", converted[0].Number, converted[0].Array, *converted[0].Ref)
}
