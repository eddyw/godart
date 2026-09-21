package main

import (
	"fmt"
	"reflect"
	"sort"
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

func swapAliases() {
	shared := 7
	values := []record{{Number: 2, Array: [2]int{20, 21}, Ref: &shared}, {Number: 1, Array: [2]int{10, 11}, Ref: &shared}}
	first, second := &values[0], &values[1]
	field, nested := &values[0].Number, &values[0].Array[1]
	swap := reflect.Swapper(values)
	swap(0, 1)
	fmt.Println("swap aliases", first.Number, second.Number, *field, *nested, first.Ref == &shared)
	swap(0, 0)
	fmt.Println("same cell", first.Number, *nested)
	sort.Slice(values, func(i, j int) bool { return values[i].Number > values[j].Number })
	fmt.Println("sort aliases", first.Number, second.Number, *nested)
	sort.SliceStable(values, func(i, j int) bool { return values[i].Number < values[j].Number })
	fmt.Println("stable aliases", first.Number, second.Number, *nested)
	arrays := [][2]record{{{Number: 1}, {Number: 2}}, {{Number: 3}, {Number: 4}}}
	ap, inner := &arrays[0], &arrays[0][1]
	reflect.Swapper(arrays)(0, 1)
	fmt.Println("array aliases", ap[0].Number, inner.Number, arrays[1][1].Number)
	ints := []int{1, 2}
	ip := &ints[0]
	reflect.Swapper(ints)(0, 1)
	fmt.Println("scalar alias", *ip)
	dynamicType := reflect.StructOf([]reflect.StructField{{Name: "Nested", Type: reflect.ArrayOf(2, reflect.TypeFor[int]())}})
	dynamic := reflect.MakeSlice(reflect.SliceOf(dynamicType), 2, 2)
	dynamic.Index(0).Field(0).Index(1).SetInt(11)
	dynamic.Index(1).Field(0).Index(1).SetInt(22)
	dynamicPointer := dynamic.Index(0).Field(0).Index(1).Addr()
	reflect.Swapper(dynamic.Interface())(0, 1)
	fmt.Println("dynamic swap", dynamicPointer.Elem().Int(), dynamic.Index(1).Field(0).Index(1).Int())
	dynamicArray := reflect.MakeSlice(reflect.SliceOf(reflect.ArrayOf(2, dynamicType)), 2, 2)
	dynamicArray.Index(0).Index(1).Set(dynamic.Index(0))
	dynamicArray.Index(1).Index(1).Set(dynamic.Index(1))
	arrayPointer := dynamicArray.Index(0).Index(1).Field(0).Index(1).Addr()
	reflect.Swapper(dynamicArray.Interface())(0, 1)
	fmt.Println("dynamic array swap", arrayPointer.Elem().Int(), dynamicArray.Index(1).Index(1).Field(0).Index(1).Int())
}

func main() {
	copyAliases()
	copyArrayViews()
	genericAliases()
	nestedArrayAliases()
	swapAliases()
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

type V struct {
	A int
	B [2]int
}

func copyAliases() {
	for _, reflective := range []bool{false, true} {
		s := []V{{1, [2]int{10, 11}}, {2, [2]int{20, 21}}, {3, [2]int{30, 31}}}
		p, q := &s[1], &s[1].B[1]
		if reflective {
			reflect.Copy(reflect.ValueOf(s[1:]), reflect.ValueOf(s[:2]))
		} else {
			copy(s[1:], s[:2])
		}
		fmt.Println("copy", reflective, s, p.A, *q)
	}
	for _, reflective := range []bool{false, true} {
		s := []V{{1, [2]int{10, 11}}, {2, [2]int{20, 21}}}
		p, q := &s[0], &s[0].B[1]
		if reflective {
			reflect.ValueOf(s).Clear()
		} else {
			clear(s)
		}
		fmt.Println("clear", reflective, s, p.A, *q)
	}
}

func copyArrayViews() {
	for _, reflective := range []bool{false, true} {
		values := []V{{A: 1}, {A: 2}, {A: 3}, {A: 4}}
		dst, src := (*[3]V)(values[1:]), (*[3]V)(values[:3])
		held := &values[2]
		if reflective {
			reflect.Copy(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
		} else {
			copy(dst[:], src[:])
		}
		fmt.Println("array view copy", reflective, values, held.A)
	}
	for _, reflective := range []bool{false, true} {
		values := []int{1, 2, 3, 4}
		dst, src := (*[3]int)(values[1:]), (*[3]int)(values[:3])
		if reflective {
			reflect.Copy(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
		} else {
			copy(dst[:], src[:])
		}
		fmt.Println("scalar view copy", reflective, values)
	}
	values := []V{{A: 1}, {A: 2}, {A: 3}}
	copy(values[:2], values[1:])
	fmt.Println("forward overlap", values)
	held := &values[0]
	clearGeneric(values)
	fmt.Println("generic clear", held.A)
	for _, pair := range [][2]reflect.Value{
		{reflect.Value{}, reflect.ValueOf([]int{1})},
		{reflect.ValueOf([1]int{}), reflect.ValueOf([]int{1})},
		{reflect.ValueOf([]int{}), reflect.Value{}},
		{reflect.ValueOf([]int{}), reflect.ValueOf([]int64{})},
		{reflect.ValueOf(struct{ a []int }{[]int{1}}).Field(0), reflect.ValueOf([]int{1})},
		{reflect.ValueOf([]int{1}), reflect.ValueOf(struct{ a []int }{[]int{1}}).Field(0)},
	} {
		func() { defer func() { fmt.Println("copy error", recover()) }(); reflect.Copy(pair[0], pair[1]) }()
	}
}
func clearGeneric[T any](values []T) { clear(values) }

func copyGeneric[T any](dst, src []T)     { copy(dst, src) }
func appendGeneric[T any](s []T, x T) []T { return append(s, x) }
func genericAliases() {
	values := []V{{A: 1}, {A: 2}, {A: 3}}
	held := &values[1]
	copyGeneric(values[1:], values[:2])
	fmt.Println("generic copy", values, held.A)
	ints := []int{1, 2, 3}
	copyGeneric(ints[1:], ints[:2])
	clearGeneric(ints)
	fmt.Println("generic scalar", ints)
	original := []V{{A: 1}}
	old := &original[0]
	grown := appendGeneric(original, V{A: 2})
	grown[0].A = 3
	fmt.Println("generic grow", old.A, grown[0].A)
	available := make([]V, 1, 2)
	cell := &available[:2][1]
	available = appendGeneric(available, V{A: 4})
	fmt.Println("generic room", cell.A)
}

type Holder[T any] struct{ Values [2]T }

func cloneArray[T any](v [2]T) [2]T { return v }
func nestedArrayAliases() {
	scalars := []Holder[int]{{Values: [2]int{7, 8}}}
	sp := &scalars[0].Values[0]
	clearGeneric(scalars)
	fmt.Println("generic scalar array clear", *sp)
	values := [][2]V{{{A: 1}, {A: 2}}, {{A: 3}, {A: 4}}, {{A: 5}, {A: 6}}}
	held := &values[1][0]
	copy(values[1:], values[:2])
	fmt.Println("nested array copy", values, held.A)
	clear(values)
	fmt.Println("nested array clear", held.A)
	holders := []Holder[V]{{Values: [2]V{{A: 7}, {A: 8}}}}
	inner := &holders[0].Values[0]
	clearGeneric(holders)
	fmt.Println("generic array clear", inner.A)
	original := [2]V{{A: 1}, {A: 2}}
	cloned := cloneArray(original)
	cloned[0].A = 9
	fmt.Println("generic array copy", original[0].A, cloned[0].A)
}
