package main

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"slices"
)

type number float64

type holder struct{ Value float64 }

func identity[T any](value T) T            { return value }
func sliceOf[T any](value T) []T           { return []T{value} }
func arrayOf[T any](value T) [1]T          { return [1]T{value} }
func first(values ...float64) float64      { return values[0] }
func pair(a, b float64) (float64, float64) { return b, a }

func expect(name string, value float64, bits uint64) {
	if got := math.Float64bits(value); got != bits {
		panic(fmt.Sprintf("%s: got %016x, want %016x", name, got, bits))
	}
}

func exercise(bits uint64) {
	value := math.Float64frombits(bits)
	expect("direct", value, bits)
	expect("identity", identity(value), bits)
	values := []float64{value}
	array := [1]float64{value}
	expect("slice literal", values[0], bits)
	expect("array literal", array[0], bits)
	expect("indexed array", [3]float64{2: value}[2], bits)
	expect("indexed slice", []float64{2: value}[2], bits)
	expect("array copy", identity(array)[0], bits)
	expect("slice to array", [1]float64(values)[0], bits)
	expect("slice to array pointer", (*[1]float64)(values)[0], bits)
	var zero [1]float64
	zero[0] = value
	expect("array assignment", zero[0], bits)
	record := holder{value}
	expect("struct", record.Value, bits)
	expect("map", map[int]float64{0: value}[0], bits)
	var boxed any = value
	expect("interface", boxed.(float64), bits)
	queue := make(chan float64, 1)
	queue <- value
	expect("channel", <-queue, bits)
	go func(v float64) { queue <- v }(value)
	expect("goroutine", <-queue, bits)
	runtime.Gosched()
	expect("suspension", value, bits)
	made := make([]float64, 1)
	made[0] = value
	expect("slice assignment", made[0], bits)
	copied := make([]float64, 1)
	copy(copied, made)
	expect("copy", copied[0], bits)
	expect("append element", append([]float64(nil), value)[0], bits)
	expect("append slice", append([]float64(nil), made...)[0], bits)
	expect("clone", slices.Clone(made)[0], bits)
	expect("generic slice", sliceOf(value)[0], bits)
	expect("generic array", arrayOf(value)[0], bits)
	expect("variadic", first(value), bits)
	call := identity[float64]
	expect("function", call(value), bits)
	input := []reflect.Value{reflect.ValueOf(value)}
	expect("reflect.Float", input[0].Float(), bits)
	expect("reflect.Call", reflect.ValueOf(call).Call(input)[0].Float(), bits)
	expect("reflect.CallSlice", reflect.ValueOf(first).CallSlice([]reflect.Value{reflect.ValueOf(values)})[0].Float(), bits)
	result := reflect.ValueOf(pair).Call([]reflect.Value{input[0], input[0]})
	expect("multiple result 0", result[0].Float(), bits)
	expect("multiple result 1", result[1].Float(), bits)
	reflected := reflect.MakeFunc(reflect.TypeOf(call), func(args []reflect.Value) []reflect.Value { return args }).Interface().(func(float64) float64)
	expect("reflect.MakeFunc", reflected(value), bits)
	var set float64
	reflect.ValueOf(&set).Elem().SetFloat(value)
	expect("reflect.SetFloat", set, bits)
	converted := reflect.ValueOf(number(value)).Convert(reflect.TypeOf(value)).Interface().(float64)
	expect("reflect.Convert", converted, bits)
	complexValue := complex(value, value)
	expect("real", real(complexValue), bits)
	expect("imag", imag(complexValue), bits)
	complexValue = reflect.ValueOf(complexValue).Complex()
	expect("reflect.Complex real", real(complexValue), bits)
	expect("reflect.Complex imag", imag(complexValue), bits)
	expect("negate", -value, bits^0x8000000000000000)
}

func main() {
	for repeat := 0; repeat < 32; repeat++ {
		for _, bits := range []uint64{
			0, 1, 0x8000000000000000, 0x8000000000000001,
			0x7fefffffffffffff, 0x7ff0000000000000, 0xfff0000000000000,
			0x7ff8000000000000, 0x7ff8000000000001,
			0x7ff0000000000001, 0x7ff123456789abcd,
			0xfff0000000000001, 0xfff123456789abcd, 0xffffffffffffffff,
		} {
			exercise(bits)
		}
	}
	fmt.Println("float64 bits preserved")
}
