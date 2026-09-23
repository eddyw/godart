package main

import (
	"fmt"
	"math"
	"reflect"
)

type named float32

var zero float32
var global float32

func identity[T ~float32](value T) T   { return value }
func negate[T ~float32](value T) T     { return -value }
func bits(label string, value float32) { fmt.Printf("%s %08x\n", label, math.Float32bits(value)) }

type echoer interface{ Echo(float32) float32 }
type echo struct{}

func (echo) Echo(value float32) float32     { return value }
func pair(value float32) (float32, float32) { return value, value }
func variadic(values ...float32) float32    { return values[0] }
func genericStorage[T ~float32](value T) T {
	values := []T{value}
	array := [1]T{values[0]}
	copy := array
	return copy[0]
}
func genericZero[T ~complex64 | ~complex128]() T { return make([]T, 1)[0] }
func main() {
	var GoFloat32 float32
	var GoComplex32 complex64
	bits("reserved-float32", GoFloat32)
	bits("reserved-complex32", real(GoComplex32))
	positiveZero := math.Float32frombits(0)
	negativeZero := math.Float32frombits(0x80000000)
	bits("min-zeros", min(positiveZero, negativeZero))
	bits("max-zeros", max(negativeZero, positiveZero))
	bits("generic-complex-zero", real(genericZero[complex64]()))
	fmt.Println("generic-complex128-zero", real(genericZero[complex128]()))
	var complexZero complex64
	bits("complex-zero", real(complexZero))
	bits("complex-slice-zero", real(make([]complex64, 1)[0]))
	bits("carrier-collision", float32(math.Float64frombits(0x7ff8f32000000003)))

	for _, raw := range []uint32{0x7f800001, 0x7fbfffff, 0xff800001, 0xffbfffff, 0x7fc00001, 0x7fffffff, 0xffc00001, 0, 0x80000000, 1, 0x3f800000} {
		value := math.Float32frombits(raw)
		global = value
		bits("global", global)
		bits("named", float32(named(value)))
		bits("generic", identity(value))
		bits("generic-storage", genericStorage(value))
		bits("interface-call", echoer(echo{}).Echo(value))
		first, second := pair(value)
		bits("pair-first", first)
		bits("pair-second", second)
		bits("variadic", variadic(value))
		ch := make(chan float32, 1)
		ch <- value
		bits("channel", <-ch)
		reflected := reflect.ValueOf(identity[float32]).Call([]reflect.Value{reflect.ValueOf(value)})
		bits("reflected-call", reflected[0].Interface().(float32))
		fmt.Println("equal-self", value == value, any(value) == any(value))
		bits("neg", -value)
		bits("generic-neg", negate(value))
		bits("add", value+zero)
		bits("sub", value-zero)
		literal := []float32{value}
		bits("literal", literal[0])
		storage := make([]float32, 1)
		storage[0] = value
		bits("slice", storage[0])
		copied := make([]float32, 1)
		copy(copied, literal)
		bits("copy", copied[0])
		appended := append([]float32(nil), value)
		bits("append", appended[0])
		array := [1]float32{value}
		bits("array", array[0])
		v := reflect.ValueOf(named(value)).Convert(reflect.TypeFor[float32]())
		bits("reflect-convert", v.Interface().(float32))
		fmt.Printf("widen %016x\n", math.Float64bits(float64(value)))
		fmt.Printf("reflect-float %016x\n", math.Float64bits(v.Float()))
		bits("roundtrip", float32(float64(value)))
		var assigned float32
		reflect.ValueOf(&assigned).Elem().SetFloat(float64(value))
		bits("reflect-set", assigned)
		z := complex(value, value)
		bits("complex-real", real(z))
		bits("complex-neg", real(-z))
		fmt.Printf("complex-widen %016x\n", math.Float64bits(real(complex128(z))))
	}
}
