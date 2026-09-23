package main

import (
	"fmt"
	"math"
	"reflect"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func convert[T integer, F ~float32 | ~float64](v F) T { return T(v) }

func generic[F ~float32 | ~float64](v F) []any {
	return []any{convert[int](v), convert[int8](v), convert[int16](v), convert[int32](v), convert[int64](v), convert[uint](v), convert[uint8](v), convert[uint16](v), convert[uint32](v), convert[uint64](v), convert[uintptr](v)}
}

func direct64(v float64) []any {
	return []any{int(v), int8(v), int16(v), int32(v), int64(v), uint(v), uint8(v), uint16(v), uint32(v), uint64(v), uintptr(v)}
}

func direct32(v float32) []any {
	return []any{int(v), int8(v), int16(v), int32(v), int64(v), uint(v), uint8(v), uint16(v), uint32(v), uint64(v), uintptr(v)}
}

func main() {
	types := []any{int(0), int8(0), int16(0), int32(0), int64(0), uint(0), uint8(0), uint16(0), uint32(0), uint64(0), uintptr(0)}
	values := []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.MaxFloat64, -math.MaxFloat64, 0x1p64, -0x1p64, 0, 1.5, -1.5}
	count := 0
	for _, f := range values {
		for i, value := range []any{f, float32(f)} {
			var direct, result []any
			if i == 0 {
				direct, result = direct64(f), generic(f)
			} else {
				direct, result = direct32(float32(f)), generic(float32(f))
			}
			for j, target := range types {
				typ := reflect.TypeOf(target)
				reflected := reflect.ValueOf(value).Convert(typ).Interface()
				for _, v := range []any{direct[j], result[j], reflected} {
					if reflect.TypeOf(v) != typ || len(fmt.Sprint(v)) == 0 {
						panic("conversion has no integer result")
					}
					count++
				}
			}
		}
	}
	fmt.Println("completed conversions:", count)
}
