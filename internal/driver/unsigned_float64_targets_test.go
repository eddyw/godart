package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestUnsignedFloat64Targets(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/unsignedfloat64\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "math"
 "reflect"
)

type unsigned uint64

func convert[T ~uint64](v T) float64 { return float64(v) }

func reflected(v any) uint64 {
 return math.Float64bits(reflect.ValueOf(v).Convert(reflect.TypeOf(float64(0))).Interface().(float64))
}

func main() {
 values := []uint64{0, 1, 2, 1<<53-1, 1<<53, 1<<53+1, math.MaxInt64, 1<<63, math.MaxUint64}
 for i:=uint64(0); i<32; i++ {
  midpoint := uint64(1)<<63 + i<<11 + 1<<10
  values=append(values,midpoint-1,midpoint,midpoint+1)
 }
 values=append(values,math.MaxUint64-1025,math.MaxUint64-1024,math.MaxUint64-1023)
 for _,v:=range values {
  fmt.Printf("u %016x %016x %016x %016x\n",v,math.Float64bits(float64(v)),math.Float64bits(convert(v)),reflected(v))
  fmt.Printf("n %016x %016x %016x %016x\n",v,math.Float64bits(float64(unsigned(v))),math.Float64bits(convert(unsigned(v))),reflected(unsigned(v)))
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
				t.Fatalf("unsigned conversion exit=%d\ngot:\n%s\nwant:\n%s", result.Exit, result.Text, want.Text)
			}
		})
	}
}
