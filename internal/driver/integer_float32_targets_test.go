package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestIntegerFloat32Targets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/integerfloat32\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
	"math"
	"reflect"
)

type signed int64
type unsigned uint64

func convert[T ~int64 | ~uint64](v T) float32 { return float32(v) }

func reflected(v any) uint32 {
	return math.Float32bits(reflect.ValueOf(v).Convert(reflect.TypeOf(float32(0))).Interface().(float32))
}

func main() {
	values := []uint64{0, 1, 2, (1 << 24) - 1, 1 << 24, (1 << 24) + 1, math.MaxInt64, 1 << 63, math.MaxUint64, 0x8234508000000001, 1<<56 + 1<<32 + 1}
	for exponent := 24; exponent < 64; exponent++ {
		half := uint64(1) << (exponent - 24)
		midpoint := uint64(1)<<exponent + half
		values = append(values, midpoint-1, midpoint, midpoint+1)
		if exponent < 63 {
			carry := uint64(1)<<(exponent+1) - half
			values = append(values, carry-1, carry, carry+1)
		}
	}
	for _, v := range values {
		fmt.Printf("u %016x %08x %08x %08x %016x\n", v, math.Float32bits(float32(v)), math.Float32bits(convert(v)), reflected(v), math.Float64bits(float64(v)))
		fmt.Printf("n %016x %08x %08x %08x\n", v, math.Float32bits(float32(unsigned(v))), math.Float32bits(convert(unsigned(v))), reflected(unsigned(v)))
		for _, s := range []int64{int64(v), -int64(v)} {
			fmt.Printf("i %016x %08x %08x %08x %016x\n", uint64(s), math.Float32bits(float32(s)), math.Float32bits(convert(s)), reflected(s), math.Float64bits(float64(s)))
			fmt.Printf("s %016x %08x %08x %08x\n", uint64(s), math.Float32bits(float32(signed(s))), math.Float32bits(convert(signed(s))), reflected(signed(s)))
		}
	}
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := programOutcome(t, dir, native)
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("integer conversion exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, want.Text)
			}
		})
	}
}
