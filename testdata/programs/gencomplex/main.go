package main

import (
	"fmt"
	"math"
	"reflect"
)

type number interface {
	~int8 | ~uint64 | ~float32 | ~complex64 | ~complex128
}
type complexNumber interface{ ~complex64 | ~complex128 }
type realOrComplex interface{ ~float32 | ~complex64 }
type C64 complex64

type C128 complex128

func operations[T number](a, b T) {
	fmt.Println(a+b, a-b, a*b, a/b, -a)
	a++
	b--
	fmt.Println(a, b)
	a += b
	a -= b
	a *= b
	a /= b
	fmt.Println(a)
}

func convert[T, U complexNumber](a T) U      { return U(a) }
func constants[T complexNumber]()            { fmt.Println(T(0), T(3.14), T(-2i), T(3.14+2i)) }
func fraction[T realOrComplex]() T           { return 1.23456789 }
func realConstant[T ~float32 | ~float64]() T { return 1.23456789 }

func main() {
	operations[int8](120, 5)
	operations[uint64](math.MaxUint64, 2)
	operations[float32](1.2345678, 2.3456789)
	operations[complex64](1.2345678+2.3456789i, 2+3i)
	operations[complex128](1.25+2.5i, 2)
	operations[C64](1+2i, 3+4i)
	constants[complex64]()
	constants[complex128]()
	fmt.Println(fraction[float32](), fraction[complex64]())
	fmt.Println(realConstant[float32](), realConstant[float64]())
	value := C128(1.23456789 + 2.34567891i)
	fmt.Println(convert[C128, C64](value), convert[C64, C128](C64(value)))
	fmt.Println(reflect.TypeFor[C128]().ConvertibleTo(reflect.TypeFor[C64]()), reflect.TypeFor[C128]().ConvertibleTo(reflect.TypeFor[float64]()), reflect.TypeFor[float64]().ConvertibleTo(reflect.TypeFor[C64]()))
	fmt.Println(reflect.ValueOf(value).Convert(reflect.TypeFor[C64]()).Interface())
}
